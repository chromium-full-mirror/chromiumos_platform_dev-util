#!/bin/bash
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Simple script to update a batch of OpenWrt APs with a new image bin.
#
# Note: This only works with hosts that are directly ssh-able. For hosts that need
# ssh tunnels to access, such as satlab hosts, use the sysupgrade_openwrt_ap_through_tunnel.sh
# script instead.
#
# Usage: bash ./sysupgrade_openwrt_ap.sh <path_to_openwrt_image_bin> <host1> [<host2>...]
#
# The hostnames provided should be the hostnames of the APs themselves, not the
# DUT name. For example, if "coffeelab-dev-host1" was the DUT, you should update
# the router and the pcap OpenWrt APs with the script like so:
#
# bash ./sysupgrade_openwrt_ap.sh <bin_path> coffeelab-dev-host1-router coffeelab-dev-host1-pcap
#
# Note that each host is just attempted to be updated, view the output to
# determine if the update was successful.

BIN_PATH=$1
shift
for HOST in "$@"; do
  echo "Updating ${HOST}"
  scp "${BIN_PATH}" "${HOST}":/tmp/openwrt_update.bin
  if [ "$?" -ne 0 ]; then
    # Try it with the -O flag, may be an image without our OpenSSh customizations.
    scp -O "${BIN_PATH}" "${HOST}":/tmp/openwrt_update.bin
  fi
  ssh "${HOST}" "sysupgrade -n \"/tmp/openwrt_update.bin\""
  echo -e "Updated ${HOST}\n"
done
