#!/bin/bash
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# This script preforms the necessary commands to build the current standard
# image for openwrt deployment with Ubiquiti UniFi 6+ test APs.
# OpenWrt wiki: N/A
# OpenWrt device: https://openwrt.org/toh/hwdata/ubiquiti/ubiquiti_unifi_6_plus

# Import build utility script.
set -e
SCRIPT_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
source "${SCRIPT_DIR}/lib/build.sh"

# Configure options for model.
image_profile='ubnt_unifi-6-plus'
extra_image_name='standard-1.0.0'
sdk_url='gs://chromeos-connectivity-test-artifacts/wifi_router/prebuilt/openwrt-sdk-mediatek-filogic_gcc-13.3.0_musl.Linux-x86_64.tar.zst'
image_builder_url='gs://chromeos-connectivity-test-artifacts/wifi_router/prebuilt/openwrt-imagebuilder-mediatek-filogic.Linux-x86_64.tar.zst'
image_feature+=(
  'WIFI_ROUTER_FEATURE_IEEE_802_11_AX'
)

# Run build.
build
