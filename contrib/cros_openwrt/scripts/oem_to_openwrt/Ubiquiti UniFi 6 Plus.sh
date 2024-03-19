#!/bin/bash
# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# This script preforms the necessary commands to install an OpenWrt image to
# UU6P APs for the first time from the OEM image.
#
# OpenWrt wiki: N/A
# OpenWrt device: https://openwrt.org/toh/hwdata/ubiquiti/ubiquiti_unifi_6_plus
#
# Usage:
# 'Ubiquiti UniFi 6 Plus.sh' '<path/to/openwrt/image.bin>' <ap_hostname1> [ap_hostname2...]
#
# Alternative usage with hostnames in lines of a file:
# cat <path/to/hosts.txt> | xargs './Ubiquiti UniFi 6 Plus.sh' '<path/to/openwrt/image.bin>'
#
# Note: When connecting to the OEM image, there will be a prompt to enter the
# the password which cannot be easily bypassed, so that must be entered when
# prompted. This should only be needed once per host, as connections are reused
# through the use of a temporary ssh control file.
#
# The path to the testing_rsa ssh key file is expected to be at `~/.ssh/testing_rsa`,
# but this can be overridden like so:
# TEST_KEY_PATH="path/to/testing_rsa"; 'Ubiquiti UniFi 6 Plus.sh' <args...>

set -e

# Default SSH OEM credentials.
OEM_USERNAME='ubnt'
OEM_PASSWORD='ubnt'
OEM_SSH_OPTS=(
  -o ControlMaster=auto
  -o ControlPath="${OEM_SSH_CONTROL_FILE:='~/.ssh/control_oem_to_openwrt'}"
  -o ControlPersist=1m
  -o StrictHostKeyChecking=no
  -o UserKnownHostsFile=/dev/null
  -o PreferredAuthentications='password'
  -o PasswordAuthentication=yes
)
OPENWRT_SSH_OPTS=(
  -o StrictHostKeyChecking=no
  -o UserKnownHostsFile=/dev/null
  -o PreferredAuthentications='publickey'
  -o IdentityFile="${TEST_KEY_PATH:-~/.ssh/testing_rsa}"
)
REBOOT_WAIT_SECONDS=180

# Collect CLI args.
OPENWRT_IMAGE_BIN_PATH=$1
shift
TARGET_HOSTS=$@
TARGET_HOSTS_COUNT=$#
RET_CODE=-1

# ssh_cmd_oem runs the ssh command on the host with the OEM ssh config options.
function ssh_cmd_oem {
  local HOST="$1"
  shift
  local SSH_CMD="$@"
  echo ssh ${OEM_SSH_OPTS[@]} "${OEM_USERNAME}@${HOST}" "${SSH_CMD[@]}"
  echo "Running ssh command on OEM host '${HOST}', please enter password '${OEM_PASSWORD}' if prompted"
  set +e
  ssh ${OEM_SSH_OPTS[@]} "${OEM_USERNAME}@${HOST}" "${SSH_CMD[@]}"
  RET_CODE=$?
  set -e
}

# ssh_cmd_openwrt runs the ssh command on the host with the OpenWrt ssh config options.
function ssh_cmd_openwrt {
  local HOST="$1"
  shift
  local SSH_CMD="$@"
  set +e
  echo ssh ${OPENWRT_SSH_OPTS[@]} "root@${HOST}" "${SSH_CMD[@]}"
  ssh ${OPENWRT_SSH_OPTS[@]} "root@${HOST}" "${SSH_CMD[@]}"
  RET_CODE=$?
  set -e
}

# clear_oem_ssh_control_file removes the OEM ssh control file if it exists.
function clear_oem_ssh_control_file {
  if [ -f "${OEM_SSH_CONTROL_FILE}" ]; then
    rm "${OEM_SSH_CONTROL_FILE}"
  fi
}

# validate_oem_host validates that the host has the expected OEM firmware installed.
function validate_oem_host {
  local HOST="$1"
  clear_oem_ssh_control_file
  echo "Validating host has OEM firmware"
  VALIDATE_CMD='echo "Validating board name from OEM board file..." && \
  set -x && \
  test -f /etc/board.info && \
  grep -q "board.name=U6+" /etc/board.info && \
  echo "Checking system partitions are as expected..." && \
  test "$(grep PARTNAME /sys/block/mmcblk0/mmcblk0p6/uevent)" = "PARTNAME=kernel0" && \
  test "$(grep PARTNAME /sys/block/mmcblk0/mmcblk0p7/uevent)" = "PARTNAME=kernel1" && \
  test "$(grep PARTNAME /sys/block/mmcblk0/mmcblk0p8/uevent)" = "PARTNAME=bs"'
  ssh_cmd_oem "${HOST}" "${VALIDATE_CMD}"
  if [ "${RET_CODE}" -ne 0 ]; then
    echo "Error: Got return code ${RET_CODE} when trying to run ssh command '${VALIDATE_CMD}' on host '${HOST}'"
    echo "Failed to validate host has OEM firmware"
    exit 2
  fi
  echo "Successfully validated host has OEM firmware"
}

