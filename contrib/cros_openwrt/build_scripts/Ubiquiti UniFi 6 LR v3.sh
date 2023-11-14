#!/bin/bash
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# This script preforms the necessary commands to build the current standard
# image for openwrt deployment with Ubiquiti UniFi 6 LR v3 test APs.
# OpenWrt device page: https://openwrt.org/toh/hwdata/ubiquiti/ubiquiti_unifi_6_lr_v3
# OpenWrt Wiki page: https://openwrt.org/toh/ubiquiti/unifi_6_lr
# Target downloads page: https://downloads.openwrt.org/snapshots/targets/mediatek/mt7622/

set -e

BUILD_PROFILE="ubnt_unifi-6-lr-v3"
SDK_DOWNLOAD_URL="https://downloads.openwrt.org/snapshots/targets/mediatek/mt7622/openwrt-sdk-mediatek-mt7622_gcc-12.3.0_musl.Linux-x86_64.tar.xz"
IMAGE_BUILDER_DOWNLOAD_URL="https://downloads.openwrt.org/snapshots/targets/mediatek/mt7622/openwrt-imagebuilder-mediatek-mt7622.Linux-x86_64.tar.xz"

SCRIPT_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
BUILD_DIR="${SCRIPT_DIR}/../build/${BUILD_PROFILE}"
IMAGE_BUILDER_WORKING_DIR="${BUILD_DIR}/cros_openwrt"

# Initialize build dir.
echo "Building standard OpenWrt image for profile ${BUILD_PROFILE} in ${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

# Building cros_openwrt_image_builder.
echo "Compiling cros_openwrt_image_builder"
bash "${SCRIPT_DIR}/../image_builder/build.sh"

# Build packages once without customizations to initialize sdk.
echo "Building initial packages with standard sdk"
cros_openwrt_image_builder build packages \
--working_dir "${IMAGE_BUILDER_WORKING_DIR}" \
--sdk_url "${SDK_DOWNLOAD_URL}"

# Build final image.
echo "Building image"
cros_openwrt_image_builder build image \
--working_dir "${IMAGE_BUILDER_WORKING_DIR}" \
--image_builder_url "${IMAGE_BUILDER_DOWNLOAD_URL}" \
--image_profile "${BUILD_PROFILE}" \
--image_feature WIFI_ROUTER_FEATURE_IEEE_802_11_A \
--image_feature WIFI_ROUTER_FEATURE_IEEE_802_11_B \
--image_feature WIFI_ROUTER_FEATURE_IEEE_802_11_G \
--image_feature WIFI_ROUTER_FEATURE_IEEE_802_11_N \
--image_feature WIFI_ROUTER_FEATURE_IEEE_802_11_AC \
--image_feature WIFI_ROUTER_FEATURE_IEEE_802_11_AX
