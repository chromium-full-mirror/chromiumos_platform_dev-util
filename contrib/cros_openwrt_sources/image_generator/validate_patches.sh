#!/bin/bash
# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Pre-flight Patch Dry-Run Validator for cros_openwrt
# Validates both feed patches (git apply --check) and package build patches (diff syntax & headers)

set -e

SCRIPT_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
CONFIG_DIR="${1:-${SCRIPT_DIR}/configs/BananaPi-R4}"
[ -d "${CONFIG_DIR}" ] || CONFIG_DIR="${SCRIPT_DIR}/configs/${1}"

OPENWRT_DIR="${OPENWRT_DIR:-${SCRIPT_DIR}}"
MTK_DIR="${MTK_DIR:-$(realpath "${OPENWRT_DIR}/../mtk-openwrt-feeds" 2>/dev/null || true)}"

# Add OpenWrt host tools (zstd, patch, fakeroot) to PATH if present
for p in "${OPENWRT_DIR}/staging_dir/host/bin" "/build/openwrt/staging_dir/host/bin"; do
	[ -d "$p" ] && export PATH="$p:$PATH"
done

# Fallbacks for local execution outside container
if [ ! -d "${MTK_DIR}" ]; then
	for d in "${HOME}/mtk-openwrt-feeds" "${SCRIPT_DIR}/../../mtk-openwrt-feeds" "${SCRIPT_DIR}/../../../mtk-openwrt-feeds"; do
		if [ -d "$d" ]; then
			MTK_DIR="$(realpath "$d")"
			break
		fi
	done
fi

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

if [ ! -d "${CONFIG_DIR}" ]; then
	echo -e "${RED}[ERROR] Config directory not found: ${CONFIG_DIR}${NC}"
	exit 1
fi

CONFIG_NAME="$(basename "${CONFIG_DIR}")"
echo -e "${BLUE}=== Validating Patches for ${CONFIG_NAME} ===${NC}"
echo -e "Config:   ${CONFIG_DIR}"
[ -d "${MTK_DIR}" ] && echo -e "MTK Feed: ${MTK_DIR}"
[ -d "${OPENWRT_DIR}" ] && echo -e "OpenWrt:  ${OPENWRT_DIR}"
echo ""

TOTAL_CHECKED=0
TOTAL_PASSED=0
TOTAL_FAILED=0

