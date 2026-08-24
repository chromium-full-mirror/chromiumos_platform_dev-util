#!/bin/bash
# Copyright 2025 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

set -e

DEVICE_CONFIG_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
OPENWRT_DIR=$(realpath -e "${DEVICE_CONFIG_DIR}/../..")
MTK_DIR=${OPENWRT_DIR}/../mtk-openwrt-feeds
OPENWRT_VERSION=25.12
MTK_AUTOBUILD_DIR=${MTK_DIR}/autobuild/unified
AUTOBUILD_VARIANT_DIR="${MTK_AUTOBUILD_DIR}/filogic/mac80211/mt798x_rfb/wifi7_nic/${OPENWRT_VERSION}"

# 1. Apply Feed Git Patches to mtk-openwrt-feeds
echo "=== Applying feed patches to mtk-openwrt-feeds ==="
cd "${MTK_DIR}"
if [ -d "${DEVICE_CONFIG_DIR}/patches-feed" ]; then
	for patch_file in "${DEVICE_CONFIG_DIR}/patches-feed"/*.patch; do
		[ -f "$patch_file" ] || continue
		echo "  Applying feed patch: $(basename "$patch_file")"
		git apply --reverse --check "$patch_file" >/dev/null 2>&1 || git apply "$patch_file"
	done
fi

cd "${OPENWRT_DIR}"

# 2. Copy feed revisions and target defconfig
echo "=== Preparing MTK folder with feed and config updates ==="
cp "${DEVICE_CONFIG_DIR}/feed_revision" "${MTK_AUTOBUILD_DIR}/"
cp "${DEVICE_CONFIG_DIR}/defconfig" "${AUTOBUILD_VARIANT_DIR}/"

# 3. Copy Package Patches
echo "=== Copying package patches ==="

# uboot-mediatek
if [ -d "${DEVICE_CONFIG_DIR}/patches-package/uboot-mediatek" ]; then
	mkdir -p "${OPENWRT_DIR}/package/boot/uboot-mediatek/patches/"
	cp -v "${DEVICE_CONFIG_DIR}/patches-package/uboot-mediatek"/*.patch "${OPENWRT_DIR}/package/boot/uboot-mediatek/patches/"
fi

# mt76
if [ -d "${DEVICE_CONFIG_DIR}/patches-package/mt76" ]; then
	mkdir -p "${AUTOBUILD_VARIANT_DIR}/files/package/kernel/mt76/patches/"
	cp -v "${DEVICE_CONFIG_DIR}/patches-package/mt76"/*.patch "${AUTOBUILD_VARIANT_DIR}/files/package/kernel/mt76/patches/"
fi

# mac80211
if [ -d "${DEVICE_CONFIG_DIR}/patches-package/mac80211" ]; then
	mkdir -p "${AUTOBUILD_VARIANT_DIR}/files/package/kernel/mac80211/patches/subsys/"
	cp -v "${DEVICE_CONFIG_DIR}/patches-package/mac80211"/*.patch "${AUTOBUILD_VARIANT_DIR}/files/package/kernel/mac80211/patches/subsys/"
fi

# hostapd
if [ -d "${DEVICE_CONFIG_DIR}/patches-package/hostapd" ]; then
	mkdir -p "${AUTOBUILD_VARIANT_DIR}/files/package/network/services/hostapd/patches/"
	cp -v "${DEVICE_CONFIG_DIR}/patches-package/hostapd"/*.patch "${AUTOBUILD_VARIANT_DIR}/files/package/network/services/hostapd/patches/"
fi

# 4. Start Autobuild
echo "=== Starting autobuild from MTK ==="
bash "${MTK_AUTOBUILD_DIR}/autobuild.sh" filogic-mac80211-mt798x_rfb-wifi7_nic log_file=logs/make_bpir4
