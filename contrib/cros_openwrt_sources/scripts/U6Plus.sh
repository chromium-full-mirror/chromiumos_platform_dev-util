#!/bin/bash
# Copyright 2025 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
set -e
SCRIPT_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
source "${SCRIPT_DIR}/lib/build.sh"

# Openwrt 24.10.5
OPENWRT_COMMIT="4f7e6e554be2aef6a55be36f9f954d56705eb2ee"
DEVICE="U6Plus"
DEVICE_NAME="Ubiquiti UniFi U6+"
ROUTER_FEATURES+=(
  'WIFI_ROUTER_FEATURE_IEEE_802_11_AX'
  'WIFI_ROUTER_FEATURE_U6PLUS_OR_U6LITE'
)
IMAGE_PROFILE="ubnt_unifi-6-plus"
IMAGE_NAME="standard-0.0.2"

build "$@"
