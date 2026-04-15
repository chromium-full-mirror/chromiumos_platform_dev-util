#!/bin/bash
# Copyright 2025 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

set -e

DEVICE_CONFIG_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
OPENWRT_DIR=$(realpath -e "${DEVICE_CONFIG_DIR}/../..")
MTK_DIR=${OPENWRT_DIR}/../mtk-openwrt-feeds

cd "${MTK_DIR}"
cp "${DEVICE_CONFIG_DIR}/hostapd_sh.patch" "${MTK_DIR}"
git apply hostapd_sh.patch

cd "${OPENWRT_DIR}"

MTK_AUTOBUILD_DIR=${MTK_DIR}/autobuild/unified

echo "Prepare MTK folder with feed and config updates"
cp "${DEVICE_CONFIG_DIR}/feed_revision" "${MTK_AUTOBUILD_DIR}"
cp "${DEVICE_CONFIG_DIR}/defconfig" "${MTK_AUTOBUILD_DIR}/filogic/mac80211/mt7988_rfb/mt7996/24.10/"
cp "${DEVICE_CONFIG_DIR}/900-add-hw-id.patch" "${OPENWRT_DIR}/package/boot/uboot-mediatek/patches/"
cp "${DEVICE_CONFIG_DIR}/defconfig_common" "${MTK_AUTOBUILD_DIR}/filogic/mac80211/mt7988_rfb/24.10/defconfig"
# cp -f "${DEVICE_CONFIG_DIR}/0003-hostapd-package-makefile-ucode-files.patch" "${MTK_AUTOBUILD_DIR}/filogic/mac80211/24.10/patches-base/0003-hostapd-package-makefile-ucode-files.patch"

cp "${DEVICE_CONFIG_DIR}/9998-hostapd-add_require_eht.patch" "${MTK_AUTOBUILD_DIR}/filogic/mac80211/24.10/files/package/network/services/hostapd/patches/"
cp "${DEVICE_CONFIG_DIR}/9999-hostapd-wpa_supplicant_add_eht_members.patch" "${MTK_AUTOBUILD_DIR}/filogic/mac80211/24.10/files/package/network/services/hostapd/patches/"

echo "Start autobuild from MTK"
bash ../mtk-openwrt-feeds/autobuild/unified/autobuild.sh filogic-mac80211-mt7988_rfb-mt7996 log_file=logs/make_bpir4
