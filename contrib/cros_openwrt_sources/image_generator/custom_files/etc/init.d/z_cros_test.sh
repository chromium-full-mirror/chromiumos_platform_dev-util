#!/bin/sh /etc/rc.common
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Configure this init script to run after all other scripts by setting the
# last possible order (99) and having the script filename start with "z". Other
# init scripts exist with order 99, so the "z" forces this script to also go
# after those as order is determined by ascii sort.
START=99
STOP=99

CROS_TMP_DIR="/tmp/cros"
CROS_TMP_STATUS_DIR="${CROS_TMP_DIR}/status"
CROS_TMP_TEST_DIR="${CROS_TMP_DIR}/test"
CROS_STATUS_READY_FILE="${CROS_TMP_STATUS_DIR}/ready"
CROS_BOOT_LOG_DIR="/root/cros_boot_log"
CROS_BOOT_STATE_FILE="${CROS_BOOT_LOG_DIR}/boot_state_log.csv"
CROS_BOOT_STATE_PREV_FILE="${CROS_BOOT_LOG_DIR}/boot_state_log.prev.csv"
LAST_BOOT_ID_FILE="${CROS_BOOT_LOG_DIR}/last_boot_id.txt"
MAX_RECORDED_BOOTS=10
MAX_BOOT_STATE_ROWS=1000
MAX_SEQUENTIAL_AUTO_REBOOTS=4
MAX_UPTIME_WAIT_SECONDS=60
PING_CHECK_IP="8.8.8.8"
NTP_SERVER="time.google.com"

prepare_tmp_test_dir() {
  if [ -d "${CROS_TMP_DIR}" ]; then
    rm -rf "${CROS_TMP_DIR}"
  fi
  mkdir "${CROS_TMP_DIR}"
  mkdir "${CROS_TMP_STATUS_DIR}"
  mkdir "${CROS_TMP_TEST_DIR}"
  date -u -Is > "${CROS_STATUS_READY_FILE}"
}

# log prints the args to stderr so they appear in logread.
log() {
  printf "%s\n" "$*" >&2
}