# validate_openwrt_host validates that the host has an OpenWrt firmware installed.
function validate_openwrt_host {
  local HOST="$1"
  echo "Validating host has OpenWrt firmware"
  VALIDATE_CMD='echo "Validating board name from OpenWrt board file..." && \
  test -f /etc/board.json && grep -q "Ubiquiti UniFi 6 Plus" /etc/board.json'
  ssh_cmd_openwrt "${HOST}" "${VALIDATE_CMD}"
  if [ "${RET_CODE}" -ne 0 ]; then
    echo "Error: Got return code ${RET_CODE} when trying to run ssh command '${VALIDATE_CMD}' on host '${HOST}'"
    echo "Failed to validate host has OpenWrt firmware"
    exit 3
  fi
  echo "Successfully validated host has OpenWrt firmware"
}

# ping_until_up pings every second until the host until it is successful or
# runs times out.
function ping_until_up {
  local HOST=$1
  local TIMEOUT_SECONDS=$2
  local SECONDS_REMAINING="${TIMEOUT_SECONDS}"
  RET_CODE=-1
  while [ "${RET_CODE}" -ne 0 ] && [ "${SECONDS_REMAINING}" -gt 0 ]; do
    SECONDS_REMAINING=$((SECONDS_REMAINING-1))
    if [ "${RET_CODE}" -ne -1 ]; then
      sleep 1
    fi
    set +e
    ping -c 1 -w 1 "${HOST}" > /dev/null
    RET_CODE=$?
    set -e
  done
  if [ "${RET_CODE}" -ne 0 ]; then
    echo "Failed to successfully ping host ${HOST} after ${TIMEOUT_SECONDS} seconds"
    exit 4
  fi
  echo "Successfully pinged host '${HOST}' with ${SECONDS_REMAINING} seconds remaining"
}

# install_openwrt_from_oem copies the openwrt image to the host, preforms the
# host-specific actions to make it boot into openwrt, then reboots the host.
function install_openwrt_from_oem {
  local HOST="$1"
  echo "Copying OpenWrt image binary to OEM host..."
  scp -O ${OEM_SSH_OPTS[@]} "${OPENWRT_IMAGE_BIN_PATH}" "${OEM_USERNAME}@${HOST}:/tmp/openwrt.bin"
  local INSTALL_CMD='echo "Configuring device to boot with OpenWrt firmware..." && \
  set -x && \
  test -f /tmp/openwrt.bin && \
  echo 5edfacbf > /proc/ubnthal/.uf && \
  fw_setenv boot_openwrt "fdt addr \$(fdtcontroladdr); fdt rm /signature; bootubnt" && \
  fw_setenv bootcmd_real "run boot_openwrt" && \
  tar xf /tmp/openwrt.bin sysupgrade-ubnt_unifi-6-plus/kernel -O | dd of=/dev/mmcblk0p6 && \
  tar xf /tmp/openwrt.bin sysupgrade-ubnt_unifi-6-plus/root -O | dd of=/dev/mmcblk0p7 && \
  echo -ne "\x00\x00\x00\x00\x2b\xe8\x4d\xa3" > /dev/mmcblk0p8 && \
  echo "Rebooting device..." && \
  reboot'
  ssh_cmd_oem "${HOST}" "${INSTALL_CMD}"
  if [ "${RET_CODE}" -ne 0 ]; then
    echo "Error: Got return code ${RET_CODE} when trying to run ssh command '${INSTALL_CMD}' on host '${HOST}'"
    echo "Failed to configure device with OEM firmware to boot with OpenWrt firmware (may require TFTP OEM firmware reset if retry fails)"
    exit 5
  fi
  clear_oem_ssh_control_file
  echo "Waiting until host fully reboots (${REBOOT_WAIT_SECONDS}s timeout)..."
  sleep 2 # Give it some time for reboot to actually trigger.
  ping_until_up "${HOST}" "${REBOOT_WAIT_SECONDS}"
}

LOG_DIVIDER='******************************************************************'
echo "Using OpenWrt image '${OPENWRT_IMAGE_BIN_PATH}'"
if [ ! -f "${OPENWRT_IMAGE_BIN_PATH}" ]; then
  echo "Invalid image path '${OPENWRT_IMAGE_BIN_PATH}'"
  exit 1
fi
echo "Replacing OEM firmware with OpenWrt firmware on ${TARGET_HOSTS_COUNT} hosts"
for HOST in ${TARGET_HOSTS[@]}; do
  echo -e "\n\n${LOG_DIVIDER}\n${LOG_DIVIDER}\n${HOST}"
  validate_oem_host "${HOST}"
  echo "Replacing OEM firmware with OpenWrt firmware on host ${HOST}"
  install_openwrt_from_oem "${HOST}"
  validate_openwrt_host "${HOST}"
  echo "Successfully replaced OEM firmware with OpenWrt firmware on host ${HOST}"
done
echo -e "\n\n${LOG_DIVIDER}\n${LOG_DIVIDER}\n\nSuccessfully replaced OEM firmware with OpenWrt firmware on ${TARGET_HOSTS_COUNT} hosts"
