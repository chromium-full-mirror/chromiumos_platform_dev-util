#!/bin/sh
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# This script preloads custom settings and is run on the router after the
# flashing process. These settings configure the router to be run in a CROS lab
# environment.
#
# OpenWrt Image Builder docs: https://openwrt.org/docs/guide-user/additional-software/imagebuilder
# https://openwrt.org/docs/guide-developer/uci-defaults

# Pull in some OpenWRT support functions.
# shellcheck source=/lib/functions/system.sh disable=SC1091
. /lib/functions/system.sh

#
# get_network_device_prefix
#
# Description:
#   Finds the full internal UCI prefix (e.g., network.cfg08927c) for a
#   network device section, given its 'name' option.
#
# Usage:
#   get_network_device_prefix <device_name>
#
# Arguments:
#   $1 - The name of the device to find (e.g., "br-lan", "eth0").
#
# Output:
#   - On success: Prints the full UCI prefix to standard output.
#   - On failure: Prints an error message to standard error.
#
# Return Code:
#   - 0 on success.
#   - 1 on failure (e.g., device not found or argument missing).
#
get_network_device_prefix() {
    if [ -z "$1" ]; then
        echo "Error: No device name supplied to get_network_device_prefix." >&2
        return 1
    fi

    local device_name="$1"
    local prefix
    local section_line

    section_line=$(uci show network | grep ".name='${device_name}'" | head -n 1)

    if [ -n "${section_line}" ]; then
        # 3. If found, extract the first two fields using '.' as a delimiter.
        #    For a line like "network.cfg08927c.name='br-lan'", this will
        #    extract "network.cfg08927c".
        prefix=$(echo "${section_line}" | cut -d'.' -f1,2)
        echo "${prefix}"
        return 0
    else
        echo "Error: Device '${device_name}' not found." >&2
        return 1
    fi
}

setup_br_lan() {
  local board="$1"
  case ${board} in
    bananapi,bpi-r3)
      # Delete existing br-lan device which services the bpi internal LAN and rename
      # br-wan --> br-lan which to stay in line with other OpenWrt images.
      # Remove br-lan only if br-wan exists, prevent from unnecessary delete if run several times

      if br_wan_device=$(get_network_device_prefix "br-wan"); then
        if br_lan_device=$(get_network_device_prefix "br-lan"); then
            uci del "${br_lan_device}"
        fi

        # update value, because it's changed after uci del above
        br_wan_device=$(get_network_device_prefix "br-wan")
        uci set "${br_wan_device}".name='br-lan'

        BR_LAN_L2_PORT="wan"
        uci set "${br_wan_device}".ports="${BR_LAN_L2_PORT}"

        # For some reason by default uci has 2 device with 'wan' name and assigned macaddr.
        # Remove them from config
        if wan_device=$(get_network_device_prefix "wan"); then
          uci delete "${wan_device}"
          if wan_device=$(get_network_device_prefix "wan"); then
            uci delete "${wan_device}"
          fi
        fi
      fi

    ;;
    bananapi,bpi-r4)
      echo "Create static lan interface"

      uci set network.lan='interface'
      uci set network.lan.device='br-lan'
      uci set network.lan.proto='static'
      uci set network.lan.ipaddr='192.168.1.1'
      uci set network.lan.netmask='255.255.255.0'
      uci set network.lan.ip6assign='60'
    ;;
    *)
      BR_LAN_L2_PORT="eth0"
      if [ -f /sys/class/net/lan/operstate ]; then
        # Some devices already have a lan eth0 bridge that cannot be removed.
        BR_LAN_L2_PORT="lan"
      fi

      if br_lan_device=$(get_network_device_prefix "br-lan"); then
        uci set "${br_lan_device}".type='bridge'
        uci set "${br_lan_device}".ports="${BR_LAN_L2_PORT}"
      fi
      ;;
  esac
}

if uci get network.lan >/dev/null 2>&1; then
  uci del network.lan
fi

# Setup br-lan interface which acts as the network bridge for the WAN interface.
setup_br_lan "$(board_name)"

# Add an interface to connect to lab network (wan) as a DHCP client.
uci set network.wan=interface
uci set network.wan.proto='dhcp'

if [ "$(board_name)" != 'bananapi,bpi-r4' ]; then
  uci set network.wan.device='br-lan'
else
  uci set network.wan.device='br-wan'
fi

# Turn on wireless radios on by default.
uci set wireless.radio0.disabled='0'
uci set wireless.radio1.disabled='0'
uci set wireless.radio2.disabled='0'

if [ "$(board_name)" != 'bananapi,bpi-r4' ]; then
  # Remove unnecessary interfaces that will not be used in the lab. The tests will
  # create their own interfaces.
  uci del wireless.default_radio0
  uci del wireless.default_radio1
else
  # Disable default radio interfaces
  uci set wireless.default_radio0.disabled='1'
  uci set wireless.default_radio1.disabled='1'
  uci set wireless.default_radio2.disabled='1'

  # Delete MLD interfaces
  uci del wireless.ap_mld_1
  uci del wireless.default_radio0_mld
  uci del wireless.default_radio1_mld
  uci del wireless.default_radio2_mld
fi

# Commit and reload UCI changes.
uci commit network
uci commit wireless
/etc/init.d/network reload
/sbin/wifi reload

# Link hostapd_cli to the location our tests expect it to be.
ln -s /usr/sbin/hostapd_cli /usr/bin/hostapd_cli

# Configure cros init script.
CROS_TEST_SERVICE_PATH="/etc/init.d/z_cros_test.sh"

if [ -f "${CROS_TEST_SERVICE_PATH}" ]; then
  chmod 755 "${CROS_TEST_SERVICE_PATH}"
  ${CROS_TEST_SERVICE_PATH} enable
  ${CROS_TEST_SERVICE_PATH} start
fi
