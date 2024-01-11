#!/bin/bash
# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Fetches the build info json file from each router and prints the
# device name, image uuid, and image build date.
#
# Example usage:
# ./bash log_openwrt_image_info.sh host1 host2 host3

function log_router_build_info {
  ROUTER_HOST="$1"
  BUILD_INFO_JSON=$(ssh -q "${ROUTER_HOST}" "cat /etc/cros/cros_openwrt_image_build_info.json")
  IMAGE_UUID=$(jq -r ".imageUuid" <<< "${BUILD_INFO_JSON}")
  DEVICE_NAME=$(jq -r ".standardBuildConfig.deviceName" <<< "${BUILD_INFO_JSON}")
  BUILD_TIME=$(jq -r ".buildTime" <<< "${BUILD_INFO_JSON}")
  echo "'${ROUTER_HOST}','${DEVICE_NAME}','${IMAGE_UUID}','${BUILD_TIME}'"
}

echo "router_hostname,standardBuildConfig.deviceName,imageUuid,buildTime"
for HOST in "$@"; do
  # Remove crossk prefix.
  HOST="${HOST#crossk-}"
  if [[ "${HOST}" =~ host[0-9]+$ ]]; then
    # Likely a DUT host, so get both the router and pcap.
    log_router_build_info "${HOST}-router"
    log_router_build_info "${HOST}-pcap"
  else
    log_router_build_info "${HOST}"
  fi
done
