#!/bin/bash
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# This file contains default options to use for running the
# cros_openwrt_image_builder CLI with the a build script. Each variable refers
# to flags of the CLI.
#
# Each model-specific build script may append to or override any of these
# variables to meet the needs of the image being built.

# Required model-specific options.
export image_profile=''
export sdk_url=''
export image_builder_url=''

# Optional model-specific options.
export extra_image_name='standard-0.0.0'
export image_feature=(
  'WIFI_ROUTER_FEATURE_IEEE_802_11_B'
  'WIFI_ROUTER_FEATURE_IEEE_802_11_G'
  'WIFI_ROUTER_FEATURE_IEEE_802_11_N'
  'WIFI_ROUTER_FEATURE_IEEE_802_11_A'
  'WIFI_ROUTER_FEATURE_IEEE_802_11_AC'

  # This is performance based and expected that all OpenWrt models can handle
  # it. If this changes later, this should be moved to per-model build scripts.
  'WIFI_ROUTER_FEATURE_DOUBLE_BRIDGE_OVER_VETH'
)
export sdk_config=(
  # For hostapd.
  'CONFIG_WPA_MBO_SUPPORT=y'
  'CONFIG_WPA_ENABLE_WEP=y'
  'CONFIG_DRIVER_11N_SUPPORT=y'
  'CONFIG_DRIVER_11AC_SUPPORT=y'
  'CONFIG_DRIVER_11AX_SUPPORT=y'
)
export sdk_make=(
  'cros-iperf' # from sourceforge.
  'cros-send-management-frame' # from local source.
  'feeds/base/hostapd' # hostapd*, wpa-supplicant*, wpad*, and eapol-test*.
)
export include_custom_package=(
  # Needed for tast-tests.
  'cros-iperf'
  'cros-send-management-frame'
  'hostapd-common'
  'hostapd-utils'
  'wpad-openssl'
  'wpa-cli'
)
export include_official_package=(
  # Needed for tast-tests.
  'iputils-ping'
  'iputils-arping'
  'kmod-veth'
  'tcpdump'
  'procps-ng-pkill'
  'netperf'
  'sudo'
  'python3-email'
  'python3-idna'
  'python3-light'
  'python3-urllib'

  # More feature-rich version of tar, needed for infra for unpacking bins.
  'tar'

  # Replaces dropbear ssh server, as it is more reliable in our testing.
  'openssh-server'

  # This allows the iptables command to work with the nftables firewall.
  'iptables-nft'

  # Allows the use of rsync instead of scp to avoid issues with different sftp
  # server versions ("scp" vs "scp -O"). Required for Tauto tests (b/320702355).
  'rsync'
)
export exclude_package=(
  # Exclude other versions of packages built by feeds/base/hostapd not being
  # used or that are redundant given those in IncludedCustomPackages.
  'hostapd'
  'hostapd-basic'
  'hostapd-basic-openssl'
  'hostapd-basic-wolfssl'
  'hostapd-mini'
  'hostapd-openssl'
  'hostapd-wolfssl'
  'wpad'
  'wpad-mesh-openssl'
  'wpad-mesh-wolfssl'
  'wpad-basic'
  'wpad-basic-mbedtls'
  'wpad-basic-openssl'
  'wpad-basic-wolfssl'
  'wpad-mini'
  'wpad-wolfssl'
  'wpa-supplicant'
  'wpa-supplicant-mesh-openssl'
  'wpa-supplicant-mesh-wolfssl'
  'wpa-supplicant-basic'
  'wpa-supplicant-mini'
  'wpa-supplicant-openssl'
  'wpa-supplicant-p2p'
  'eapol-test'
  'eapol-test-openssl'
  'eapol-test-wolfssl'

  # Replaced by openssh-server.
  'dropbear'
)
export disable_service=(
  # Our tests configure wireless settings manually.
  'wpad'
  'dnsmasq'
  'odhcpd'

  # Our custom boot script will do this after WAN is confirmed as connected.
  'sysntpd'
)

# Catch-all for anything else, includes any script CLI args.
export EXTRA_ARGS=()
for ARG in "$@"; do
  EXTRA_ARGS+=("${ARG}")
done
