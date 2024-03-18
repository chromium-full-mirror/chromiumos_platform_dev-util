#!/bin/bash
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# This script preforms the necessary commands to build the current standard
# image for openwrt deployment with TP-Link Archer C7 v5 test APs.
# OpenWrt wiki: https://openwrt.org/toh/tp-link/archer_c7
# OpenWrt device: https://openwrt.org/toh/hwdata/tp-link/tp-link_archer_c7_v5

# Import build utility script.
set -e
SCRIPT_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
source "${SCRIPT_DIR}/lib/build.sh"

# Configure options for model.
image_profile='tplink_archer-c7-v5'
extra_image_name='standard-1.0.1'
sdk_url='https://downloads.openwrt.org/releases/23.05.0/targets/ath79/generic/openwrt-sdk-23.05.0-ath79-generic_gcc-12.3.0_musl.Linux-x86_64.tar.xz'
image_builder_url='https://downloads.openwrt.org/releases/23.05.0/targets/ath79/generic/openwrt-imagebuilder-23.05.0-ath79-generic.Linux-x86_64.tar.xz'

# Run build.
build