record_and_verify_boot() {
  # Prepare log dir.
  if [ ! -d "${CROS_BOOT_LOG_DIR}" ]; then
    mkdir -p "${CROS_BOOT_LOG_DIR}"
  fi

  # Get the last boot ID (defaulting to 0 if it does not exist or is NAN).
  LAST_BOOT_ID=0
  if [ -f "${LAST_BOOT_ID_FILE}" ]; then
    LAST_BOOT_ID=$(cat "${LAST_BOOT_ID_FILE}")
    if ! [ "${LAST_BOOT_ID}" -eq "${LAST_BOOT_ID}" ] 2>/dev/null; then
      LAST_BOOT_ID=0
    fi
  else
    # We assume this is the first boot after re-imaging OpenWrt and just exit
    # early since the boot check will always fail in this state and rebooting
    # can cause infinite boot loops.
    log "No boot log present, assuming this is first boot after flashing a new OpenWrt image and skipping boot check"
    echo -n "0" > "${LAST_BOOT_ID_FILE}"
    return
  fi

  # Calculate next boot ID.
  BOOT_ID=$((LAST_BOOT_ID+1))
  if [ "${BOOT_ID}" -gt "${MAX_RECORDED_BOOTS}" ]; then
    BOOT_ID=1
  fi
  BOOT_NAME="boot_$(printf "%02d" "${BOOT_ID}")"
  echo -n "${BOOT_ID}" > "${LAST_BOOT_ID_FILE}"

  # Initialize new record dir.
  RECORD_DIR_LOCAL="${BOOT_NAME}"
  RECORD_DIR="${CROS_BOOT_LOG_DIR}/${RECORD_DIR_LOCAL}"
  RECORD_DIR_ARCHIVE="${RECORD_DIR}.tar.xz"
  if [ -f "${RECORD_DIR_ARCHIVE}" ]; then
    # Max boot records reached, replacing old boot archive.
    rm "${RECORD_DIR_ARCHIVE}"
  fi
  mkdir -p "${RECORD_DIR}"

  # Wait for links to all be up or it times out.
  BOOT_CHECK_RESULT_FILE="${RECORD_DIR}/boot_check_result.csv"
  BOOT_CHECK_RESULT_HEADER="PING_GOOGLE_DNS_ERROR,LINK_STATE_ETH0,LINK_STATE_BR_LAN"
  echo "${BOOT_CHECK_RESULT_HEADER}" > "${BOOT_CHECK_RESULT_FILE}"
  SUCCESS_RESULT="0,up,up"
  BOOT_CHECK_RESULT=""
  UPTIME_WAIT_SECONDS="${MAX_UPTIME_WAIT_SECONDS}"
  log "Starting boot check for ${BOOT_NAME}"
  while [ "${BOOT_CHECK_RESULT}" != "${SUCCESS_RESULT}" ] && [ "${UPTIME_WAIT_SECONDS}" -gt 0 ]; do
    if [ "${BOOT_CHECK_RESULT}" != "" ]; then
      log "Boot check failed, retying after 1 second sleep (${UPTIME_WAIT_SECONDS} attempts remaining)"
      sleep 1
    fi
    ping -w 1 -c 1 "${PING_CHECK_IP}" > /dev/null 2> /dev/null
    PING_GOOGLE_DNS_ERROR=$?
    LINK_STATE_ETH0=$(cat "/sys/class/net/eth0/operstate")
    LINK_STATE_BR_LAN=$(cat "/sys/class/net/br-lan/operstate")
    BOOT_CHECK_RESULT="${PING_GOOGLE_DNS_ERROR},${LINK_STATE_ETH0},${LINK_STATE_BR_LAN}"
    echo "$(date -u -Is) : ${BOOT_CHECK_RESULT}" >> "${BOOT_CHECK_RESULT_FILE}"
    UPTIME_WAIT_SECONDS=$((UPTIME_WAIT_SECONDS-1))
    log "Boot check result for ${BOOT_NAME}: ${BOOT_CHECK_RESULT}"
  done

  # Record last network state.
  ip link show > "${RECORD_DIR}/ip_link_show.txt"
  ifconfig > "${RECORD_DIR}/ifconfig.txt"
  free -m > "${RECORD_DIR}/free_mem.txt"
  netstat -plunt > "${RECORD_DIR}/netstat_plunt.txt"

  # Prepare boot state log.
  BOOT_STATE_HEADER="BOOT_NAME,BOOT_CHECK_RESULT,${BOOT_CHECK_RESULT_HEADER}"
  NUM_SEQUENTIAL_FAILURES=0
  if [ ! -f "${CROS_BOOT_STATE_FILE}" ]; then
    # New log.
    echo "${BOOT_STATE_HEADER}" > "${CROS_BOOT_STATE_FILE}"
  else
    # Existing log, count recent failures and make sure it's not full.
    LOG_OFFSET=1
    while true; do
      IS_FAIL_LOG_LINE=$(tail "-${LOG_OFFSET}" "${CROS_BOOT_STATE_FILE}" | head -1 | grep -q "FAILURE"; echo $?)
      if [ "${IS_FAIL_LOG_LINE}" -ne 0 ]; then
        break
      fi
      NUM_SEQUENTIAL_FAILURES=$((NUM_SEQUENTIAL_FAILURES+1))
      LOG_OFFSET=$((LOG_OFFSET+1))
    done
    log "Found ${NUM_SEQUENTIAL_FAILURES} previous sequential boot failures"
    BOOT_STATE_DATA_ROWS=$(($(wc -l < "${CROS_BOOT_STATE_FILE}")-1))
    if [ "${BOOT_STATE_DATA_ROWS}" -ge "${MAX_BOOT_STATE_ROWS}" ]; then
      # Full log, archive log and create a fresh log.
      if [ -f "${CROS_BOOT_STATE_PREV_FILE}" ]; then
        # Remove last archive, we only keep one at a time.
        rm "${CROS_BOOT_STATE_PREV_FILE}"
      fi
      log "Boot log '${CROS_BOOT_STATE_FILE}' full, archiving log to '${CROS_BOOT_STATE_PREV_FILE}"
      mv "${CROS_BOOT_STATE_FILE}" "${CROS_BOOT_STATE_PREV_FILE}"
      echo "${BOOT_STATE_HEADER}" > "${CROS_BOOT_STATE_FILE}"
    fi
  fi

  # Evaluate and record final boot check.
  DO_REBOOT=0
  if [ "${BOOT_CHECK_RESULT}" != "${SUCCESS_RESULT}" ]; then
    # Boot check failed. Reboot if not already rebooted too many times.
    log "Boot check result: FAILURE ($((NUM_SEQUENTIAL_FAILURES+1)) sequential failures)"
    echo "${BOOT_NAME},FAILURE,${BOOT_CHECK_RESULT}" >> "${CROS_BOOT_STATE_FILE}"
    if [ "${NUM_SEQUENTIAL_FAILURES}" -lt "${MAX_SEQUENTIAL_AUTO_REBOOTS}" ]; then
      DO_REBOOT=1
    else
      log "Skipping reboot, MAX_SEQUENTIAL_AUTO_REBOOTS reached (${MAX_SEQUENTIAL_AUTO_REBOOTS})"
    fi
  else
    log "Boot check result: SUCCESS"
    echo "${BOOT_NAME},SUCCESS,${BOOT_CHECK_RESULT}" >> "${CROS_BOOT_STATE_FILE}"
    log "Updating system time with NTP server '${NTP_SERVER}'"
    ntpd -dnq -p "${NTP_SERVER}"
  fi

  # Collect and archive logs.
  log "Archiving boot logs to ${RECORD_DIR_ARCHIVE}"
  logread > "${RECORD_DIR}/logread.txt"
  cd "${CROS_BOOT_LOG_DIR}" && tar -c -f "${RECORD_DIR_ARCHIVE}" -J --xz "${RECORD_DIR_LOCAL}"
  rm -r "${RECORD_DIR}"

  if [ "${DO_REBOOT}" -eq 1 ]; then
    log "Rebooting"
    reboot
  fi
}

start() {
  record_and_verify_boot
  prepare_tmp_test_dir
}

stop() {
    rm -rf "${CROS_TMP_DIR}"
}
