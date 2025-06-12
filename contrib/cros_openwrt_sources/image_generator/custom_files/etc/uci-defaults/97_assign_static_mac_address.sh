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

  # $1 $2 %3 $4 ould store all components like e5510793, 216655bc, 731c89c8, 61a9e4bc
  val_1=$1
  val_2=$2
  val_3=$3
  byte_part1=${val_1:start_char:2}
  byte_part2=${val_2:start_char:2}
  byte_part3=${val_3:start_char:2}

  echo "${MAC_PREFIX}:${byte_part1}:${byte_part2}:${byte_part3}"
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
