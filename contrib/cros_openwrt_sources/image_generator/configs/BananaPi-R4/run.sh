#!/bin/bash
# Copyright 2025 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

set -e

DEVICE_CONFIG_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
OPENWRT_DIR=$(realpath -e "${DEVICE_CONFIG_DIR}/../..")

cd "${OPENWRT_DIR}"

MTK_AUTOBUILD_DIR=${OPENWRT_DIR}/../mtk-openwrt-feeds/autobuild/unified

echo "Prepare MTK folder with feed and config updates"
cp "${DEVICE_CONFIG_DIR}/feed_revision" "${MTK_AUTOBUILD_DIR}"
cp "${DEVICE_CONFIG_DIR}/defconfig" "${MTK_AUTOBUILD_DIR}/filogic/mac80211/mt7988_rfb/mt7996/24.10/"
cp -f "${DEVICE_CONFIG_DIR}/0003-hostapd-package-makefile-ucode-files.patch" "${MTK_AUTOBUILD_DIR}/filogic/mac80211/24.10/patches-base/0003-hostapd-package-makefile-ucode-files.patch"

echo "Start autobuild from MTK"
bash ../mtk-openwrt-feeds/autobuild/unified/autobuild.sh filogic-mac80211-mt7988_rfb-mt7996 log_file=logs/make_bpir4.log
