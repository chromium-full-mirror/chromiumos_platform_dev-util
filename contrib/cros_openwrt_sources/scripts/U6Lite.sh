#!/bin/bash
# Copyright 2025 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
set -e
SCRIPT_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
source "${SCRIPT_DIR}/lib/build.sh"

OPENWRT_COMMIT="3cf0c26074bffb359eb22d77b3ee905b802d8433" # Openwrt-23.05
DEVICE="U6Lite"
DEVICE_NAME="Ubiquiti UniFi 6 Lite"
ROUTER_FEATURES+=(
  'WIFI_ROUTER_FEATURE_IEEE_802_11_AX'
  'WIFI_ROUTER_FEATURE_SAE_EXT_KEY'
  'WIFI_ROUTER_FEATURE_GCMP'
  'WIFI_ROUTER_FEATURE_NOT_U6PLUS_ROUTER'
  'WIFI_ROUTER_FEATURE_U6PLUS_OR_U6LITE'
)
IMAGE_PROFILE="ubnt_unifi-6-lite"
IMAGE_NAME="standard-0.0.2"

build "$@"
