#!/bin/bash
# Copyright 2025 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
set -e
SCRIPT_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
source "${SCRIPT_DIR}/lib/build.sh"

OPENWRT_COMMIT="3a481ae21bdc504f7f0325151ee0cb4f25dfd2cd"
MTK_OPENWRT_FEEDS_COMMIT="c8e3540e54b5b07b7afb4a5bd09d6f8db8c8a496"
DEVICE="BananaPi-R4"
DEVICE_NAME="BananaPi-R4"
ROUTER_FEATURES+=(
  'WIFI_ROUTER_FEATURE_IEEE_802_11_AX'
  'WIFI_ROUTER_FEATURE_SAE_EXT_KEY'
  'WIFI_ROUTER_FEATURE_GCMP'
)
IMAGE_PROFILE="bananapi_bpi-r4"
IMAGE_NAME="standard-0.0.1"

build
