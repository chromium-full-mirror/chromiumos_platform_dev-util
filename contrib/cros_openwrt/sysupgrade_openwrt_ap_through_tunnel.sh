#!/bin/bash
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Simple script to update a batch of OpenWrt APs with a new image bin through ssh tunnels.
#
# Usage: bash ./sysupgrade_openwrt_ap_through_tunnel.sh <path_to_openwrt_image_bin> <port1> [<port2>...]
#
# The ports provided are the port of the local ssh tunnel that routes to the
# desired OpenWrt AP.
#
# It's recommend to create the tunnels with "labtunnel hosts --ssh <host1> [-ssh <host2>...]"
# and then record the ports it logs for each host. Then, use those ports with this
# script.
#
# For example, if you wanted to update satlab-0wgatfqi2208800c-chromeos1-dev-host26-pcap,
# you would first run "labtunnel hosts --ssh satlab-0wgatfqi2208800c-chromeos1-dev-host26-pcap",
# then review the logs to find the localhost port, such as port 2200. Then, you
# would call this script like so:
#
# bash ./sysupgrade_openwrt_ap.sh <bin_path> 2200
#
# Note that each host is just attempted to be updated, view the output to
# determine if the update was successful.

BIN_PATH=$1
shift
for LOCALHOST_PORT in "$@"; do
  echo "Updating host through tunnel at localhost:${LOCALHOST_PORT}"
  scp -i ~/.ssh/testing_rsa -P "${LOCALHOST_PORT}" -o StrictHostKeyChecking="no" -o UserKnownHostsFile="/dev/null" "${BIN_PATH}" root@localhost:/tmp/openwrt_update.bin
  if [ "$?" -ne 0 ]; then
    # Try it with the -O flag, may be an image without our OpenSSh customizations.
    scp -O -i ~/.ssh/testing_rsa -P "${LOCALHOST_PORT}" -o StrictHostKeyChecking="no" -o UserKnownHostsFile="/dev/null" "${BIN_PATH}" root@localhost:/tmp/openwrt_update.bin
  fi
  ssh -i ~/.ssh/testing_rsa -p "${LOCALHOST_PORT}" -o StrictHostKeyChecking="no" -o UserKnownHostsFile="/dev/null" root@localhost "sysupgrade -n \"/tmp/openwrt_update.bin\""
  echo -e "Updated host through tunnel at localhost${LOCALHOST_PORT}\n"
done
