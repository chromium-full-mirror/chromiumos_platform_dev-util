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
  [ -d "${CROS_TMP_DIR}" ] && rm -rf "${CROS_TMP_DIR}"
  mkdir -p "${CROS_TMP_DIR}" "${CROS_TMP_STATUS_DIR}" "${CROS_TMP_TEST_DIR}"
  date -u -Is > "${CROS_STATUS_READY_FILE}"
}

log() {
  printf "%s\n" "$*" >&2
}

get_next_boot_id() {
  local last_boot_id=0
  if [ -f "${LAST_BOOT_ID_FILE}" ]; then
    last_boot_id=$(cat "${LAST_BOOT_ID_FILE}")
    if ! [ "${last_boot_id}" -eq "${last_boot_id}" ] 2>/dev/null; then
      last_boot_id=0
    fi
  else
    log "No boot log present, assuming first boot after flashing. Skipping boot check."
    echo -n "0" > "${LAST_BOOT_ID_FILE}"
    return 1
  fi

  local boot_id=$((last_boot_id + 1))
  if [ "${boot_id}" -gt "${MAX_RECORDED_BOOTS}" ]; then
    boot_id=1
  fi
  echo -n "${boot_id}" > "${LAST_BOOT_ID_FILE}"
  echo "${boot_id}"
}

wait_for_connectivity() {
  local boot_name="$1"
  local record_dir="$2"
  local boot_check_result_file="${record_dir}/boot_check_result.csv"
  local success_result="0,up,up"
  local boot_check_result=""
  local uptime_wait_seconds="${MAX_UPTIME_WAIT_SECONDS}"
  local ping_err link_eth0 link_br

  local br_iface="br-lan"
  [ -d "/sys/class/net/br-wan" ] && br_iface="br-wan"

  log "Starting boot check for ${boot_name} (using ${br_iface})"
  while [ "${boot_check_result}" != "${success_result}" ] && [ "${uptime_wait_seconds}" -gt 0 ]; do
    if [ -n "${boot_check_result}" ]; then
      log "Boot check failed, retrying in 1 second (${uptime_wait_seconds} attempts remaining)"
      sleep 1
    fi

    ping -w 1 -c 1 "${PING_CHECK_IP}" > /dev/null 2>&1
    ping_err=$?
    link_eth0=$(cat "/sys/class/net/eth0/operstate" 2>/dev/null || echo "down")
    link_br=$(cat "/sys/class/net/${br_iface}/operstate" 2>/dev/null || echo "down")

    boot_check_result="${ping_err},${link_eth0},${link_br}"
    echo "$(date -u -Is) : ${boot_check_result}" >> "${boot_check_result_file}"
    uptime_wait_seconds=$((uptime_wait_seconds - 1))
    log "Boot check result for ${boot_name}: ${boot_check_result}"
  done
  echo "${boot_check_result}"
}

record_system_state() {
  local record_dir="$1"
  ip link show > "${record_dir}/ip_link_show.txt"
  ifconfig > "${record_dir}/ifconfig.txt"
  free -m > "${record_dir}/free_mem.txt"
  netstat -plunt > "${record_dir}/netstat_plunt.txt"
}

get_sequential_failures() {
  local log_file="$1"
  [ ! -f "${log_file}" ] && { echo 0; return; }
  awk -F, '
    NR > 1 {
      if ($2 == "FAILURE") count++; else count = 0
    }
    END { print count+0 }
  ' "${log_file}"
}

process_boot_outcome() {
  local boot_name="$1"
  local boot_check_result="$2"
  local success_result="0,up,up"
  local num_failures="$3"
  local do_reboot=0

  if [ "${boot_check_result}" != "${success_result}" ]; then
    log "Boot check result: FAILURE ($((num_failures + 1)) sequential failures)"
    echo "${boot_name},FAILURE,${boot_check_result}" >> "${CROS_BOOT_STATE_FILE}"
    if [ "${num_failures}" -lt "${MAX_SEQUENTIAL_AUTO_REBOOTS}" ]; then
      do_reboot=1
    else
      log "Skipping reboot, MAX_SEQUENTIAL_AUTO_REBOOTS reached"
    fi
  else
    log "Boot check result: SUCCESS"
    echo "${boot_name},SUCCESS,${boot_check_result}" >> "${CROS_BOOT_STATE_FILE}"
    log "Updating system time with NTP server '${NTP_SERVER}'"
    ntpd -dnq -p "${NTP_SERVER}"
  fi
  return "${do_reboot}"
}

archive_boot_logs() {
  local record_dir_local="$1"
  local record_dir_archive="$2"
  local record_dir="$3"

  log "Archiving boot logs to ${record_dir_archive}"
  logread > "${record_dir}/logread.txt"
  (cd "${CROS_BOOT_LOG_DIR}" && tar -c -f "${record_dir_archive}" -J --xz "${record_dir_local}")
  rm -rf "${record_dir}"
}

record_and_verify_boot() {
  [ ! -d "${CROS_BOOT_LOG_DIR}" ] && mkdir -p "${CROS_BOOT_LOG_DIR}"

  local boot_id
  boot_id=$(get_next_boot_id)
  if [ $? -ne 0 ]; then
    return
  fi

  local boot_name="boot_$(printf "%02d" "${boot_id}")"
  local record_dir_local="${boot_name}"
  local record_dir="${CROS_BOOT_LOG_DIR}/${record_dir_local}"
  local record_dir_archive="${record_dir}.tar.xz"

  [ -f "${record_dir_archive}" ] && rm "${record_dir_archive}"
  mkdir -p "${record_dir}"

  local boot_check_result_file="${record_dir}/boot_check_result.csv"
  echo "PING_GOOGLE_DNS_ERROR,LINK_STATE_ETH0,LINK_STATE_BR_LAN" > "${boot_check_result_file}"

  local boot_check_result
  boot_check_result=$(wait_for_connectivity "${boot_name}" "${record_dir}")

  record_system_state "${record_dir}"

  local num_failures
  num_failures=$(get_sequential_failures "${CROS_BOOT_STATE_FILE}")

  if [ -f "${CROS_BOOT_STATE_FILE}" ]; then
    local rows
    rows=$(($(wc -l < "${CROS_BOOT_STATE_FILE}") - 1))
    if [ "${rows}" -ge "${MAX_BOOT_STATE_ROWS}" ]; then
      [ -f "${CROS_BOOT_STATE_PREV_FILE}" ] && rm "${CROS_BOOT_STATE_PREV_FILE}"
      log "Boot log full, archiving to '${CROS_BOOT_STATE_PREV_FILE}'"
      mv "${CROS_BOOT_STATE_FILE}" "${CROS_BOOT_STATE_PREV_FILE}"
    fi
  fi

  if [ ! -f "${CROS_BOOT_STATE_FILE}" ]; then
    echo "BOOT_NAME,BOOT_CHECK_RESULT,PING_GOOGLE_DNS_ERROR,LINK_STATE_ETH0,LINK_STATE_BR_LAN" > "${CROS_BOOT_STATE_FILE}"
  fi

  process_boot_outcome "${boot_name}" "${boot_check_result}" "${num_failures}"
  local do_reboot=$?

  archive_boot_logs "${record_dir_local}" "${record_dir_archive}" "${record_dir}"

  if [ "${do_reboot}" -eq 1 ]; then
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
