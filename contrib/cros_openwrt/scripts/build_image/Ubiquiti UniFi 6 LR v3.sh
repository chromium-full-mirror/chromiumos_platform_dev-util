#!/bin/bash
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# This script preforms the necessary commands to build the current standard
# image for openwrt deployment with Ubiquiti UniFi 6 LR v3 test APs.
# OpenWrt wiki: https://openwrt.org/toh/ubiquiti/unifi_6_lr
# OpenWrt device: https://openwrt.org/toh/hwdata/ubiquiti/ubiquiti_unifi_6_lr_v3

# Import build utility script.
set -e
SCRIPT_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
source "${SCRIPT_DIR}/lib/build.sh"

# Configure options for model.
image_profile='ubnt_unifi-6-lr-v3'
extra_image_name='standard-0.1.1'
sdk_url='https://downloads.openwrt.org/snapshots/targets/mediatek/mt7622/openwrt-sdk-mediatek-mt7622_gcc-12.3.0_musl.Linux-x86_64.tar.xz'
image_builder_url='https://downloads.openwrt.org/snapshots/targets/mediatek/mt7622/openwrt-imagebuilder-mediatek-mt7622.Linux-x86_64.tar.xz'
image_feature+=(
  'WIFI_ROUTER_FEATURE_IEEE_802_11_AX'
)
sdk_make+=(
  'feeds/base/mtd' # mtd
)
include_custom_package+=(
  # For mtd,
  'mtd'
)

# Run build.
build
