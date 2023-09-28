#!/bin/bash

# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# TODO(markyacoub): Add a description of what this script does.

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

time_until_display_suspend_complete_s=0

# Function to display script usage
usage() {
  echo "Usage: $0 [OPTIONS] HOSTNAME"
  echo "Run a command on a remote host using SSH."
  echo "OPTIONS:"
  echo "  -h, --help     Display this help message"
  exit 1
}

parse_hostname_argument() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -h|--help)
        usage
        ;;
      *)
        hostname="$1"
        ;;
    esac
    shift
  done

  # Check if hostname is provided
  if [ -z "${hostname}" ]; then
    echo "Error: HOSTNAME must be provided."
    usage
  fi
}

run_command_on_remote() {
  local command="$1"
  local background="$2"
  local remote_output

  if [ -z "${isBackground}" ]; then
    isBackground=false
  fi

  if [ -z "${background}" ]; then
    # Run the SSH command and capture the output
    # shellcheck disable=SC2029
    remote_output=$(ssh "${hostname}" "${command}" 2>&1)
    echo "${remote_output}"
  else
    # Run the SSH command in the background and return immediately
    # shellcheck disable=SC2029
    ssh "${hostname}" "${command}" &
  fi
}

check_reqs() {
    # Check if you have a camera connected
    if ! ls /dev/video* 2>/dev/null; then
      echo "No Camera is connected"
      exit 1
    fi

    # Check if fswebcam is installed
    if ! command -v fswebcam &> /dev/null
    then
        echo "fswebcam is not installed. Installing it..."
        sudo apt-get update
        sudo apt-get install -y fswebcam
    fi

    # Check if ImageMagick (convert command) is installed
    if ! command -v convert &> /dev/null; then
        echo "ImageMagick is not installed. Installing it..."
        sudo apt-get update
        sudo apt-get install -y imagemagick
    fi

    mkdir -p temp
}

start_display_suspension_in_s() {
  time_until_dim_ms=$(($1*1000))
  time_until_display_off_ms=$((time_until_dim_ms + (5 * 1000)))
  time_until_display_suspend_ms=$((time_until_display_off_ms + (5 * 1000)))
  time_until_display_suspend_complete_s=$(( (time_until_display_suspend_ms / 1000) + 5 ))

  # Directory where powerd's runtime-settable pref files are stored.
  DIR=/var/lib/power_manager

  # Emulate display naturally going to sleep and Restart powerd to apply the new settings.
  echo "Setting up powerd to dim, turn off, and suspend the display starting in $1 seconds."
  command="echo 1 >${DIR}/ignore_external_policy \
    && echo ${time_until_dim_ms} > ${DIR}/plugged_dim_ms \
    && echo ${time_until_display_off_ms} > ${DIR}/plugged_off_ms \
    && echo ${time_until_display_suspend_ms} > ${DIR}/plugged_suspend_ms \
    && ( (status powerd | fgrep -q 'start/running' ) && restart powerd ) || start powerd"
  run_command_on_remote "${command}"
}

# shellcheck disable=SC2120
capture_photo() {
    num_cameras="${1:-1}"

    if [ "${num_cameras}" -le 0 ] || [ "${num_cameras}" -gt 2 ]; then
      echo "Error: Invalid number of cameras specified. You can capture photos from 1 or 2 cameras."
      return 1
    fi

    timestamp=$(date +"%Y%m%d%H%M%S")

    # Capture a photo from the first camera (video0)
    filename1="temp/image_${timestamp}_camera1.jpg"
    fswebcam -r 640x480 --no-banner --device /dev/video0 "${filename1}"
    echo "${filename1}"

    # Capture a photo from the second camera (video2) if num_cameras is 2
    if [ "${num_cameras}" -eq 2 ]; then
        filename2="temp/image_${timestamp}_camera2.jpg"
        fswebcam -r 640x480 --no-banner --device /dev/video2 "${filename2}"
        echo "${filename2}"
    fi
}

check_screen_on() {
    local image_filename="$1"
    gray_image_filename="${image_filename}_gray.jpg"

    convert "${image_filename}" -colorspace gray "${gray_image_filename}"
    average_brightness=$(identify -verbose "${gray_image_filename}" | grep -i "mean: " | awk '{print $2}' | cut -d, -f1)
    echo "${average_brightness}"

    threshold=200 # through empirical testing, this is the threshold for screen on/off.
    # Make a decision based on the brightness
    if [ "$(echo "${average_brightness} > ${threshold}" | bc)" -eq 1 ]; then
        echo true
    else
        echo false
    fi
}

