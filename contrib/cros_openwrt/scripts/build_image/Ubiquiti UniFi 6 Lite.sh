#!/bin/bash
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# This script preforms the necessary commands to build the current standard
# image for openwrt deployment with Ubiquiti UniFi 6 Lite test APs.
# OpenWrt wiki:
# OpenWrt device: https://openwrt.org/toh/hwdata/ubiquiti/ubiquiti_unifi_6_lite

# Import build utility script.
set -e
SCRIPT_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
source "${SCRIPT_DIR}/lib/build.sh"

# Configure options for model.
image_profile='ubnt_unifi-6-lite'
extra_image_name='standard-1.0.2'
sdk_url='https://archive.openwrt.org/releases/23.05.0/targets/ramips/mt7621/openwrt-sdk-23.05.0-ramips-mt7621_gcc-12.3.0_musl.Linux-x86_64.tar.xz'
image_builder_url='https://archive.openwrt.org/releases/23.05.0/targets/ramips/mt7621/openwrt-imagebuilder-23.05.0-ramips-mt7621.Linux-x86_64.tar.xz'
image_feature+=(
  'WIFI_ROUTER_FEATURE_IEEE_802_11_AX'
  'WIFI_ROUTER_FEATURE_SAE_EXT_KEY'
  'WIFI_ROUTER_FEATURE_GCMP'
)

# Run build.
build
