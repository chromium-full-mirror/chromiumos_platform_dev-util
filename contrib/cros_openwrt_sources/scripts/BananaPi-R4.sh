#!/bin/bash
# Copyright 2025 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
set -e
SCRIPT_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
source "${SCRIPT_DIR}/lib/build.sh"

OPENWRT_COMMIT="34ae6ba2d861e710e3c130c95cdb7eb4a1286121"
MTK_OPENWRT_FEEDS_COMMIT="9b5754e3a361635595ea31a14176b14169e69d18"
DEVICE="BananaPi-R4"
DEVICE_NAME="Bananapi BPi-R4"
ROUTER_FEATURES+=(
  'WIFI_ROUTER_FEATURE_IEEE_802_11_AX'
  'WIFI_ROUTER_FEATURE_SAE_EXT_KEY'
  'WIFI_ROUTER_FEATURE_GCMP'
  'WIFI_ROUTER_FEATURE_IEEE_802_11_BE'
  'WIFI_ROUTER_FEATURE_IEEE_802_11_AX_E'
  'WIFI_ROUTER_FEATURE_NOT_U6PLUS_ROUTER'
)
IMAGE_PROFILE="bananapi_bpi-r4"
IMAGE_NAME="standard-0.0.3"

build "$@"