capture_and_check_2_screens() {
  filenames=$(capture_photo 2)
  image1_filename=$(echo "${filenames}" | sed -n '1p')
  image2_filename=$(echo "${filenames}" | sed -n '2p')

  screen_check1=$(check_screen_on "${image1_filename}")
  brightness_output1=$(echo "${screen_check1}" | sed -n '1p')
  is_screen_on1=$(echo "${screen_check1}" | sed -n '2p')

  screen_check2=$(check_screen_on "${image2_filename}")
  brightness_output2=$(echo "${screen_check2}" | sed -n '1p')
  is_screen_on2=$(echo "${screen_check2}" | sed -n '2p')

  if [ "${is_screen_on1}" = "true" ] && [ "${is_screen_on2}" = "true" ]; then
    echo true
  else
    echo false
  fi

  average_brightness=$(echo "scale=2; (${brightness_output1} + ${brightness_output2}) / 2" | bc)
  echo "${average_brightness}"

  echo "${image1_filename}"
  echo "${image2_filename}"
}

process_logs() {
    local remote_output="$1"
    directory_path=$(echo "${remote_output}" | grep -oE '/tmp/debug-logs_[0-9]+-[0-9]+\.tgz')
    echo "${directory_path}"
}

## SCRIPT STARTS HERE ##

parse_hostname_argument "$@"
check_reqs

# Basic Test Plan:
# 1. Dim, turn off, and suspend the display
# 2. Suspend the SoC
# 3. at Wake, wait for |wait_time| seconds
# 4. take a webcam shot
# 5. Analyze the photo brightness to see if the screen is off
# 6. If it's off, generate logs and save it with the picture.

soc_suspend_time_s=120
wait_until_on_s=60

# Guaranteeing that no dimming or off happens until we've captured a photo.
start_display_suspension_in_s $((wait_until_on_s + 5))

for ((i = 0; i < 1000; i++)); do
  echo -e "${YELLOW}Starting run ${i}..${NC}"
  echo "Waiting for displays to turn off within ${time_until_display_suspend_complete_s}s"
  sleep "${time_until_display_suspend_complete_s}"

  echo "Screen should be off. Suspending SoC for ${soc_suspend_time_s} seconds."
  run_command_on_remote "powerd_dbus_suspend --suspend_for_sec=${soc_suspend_time_s}" true # setting background to true
  echo "Requested SoC suspend for ${soc_suspend_time_s} seconds."

  echo "Verifying that the screen is off.."
  screens_check=$(capture_and_check_2_screens)
  are_screens_on=$(echo "${screens_check}" | sed -n '1p')
  average_brightness=$(echo "${screens_check}" | sed -n '2p')
  if [ "${are_screens_on}" = true ]; then
    echo -e "${RED}Screen seems to be on while SoC is suspended. Reported avr brightness: ${average_brightness}. Retrying..${NC}"
    continue
  else
    echo "Indeed, the screen is off - Gonna wait for suspend for ${soc_suspend_time_s} seconds."
    image_filename1=$(echo "${screens_check}" | sed -n '3p')
    rm "${image_filename1}"
    rm "${image_filename1}_gray.jpg"
    image_filename2=$(echo "${screens_check}" | sed -n '4p')
    rm "${image_filename2}"
    rm "${image_filename2}_gray.jpg"
  fi

  sleep "${soc_suspend_time_s}"
  echo "Waking up after ${soc_suspend_time_s} seconds. gonna wait for ${wait_until_on_s} seconds to take a photo."
  echo "Waiting ${wait_until_on_s}s to ensure screen is on."
  sleep "${wait_until_on_s}"

  screens_check=$(capture_and_check_2_screens)
  average_brightness=$(echo "${screens_check}" | sed -n '2p')
  echo "Screen Average Brightness: ${average_brightness}"
  are_screens_on=$(echo "${screens_check}" | sed -n '1p')
  image_filename1=$(echo "${screens_check}" | sed -n '3p')
  image_filename2=$(echo "${screens_check}" | sed -n '4p')
  if [ "${are_screens_on}" = true ]; then
    echo -e "${GREEN}Screen is on. Deleting image and continuing.${NC}"
    rm "${image_filename1}"
    rm "${image_filename1}_gray.jpg"
    rm "${image_filename2}"
    rm "${image_filename2}_gray.jpg"
    continue
  fi

  # If we get here, the screen is off. Generate logs and report.
  echo -e "${RED}Screen is off. Generating logs and reporting.${NC}"
  output=$(run_command_on_remote "generate_logs")
  echo "out: ${output}"
  logs_dir=$(process_logs "${output}")
  # check if logs_dir is invalid
  if [ -z "${logs_dir}" ]; then
    echo -e "${RED}Failed to get the logs file directory. SSH output: ${output}.${NC}"
    continue
  fi

  # Combine the image and logs into a directory
  combined_dir="temp/run_${i}_${are_screens_on}"
  mkdir -p "${combined_dir}"
  mv "${image_filename1}" "${combined_dir}"
  mv "${image_filename1}_gray.jpg" "${combined_dir}"
  mv "${image_filename2}" "${combined_dir}"
  mv "${image_filename2}_gray.jpg" "${combined_dir}"
  scp "${hostname}:${logs_dir}" "${combined_dir}"

done
