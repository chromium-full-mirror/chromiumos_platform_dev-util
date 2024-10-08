#!/bin/bash
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

setup_br_lan() {
  local board="$1"
  case ${board} in
    bananapi,bpi-r3)
      # Delete existing br-lan device which services the bpi internal LAN and rename
      # br-wan --> br-lan which to stay in line with other OpenWrt images.
      uci del network.@device[0]
      uci set network.@device[0].name='br-lan'
    ;;
    *)
      BR_LAN_L2_PORT="eth0"
      if [ -f /sys/class/net/lan/operstate ]; then
        # Some devices already have a lan eth0 bridge that cannot be removed.
        BR_LAN_L2_PORT="lan"
      fi
      uci set network.@device[0]=device
      uci set network.@device[0].name='br-lan'
      uci set network.@device[0].type='bridge'
      uci set network.@device[0].ports="${BR_LAN_L2_PORT}"
      ;;
  esac
}

uci del network.lan
/etc/init.d/network reload

# Setup br-lan interface which acts as the network bridge for the WAN interface.
setup_br_lan "$(board_name)"

# Add an interface to connect to lab network (wan) as a DHCP client.
uci set network.wan=interface
uci set network.wan.proto='dhcp'
uci set network.wan.device='br-lan'

# Turn on wireless radios on by default.
uci set wireless.radio0.disabled='0'
uci set wireless.radio1.disabled='0'

# Remove unnecessary interfaces that will not be used in the lab. The tests will
# create their own interfaces.
uci del wireless.default_radio0
uci del wireless.default_radio1

# Commit and reload UCI changes.
uci commit network
uci commit wireless
/etc/init.d/network reload
/sbin/wifi reload

# Link hostapd_cli to the location our tests expect it to be.
ln -s /usr/sbin/hostapd_cli /usr/bin/hostapd_cli

# Configure cros init script.
chmod 755 /etc/init.d/z_cros_test.sh
/etc/init.d/z_cros_test.sh enable
/etc/init.d/z_cros_test.sh start