# 1. Validate Feed Patches (git apply --check against mtk-openwrt-feeds)
FEED_PATCHES_DIR="${CONFIG_DIR}/patches-feed"
if [ -d "${FEED_PATCHES_DIR}" ]; then
	echo -e "${BLUE}[1/2] Checking Feed Git Patches (patches-feed/)...${NC}"
	if [ -d "${MTK_DIR}" ]; then
		cd "${MTK_DIR}"
		for patch_file in "${FEED_PATCHES_DIR}"/*.patch; do
			[ -f "$patch_file" ] || continue
			TOTAL_CHECKED=$((TOTAL_CHECKED + 1))
			patch_name="$(basename "$patch_file")"

			if git apply --reverse --check "$patch_file" >/dev/null 2>&1; then
				echo -e "  ${GREEN}[PASS] (applied in tree)${NC} ${patch_name}"
				TOTAL_PASSED=$((TOTAL_PASSED + 1))
			elif git apply --check "$patch_file" >/dev/null 2>&1; then
				echo -e "  ${GREEN}[PASS] (clean dry-run)${NC}   ${patch_name}"
				TOTAL_PASSED=$((TOTAL_PASSED + 1))
			else
				echo -e "  ${RED}[FAIL]${NC} ${patch_name}"
				git apply --check "$patch_file" 2>&1 | sed 's/^/         /'
				TOTAL_FAILED=$((TOTAL_FAILED + 1))
			fi
		done
	else
		echo -e "  ${YELLOW}[SKIP] mtk-openwrt-feeds directory not found${NC}"
	fi
	echo ""
fi

# 2. Validate Package Patches (diff structure and dry-run syntax)
PKG_PATCHES_DIR="${CONFIG_DIR}/patches-package"
if [ -d "${PKG_PATCHES_DIR}" ]; then
	echo -e "${BLUE}[2/2] Checking Package Patches (patches-package/)...${NC}"
	for pkg_dir in "${PKG_PATCHES_DIR}"/*; do
		[ -d "$pkg_dir" ] || continue
		pkg_name="$(basename "$pkg_dir")"
		[ "$pkg_name" = "unused" ] && continue

		echo -e "  ${BLUE}Package: ${pkg_name}${NC}"
		for patch_file in "${pkg_dir}"/*.patch; do
			[ -f "$patch_file" ] || continue
			TOTAL_CHECKED=$((TOTAL_CHECKED + 1))
			patch_name="$(basename "$patch_file")"

			# Deep unified diff hunk and syntax validation
			validation_output="$(python3 - "$patch_file" << 'PYEOF'
import sys

def validate_patch(filepath):
    with open(filepath, "r", errors="replace") as f:
        lines = f.readlines()

    in_hunk = False
    hunks_found = 0
    for idx, line in enumerate(lines):
        if line.startswith("--- ") or line.startswith("+++ "):
            in_hunk = False
            continue
        if line.startswith("@@ "):
            in_hunk = True
            hunks_found += 1
            continue
        if in_hunk:
            if line.startswith("-- ") or line.startswith("---"):
                in_hunk = False
                continue
            if line.startswith("diff --git") or line.startswith("Index:") or line.startswith("==="):
                in_hunk = False
                continue
            first_char = line[0] if len(line) > 0 else ""
            if first_char not in (" ", "+", "-", "\\", "\n", "\r"):
                return False, f"Line {idx+1}: Invalid hunk prefix {repr(first_char)}: {line.strip()}"

    if hunks_found == 0:
        return False, "No unified diff hunks (@@) found"
    return True, "OK"

ok, msg = validate_patch(sys.argv[1])
if not ok:
    print(msg)
    sys.exit(1)
PYEOF
)"
			if [ $? -eq 0 ]; then
				echo -e "    ${GREEN}[PASS]${NC} ${patch_name}"
				TOTAL_PASSED=$((TOTAL_PASSED + 1))
			else
				echo -e "    ${RED}[FAIL] (${validation_output})${NC} ${patch_name}"
				TOTAL_FAILED=$((TOTAL_FAILED + 1))
			fi
		done
	done
	echo ""
fi

# 3. Sequential Package Patch Dry-Run (Extracts and applies all patches sequentially)
DL_DIR="${OPENWRT_DIR}/dl"
if [ ! -d "${DL_DIR}" ]; then
	for d in "${SCRIPT_DIR}/dl" "${HOME}/openwrt/dl" "/build/openwrt/dl"; do
		if [ -d "$d" ]; then
			DL_DIR="$d"
			break
		fi
	done
fi

if [ -d "${DL_DIR}" ] && command -v patch >/dev/null 2>&1; then
	echo -e "${BLUE}[3/3] Performing Sequential Package Patch Dry-Runs in Temp Directory...${NC}"
	TMP_VAL_DIR="$(mktemp -d /tmp/openwrt_val_XXXXXX)"
	trap 'rm -rf "${TMP_VAL_DIR}"' EXIT

	# 1. hostapd
	HOSTAPD_TAR="$(find "${DL_DIR}" -maxdepth 1 -name "hostapd-*.tar*" 2>/dev/null | head -n 1 || true)"
	if [ -n "${HOSTAPD_TAR}" ] && [ -f "${HOSTAPD_TAR}" ]; then
		mkdir -p "${TMP_VAL_DIR}/hostapd"
		tar -I zstd -xf "${HOSTAPD_TAR}" -C "${TMP_VAL_DIR}/hostapd" 2>/dev/null || tar -xf "${HOSTAPD_TAR}" -C "${TMP_VAL_DIR}/hostapd" 2>/dev/null || true
		SRC_DIR="$(ls -d "${TMP_VAL_DIR}/hostapd"/hostapd-* 2>/dev/null | head -n 1 || true)"
		if [ -n "${SRC_DIR}" ] && [ -d "${SRC_DIR}" ]; then
			cd "${SRC_DIR}"
			HOSTAPD_FAIL=0
			# Collect all patches into a combined directory for exact alphabetical application
			mkdir -p "${TMP_VAL_DIR}/hostapd_all_patches"
			VENDOR_PATCHES="${OPENWRT_DIR}/package/network/services/hostapd/patches"
			[ -d "${VENDOR_PATCHES}" ] && cp -f "${VENDOR_PATCHES}"/*.patch "${TMP_VAL_DIR}/hostapd_all_patches/" 2>/dev/null || true
			cp -f "${PKG_PATCHES_DIR}/hostapd"/*.patch "${TMP_VAL_DIR}/hostapd_all_patches/" 2>/dev/null || true
			for p in $(ls "${TMP_VAL_DIR}/hostapd_all_patches"/*.patch 2>/dev/null | sort); do
				[ -f "$p" ] || continue
				if ! patch -p1 --batch -N -s < "$p" >/dev/null 2>&1; then
					echo -e "    ${RED}[FAIL] (sequential dry-run failed)${NC} $(basename "$p")"
					TOTAL_FAILED=$((TOTAL_FAILED + 1))
					HOSTAPD_FAIL=1
					break
				fi
			done
			[ ${HOSTAPD_FAIL} -eq 0 ] && echo -e "  ${GREEN}[PASS] hostapd: all $(ls "${TMP_VAL_DIR}/hostapd_all_patches"/*.patch | wc -l) patches applied cleanly sequentially${NC}"
		fi
	fi

	# 2. mt76
	MT76_TAR="$(find "${DL_DIR}" -maxdepth 1 -name "mt76-*.tar*" 2>/dev/null | head -n 1 || true)"
	if [ -n "${MT76_TAR}" ] && [ -f "${MT76_TAR}" ]; then
		mkdir -p "${TMP_VAL_DIR}/mt76"
		tar -I zstd -xf "${MT76_TAR}" -C "${TMP_VAL_DIR}/mt76" 2>/dev/null || tar -xf "${MT76_TAR}" -C "${TMP_VAL_DIR}/mt76" 2>/dev/null || true
		SRC_DIR="$(ls -d "${TMP_VAL_DIR}/mt76"/mt76-* 2>/dev/null | head -n 1 || true)"
		if [ -n "${SRC_DIR}" ] && [ -d "${SRC_DIR}" ]; then
			cd "${SRC_DIR}"
			MT76_FAIL=0
			mkdir -p "${TMP_VAL_DIR}/mt76_all_patches"
			VENDOR_PATCHES="${OPENWRT_DIR}/package/kernel/mt76/patches"
			[ -d "${VENDOR_PATCHES}" ] && cp -f "${VENDOR_PATCHES}"/*.patch "${TMP_VAL_DIR}/mt76_all_patches/" 2>/dev/null || true
			cp -f "${PKG_PATCHES_DIR}/mt76"/*.patch "${TMP_VAL_DIR}/mt76_all_patches/" 2>/dev/null || true
			for p in $(ls "${TMP_VAL_DIR}/mt76_all_patches"/*.patch 2>/dev/null | sort); do
				[ -f "$p" ] || continue
				if ! patch -p1 --batch -N -s < "$p" >/dev/null 2>&1; then
					echo -e "    ${RED}[FAIL] (sequential dry-run failed)${NC} $(basename "$p")"
					TOTAL_FAILED=$((TOTAL_FAILED + 1))
					MT76_FAIL=1
					break
				fi
			done
			[ ${MT76_FAIL} -eq 0 ] && echo -e "  ${GREEN}[PASS] mt76: all $(ls "${TMP_VAL_DIR}/mt76_all_patches"/*.patch | wc -l) patches applied cleanly sequentially${NC}"
		fi
	fi

	# 3. uboot-mediatek
	UBOOT_TAR="$(find "${DL_DIR}" -maxdepth 1 -name "u-boot-*.tar*" 2>/dev/null | head -n 1 || true)"
	if [ -n "${UBOOT_TAR}" ] && [ -f "${UBOOT_TAR}" ]; then
		mkdir -p "${TMP_VAL_DIR}/uboot"
		tar -xf "${UBOOT_TAR}" -C "${TMP_VAL_DIR}/uboot" 2>/dev/null || true
		SRC_DIR="$(ls -d "${TMP_VAL_DIR}/uboot"/u-boot-* 2>/dev/null | head -n 1 || true)"
		if [ -n "${SRC_DIR}" ] && [ -d "${SRC_DIR}" ]; then
			cd "${SRC_DIR}"
			UBOOT_FAIL=0
			mkdir -p "${TMP_VAL_DIR}/uboot_all_patches"
			VENDOR_PATCHES="${OPENWRT_DIR}/package/boot/uboot-mediatek/patches"
			[ -d "${VENDOR_PATCHES}" ] && cp -f "${VENDOR_PATCHES}"/*.patch "${TMP_VAL_DIR}/uboot_all_patches/" 2>/dev/null || true
			cp -f "${PKG_PATCHES_DIR}/uboot-mediatek"/*.patch "${TMP_VAL_DIR}/uboot_all_patches/" 2>/dev/null || true
			for p in $(ls "${TMP_VAL_DIR}/uboot_all_patches"/*.patch 2>/dev/null | sort); do
				[ -f "$p" ] || continue
				if ! patch -p1 --batch -N -s < "$p" >/dev/null 2>&1; then
					echo -e "    ${RED}[FAIL] (sequential dry-run failed)${NC} $(basename "$p")"
					TOTAL_FAILED=$((TOTAL_FAILED + 1))
					UBOOT_FAIL=1
					break
				fi
			done
			[ ${UBOOT_FAIL} -eq 0 ] && echo -e "  ${GREEN}[PASS] uboot-mediatek: all $(ls "${TMP_VAL_DIR}/uboot_all_patches"/*.patch | wc -l) patches applied cleanly sequentially${NC}"
		fi
	fi
	rm -rf "${TMP_VAL_DIR}"
	echo ""
fi

# Summary
echo -e "${BLUE}=== Validation Summary ===${NC}"
echo -e "Total checked: ${TOTAL_CHECKED}"
echo -e "Passed:        ${GREEN}${TOTAL_PASSED}${NC}"
if [ ${TOTAL_FAILED} -gt 0 ]; then
	echo -e "Failed:        ${RED}${TOTAL_FAILED}${NC}"
	exit 1
else
	echo -e "Failed:        0"
	echo -e "${GREEN}All patches validated successfully!${NC}"
	exit 0
fi
