#!/bin/bash
# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# This script preforms the necessary commands to install an OpenWrt image to
# UU6L APs for the first time from the OEM image.
#
# The OEM firmware will be updated using the provided
# binary prior to installing OpenWrt, as the installation does not work with
# newer firmware versions.
#
# OpenWrt device: https://openwrt.org/toh/hwdata/ubiquiti/ubiquiti_unifi_6_lite
#
# Usage:
# 'Ubiquiti UniFi 6 Lite.sh' '<path/to/oem_firmware.bin>' '<path/to/openwrt_image.bin>' <ap_hostname1> [ap_hostname2...]
#
# Alternative usage with hostnames in lines of a file:
# cat <path/to/hosts.txt> | xargs './Ubiquiti UniFi 6 Lite.sh' '<path/to/oem_firmware.bin>' '<path/to/openwrt_image.bin>'
#
# Note: When connecting to the OEM image, there will be a prompt to enter the
# the password which cannot be easily bypassed, so that must be entered when
# prompted. This should only be needed twice per host, as connections are reused
# through the use of a temporary ssh control file.
#
# The path to the testing_rsa ssh key file is expected to be at `~/.ssh/testing_rsa`,
# but this can be overridden like so:
# TEST_KEY_PATH="path/to/testing_rsa"; 'Ubiquiti UniFi 6 Lite.sh' <args...>

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
  -o MACs=hmac-sha1
)
OPENWRT_SSH_OPTS=(
  -o StrictHostKeyChecking=no
  -o UserKnownHostsFile=/dev/null
  -o PreferredAuthentications='publickey'
  -o IdentityFile="${TEST_KEY_PATH:-~/.ssh/testing_rsa}"
)
REBOOT_WAIT_SECONDS=180
OPENWRT_FIRST_BOOT_SSH_ATTEMPTS=30 # About 10s per attempt
OPENWRT_FIRST_BOOT_SSH_DELAY_BETWEEN_ATTEMPTS_SECONDS=10

# Collect CLI args.
OEM_FIRMWARE_BIN_PATH=$1
OPENWRT_IMAGE_BIN_PATH=$2
shift
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
  grep -q "board.name=U6-Lite" /etc/board.info'
  ssh_cmd_oem "${HOST}" "${VALIDATE_CMD}"
  if [ "${RET_CODE}" -ne 0 ]; then
    echo "Error: Got return code ${RET_CODE} when trying to run ssh command '${VALIDATE_CMD}' on host '${HOST}'"
    echo "Failed to validate host has OEM firmware"
    exit 2
  fi
  echo "Successfully validated host has OEM firmware"
}

