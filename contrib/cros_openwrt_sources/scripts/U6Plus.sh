#!/bin/bash
# Copyright 2025 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
set -e
SCRIPT_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
source "${SCRIPT_DIR}/lib/build.sh"

OPENWRT_COMMIT="1fad1b4965dc6f4e5f4ba7b9605987f443a4c276"
DEVICE="U6Plus"
DEVICE_NAME="Ubiquiti UniFi U6+"
ROUTER_FEATURES+=(
  'WIFI_ROUTER_FEATURE_IEEE_802_11_AX'
  'WIFI_ROUTER_FEATURE_U6PLUS_OR_U6LITE'
)
IMAGE_PROFILE="ubnt_unifi-6-plus"
IMAGE_NAME="standard-0.0.1"

build "$@"
