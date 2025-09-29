#!/bin/sh
# Copyright 2025 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Pull in some OpenWRT support functions.
# shellcheck source=/lib/functions/system.sh disable=SC1091
. /lib/functions/system.sh


get_macaddr_for_hw_id() {
  local hw_id="$1"
  local offset="$2"
  local start_char=$(( (3 - offset) * 2 ))

  local MAC_PREFIX="02:b4:79"

  if [ -z "${hw_id}" ]; then
    exit 0
  fi

  OLD_IFS="${IFS}"
  IFS=':'
  set -- ${hw_id}
  IFS="${OLD_IFS}"

  val_1=$1
  val_2=$2
  val_3=$3

  # Extract raw substrings. Use :-0 default to prevent errors if a string is too short
  # and the substring returns nothing.
  raw_p1=${val_1:start_char:2}
  raw_p2=${val_2:start_char:2}
  raw_p3=${val_3:start_char:2}

  # Force formatting as a 2-digit hex number. This converts "0" to "00".
  # We must prepend "0x" so the shell arithmetic ((...)) treats the string as hex.
  byte_part1=$(printf "%02X" $((0x${raw_p1:-0})))
  byte_part2=$(printf "%02X" $((0x${raw_p2:-0})))
  byte_part3=$(printf "%02X" $((0x${raw_p3:-0})))

  current_mac="${MAC_PREFIX}:${byte_part1}:${byte_part2}:${byte_part3}"

  # This hack added to keep MAC address on br-wan same as it was on first image were wan had same MAC as eth0
  # Now eth0 would be 1 less, and uci scripts would assign eth0 MAC + 1 to eth2 and wan,
  # so br-wan would be same as before but different from eth0 and no updates required for already flashed BPi's

  # --- START: REPLACEMENT CODE ---
  # Remove the colons from the MAC address string
  mac_no_colons=$(echo "${current_mac}" | tr -d ':')

  # Use shell arithmetic to convert from hex to decimal, subtract 1
  # The shell understands "0x..." as a hex number inside ((...))
  prev_mac_decimal=$((0x${mac_no_colons} - 1))

  # Use printf to convert the new decimal value back to a 12-character
  # hexadecimal string, padded with leading zeros.
  prev_mac_hex=$(printf "%012X" "${prev_mac_decimal}")
  # --- END: REPLACEMENT CODE ---

  # Finally, use sed to insert the colons every two characters.
  prev_mac=$(echo "${prev_mac_hex}" | sed 's/../&:/g;s/:$//')

  echo "${prev_mac}"
  return 0
}

setup_macaddr() {
  local board="$1"
  if [ "${board}" != 'bananapi,bpi-r4' ]; then
    exit 0
  fi

  # hw_id have form e5510793:216655bc:731c89c8:61a9e4bc
  hw_id=$(fw_printenv hw_id | awk -F'=' '{print $2}')
  NEW_MAC_ETH0=$(get_macaddr_for_hw_id "${hw_id}" 0)
  NEW_MAC_ETH1=$(get_macaddr_for_hw_id "${hw_id}" 1)

  CURRENT_MAC_ETH0=$(fw_printenv --noheader ethaddr)
  CURRENT_MAC_ETH1=$(fw_printenv --noheader eth1addr)
  local need_reboot=0

  if [ "${CURRENT_MAC_ETH0}" != "${NEW_MAC_ETH0}" ]; then
    fw_setenv ethaddr "${NEW_MAC_ETH0}"
    need_reboot=1
  fi

  if [ "${CURRENT_MAC_ETH1}" != "${NEW_MAC_ETH1}" ]; then
    fw_setenv eth1addr "${NEW_MAC_ETH1}"
    need_reboot=1
  fi

  if [ "${need_reboot}" = 1 ]; then
    reboot
  fi
}

setup_macaddr "$(board_name)"
