#!/bin/bash -e
#
# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
#
# Sets up NAT64 on the amarisoft callbox.

# If tayga is not installed:
#   sudo yum install tayga -y

TUN_DEV_NAME=nat64
TUN_DEV_ADDR=192.168.255.1
TUN_DYNAMIC_POOL=192.168.255.0/24
TAYGA_CONFIG_PATH="/etc/tayga/${TUN_DEV_NAME}.conf"

systemctl stop "tayga@${TUN_DEV_NAME}"

cat <<EOT > "${TAYGA_CONFIG_PATH}"
tun-device ${TUN_DEV_NAME}
ipv4-addr ${TUN_DEV_ADDR}
ipv6-addr 2001:db8:1::2
prefix 64:ff9b::/96
dynamic-pool ${TUN_DYNAMIC_POOL}
data-dir /var/spool/tayga
EOT

tayga --mktun --config="${TAYGA_CONFIG_PATH}" || true
ip link set "${TUN_DEV_NAME}" up
ip addr add "${TUN_DEV_ADDR}" dev "${TUN_DEV_NAME}"
ip route add 64:ff9b::/96 dev "${TUN_DEV_NAME}"
ip route add "${TUN_DYNAMIC_POOL}" dev "${TUN_DEV_NAME}"

systemctl daemon-reload
systemctl enable "tayga@${TUN_DEV_NAME}"
systemctl start "tayga@${TUN_DEV_NAME}"