# validate_openwrt_host validates that the host has an OpenWrt firmware installed.
# Retries until successful or no more attempts remain, as this can be flaky
# during the first boot even after a successful ping.
function validate_openwrt_host {
  local HOST="$1"
  local CURR_ATTEMPT=1
  RET_CODE=-1
  while [ "${RET_CODE}" -ne 0 ] && [ "${CURR_ATTEMPT}" -le "${OPENWRT_FIRST_BOOT_SSH_ATTEMPTS}" ]; do
    if [ "${CURR_ATTEMPT}" -gt 1 ]; then
      echo "Waiting ${OPENWRT_FIRST_BOOT_SSH_DELAY_BETWEEN_ATTEMPTS_SECONDS} until retrying..."
      sleep "${OPENWRT_FIRST_BOOT_SSH_DELAY_BETWEEN_ATTEMPTS_SECONDS}"
      echo "Waiting for host to be pingable (${REBOOT_WAIT_SECONDS} timeout)..."
      ping_until_up "${HOST}" "${REBOOT_WAIT_SECONDS}"
    fi
    echo "Validating host has OpenWrt firmware (attempt ${CURR_ATTEMPT} of ${OPENWRT_FIRST_BOOT_SSH_ATTEMPTS})"
    VALIDATE_CMD='echo "Validating board name from OpenWrt board file..." && \
    set -x && \
    test -f /etc/board.json && \
    grep -q "Ubiquiti UniFi 6 Lite" /etc/board.json'
    ssh_cmd_openwrt "${HOST}" "${VALIDATE_CMD}"
    if [ "${RET_CODE}" -eq 0 ]; then
      # Success.
      break
    elif [ "${RET_CODE}" -eq 1 ] || [ "${RET_CODE}" -eq 2 ]; then
      # Successful grep run, but device name not found in file or file does not exist.
      echo "Error: Got return code ${RET_CODE} when trying to run ssh command '${VALIDATE_CMD}' on host '${HOST}'"
      echo "Failed to validate host has OpenWrt firmware on attempt ${CURR_ATTEMPT} (board info does not match)"
      exit 3
    fi
    # Not an expected return code, continue loop to retry.
    echo "Error: Got return code ${RET_CODE} when trying to run ssh command '${VALIDATE_CMD}' on host '${HOST}'"
    echo "Failed to validate host has OpenWrt firmware on attempt ${CURR_ATTEMPT} (failed to run test command)"
    CURR_ATTEMPT=$((CURR_ATTEMPT+1))
  done
  if [ "${RET_CODE}" -ne 0 ]; then
    echo "Failed to validate host has OpenWrt firmware (no attempts remaining)"
    exit 3
  fi
  echo "Successfully validated host has OpenWrt firmware on attempt ${CURR_ATTEMPT}"
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
  fw_setenv devmode TRUE && \
  fw_setenv boot_openwrt "fdt addr \$(fdtcontroladdr); fdt rm /signature; bootubnt" && \
  fw_setenv bootcmd "run boot_openwrt" && \
  dd if=/dev/zero bs=1 count=1 of=/dev/mtdblock4 && \
  dd if=/tmp/openwrt.bin of=/dev/mtdblock6 && \
  dd if=/tmp/openwrt.bin of=/dev/mtdblock7 && \
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

# install_oem_from_oem copies the OEM firmware image to the host, updates it,
# then reboots the host.
function install_oem_from_oem {
  local HOST="$1"
  echo "Copying OEM firmware image binary to OEM host..."
  scp -O ${OEM_SSH_OPTS[@]} "${OEM_FIRMWARE_BIN_PATH}" "${OEM_USERNAME}@${HOST}:/tmp/fwupdate.bin"
  local INSTALL_CMD='echo "Upgrading OEM firmware..." && \
  set -x && \
  test -f /tmp/fwupdate.bin && \
  syswrapper.sh upgrade2'
  ssh_cmd_oem "${HOST}" "${INSTALL_CMD}"
  if [ "${RET_CODE}" -ne 255 ]; then
    echo "Error: Got return code ${RET_CODE} (expected 255 from upgrade disconnect) when trying to run ssh command '${INSTALL_CMD}' on host '${HOST}'"
    echo "Failed to upgrade OEM firmware (may require TFTP OEM firmware reset if retry fails)"
    exit 5
  fi
  clear_oem_ssh_control_file
  echo "Waiting until host fully reboots (${REBOOT_WAIT_SECONDS}s timeout)..."
  sleep 2 # Give it some time for reboot to actually trigger.
  ping_until_up "${HOST}" "${REBOOT_WAIT_SECONDS}"
}

LOG_DIVIDER='******************************************************************'
echo "Using OEM firmware image '${OEM_FIRMWARE_BIN_PATH}'"
if [ ! -f "${OEM_FIRMWARE_BIN_PATH}" ]; then
  echo "Invalid image path '${OEM_FIRMWARE_BIN_PATH}'"
  exit 1
fi
echo "Using OpenWrt image '${OPENWRT_IMAGE_BIN_PATH}'"
if [ ! -f "${OPENWRT_IMAGE_BIN_PATH}" ]; then
  echo "Invalid image path '${OPENWRT_IMAGE_BIN_PATH}'"
  exit 1
fi
echo "Replacing OEM firmware with OpenWrt firmware on ${TARGET_HOSTS_COUNT} hosts"
for HOST in ${TARGET_HOSTS[@]}; do
  echo -e "\n\n${LOG_DIVIDER}\n${LOG_DIVIDER}\n${HOST}"
  validate_oem_host "${HOST}"
  echo "Updating OEM firmware on host ${HOST}"
  install_oem_from_oem "${HOST}"
  validate_oem_host "${HOST}"
  echo "Successfully updated OEM firmware on host ${HOST}"
  echo "Replacing OEM firmware with OpenWrt firmware on host ${HOST}"
  install_openwrt_from_oem "${HOST}"
  validate_openwrt_host "${HOST}"
  echo "Successfully replaced OEM firmware with OpenWrt firmware on host ${HOST}"
done
echo -e "\n\n${LOG_DIVIDER}\n${LOG_DIVIDER}\n\nSuccessfully replaced OEM firmware with OpenWrt firmware on ${TARGET_HOSTS_COUNT} hosts"
