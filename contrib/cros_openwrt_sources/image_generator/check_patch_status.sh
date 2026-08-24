#!/bin/bash
# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Patch Status & Upstream Sync Checker
# Inspects local patches against upstream repositories to identify merged, active, or conflicting patches.

set -e

SCRIPT_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
CONFIG_DIR="${1:-${SCRIPT_DIR}/configs/BananaPi-R4}"
[ -d "${CONFIG_DIR}" ] || CONFIG_DIR="${SCRIPT_DIR}/configs/${1}"

OPENWRT_DIR="${OPENWRT_DIR:-${SCRIPT_DIR}}"
MTK_DIR="${MTK_DIR:-$(realpath "${OPENWRT_DIR}/../mtk-openwrt-feeds" 2>/dev/null || true)}"

# Fallbacks for local execution outside container
if [ ! -d "${MTK_DIR}" ]; then
	for d in "${HOME}/mtk-openwrt-feeds" "${SCRIPT_DIR}/../../mtk-openwrt-feeds" "${SCRIPT_DIR}/../../../mtk-openwrt-feeds"; do
		if [ -d "$d" ]; then
			MTK_DIR="$(realpath "$d")"
			break
		fi
	done
fi

if [ ! -d "${OPENWRT_DIR}" ] || [ ! -f "${OPENWRT_DIR}/feeds.conf.default" ]; then
	for d in "${HOME}/openwrt" "${SCRIPT_DIR}/../../openwrt" "${SCRIPT_DIR}/../../../openwrt"; do
		if [ -d "$d" ]; then
			OPENWRT_DIR="$(realpath "$d")"
			break
		fi
	done
fi

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

if [ ! -d "${CONFIG_DIR}" ]; then
	echo -e "${RED}[ERROR] Config directory not found: ${CONFIG_DIR}${NC}"
	exit 1
fi

CONFIG_NAME="$(basename "${CONFIG_DIR}")"
echo -e "${BLUE}=== Checking Patch Upstream Status for ${CONFIG_NAME} ===${NC}"
[ -d "${MTK_DIR}" ] && echo -e "MTK Feed: ${MTK_DIR}"
[ -d "${OPENWRT_DIR}" ] && echo -e "OpenWrt:  ${OPENWRT_DIR}"
echo ""

TOTAL_CHECKED=0
TOTAL_RETIRED=0
TOTAL_ACTIVE=0
TOTAL_CONFLICT=0

# Helper to inspect patch status against a repository
check_git_patch() {
	local patch_file="$1"
	local target_repo="$2"
	local patch_name="$(basename "$patch_file")"
	TOTAL_CHECKED=$((TOTAL_CHECKED + 1))

	if [ ! -d "$target_repo" ]; then
		echo -e "  ${YELLOW}[UNKNOWN]${NC}  ${patch_name} (repo not found)"
		return
	fi

	cd "$target_repo"
	if git apply --reverse --check "$patch_file" >/dev/null 2>&1; then
		echo -e "  ${GREEN}[RETIRED]${NC}  ${patch_name} -> Already merged upstream (safe to delete)"
		TOTAL_RETIRED=$((TOTAL_RETIRED + 1))
	elif git apply --check "$patch_file" >/dev/null 2>&1; then
		echo -e "  ${CYAN}[ACTIVE]${NC}   ${patch_name} -> Pending upstream merge (applies cleanly)"
		TOTAL_ACTIVE=$((TOTAL_ACTIVE + 1))
	else
		echo -e "  ${RED}[CONFLICT]${NC} ${patch_name} -> Needs rebase against upstream"
		TOTAL_CONFLICT=$((TOTAL_CONFLICT + 1))
	fi
}

# 1. Feed Patches
FEED_PATCHES_DIR="${CONFIG_DIR}/patches-feed"
if [ -d "${FEED_PATCHES_DIR}" ]; then
	echo -e "${BLUE}[1/2] Feed Patches (patches-feed/)${NC}"
	for patch_file in "${FEED_PATCHES_DIR}"/*.patch; do
		[ -f "$patch_file" ] || continue
		check_git_patch "$patch_file" "${MTK_DIR}"
	done
	echo ""
fi

# 2. Package Patches
PKG_PATCHES_DIR="${CONFIG_DIR}/patches-package"
if [ -d "${PKG_PATCHES_DIR}" ]; then
	echo -e "${BLUE}[2/2] Package Patches (patches-package/)${NC}"
	for pkg_dir in "${PKG_PATCHES_DIR}"/*; do
		[ -d "$pkg_dir" ] || continue
		pkg_name="$(basename "$pkg_dir")"
		[ "$pkg_name" = "unused" ] && continue

		echo -e "  ${BLUE}Package: ${pkg_name}${NC}"
		for patch_file in "${pkg_dir}"/*.patch; do
			[ -f "$patch_file" ] || continue
			TOTAL_CHECKED=$((TOTAL_CHECKED + 1))
			patch_name="$(basename "$patch_file")"
			echo -e "    ${CYAN}[DOWNSTREAM]${NC} ${patch_name}"
			TOTAL_ACTIVE=$((TOTAL_ACTIVE + 1))
		done
	done
	echo ""
fi

# Summary
echo -e "${BLUE}=== Status Summary ===${NC}"
echo -e "Total checked:     ${TOTAL_CHECKED}"
echo -e "Active Downstream: ${CYAN}${TOTAL_ACTIVE}${NC}"
echo -e "Merged / Retired:  ${GREEN}${TOTAL_RETIRED}${NC}"
echo -e "Conflicted/Rebase: ${RED}${TOTAL_CONFLICT}${NC}"

if [ ${TOTAL_RETIRED} -gt 0 ]; then
	echo ""
	echo -e "${YELLOW}Notice: Some patches are already merged upstream and can be deleted from ${CONFIG_NAME}.${NC}"
fi
