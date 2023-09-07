#!/bin/bash

# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# TODO(markyacoub): Add a description of what this script does.

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
  local remote_output

  # Run the SSH command and capture the output
  # shellcheck disable=SC2029
  remote_output=$(ssh "${hostname}" "${command}" 2>&1)
  echo "${remote_output}"

  }

check_pkgs() {
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
}



capture_photo() {
    mkdir -p temp
    timestamp=$(date +"%Y%m%d%H%M%S")
    filename="temp/image_${timestamp}.jpg"

    fswebcam -r 640x480 --no-banner "${filename}"
    echo "${filename}"
}

check_screen_on() {
    local image_filename="$1"
    gray_image_filename="${image_filename}_gray.jpg"

    convert "${image_filename}" -colorspace gray "${gray_image_filename}"
    average_brightness=$(identify -verbose "${gray_image_filename}" | grep -i "mean: " | awk '{print $2}' | cut -d, -f1)
    echo "${average_brightness}"

    threshold=5 # through empirical testing, this is the threshold for screen on/off.
    # Make a decision based on the brightness
    if [ "$(echo "${average_brightness} < ${threshold}" | bc)" -eq 1 ]; then
        echo false
    else
        echo true
    fi
}

process_logs() {
    local remote_output="$1"
    directory_path=$(echo "${remote_output}" | grep -oE '/tmp/debug-logs_[0-9]+-[0-9]+\.tgz')
    echo "${directory_path}"
}

## SCRIPT STARTS HERE ##

parse_hostname_argument "$@"
check_pkgs

# Basic Test Plan:
# 1. Suspend device for |suspend_time| seconds
# 2. at Wake, wait for |wait_time| seconds
# 3. take a webcam shot
# 4. Analyze the photo brightness to see if the screen is off
# 5. If it's off, generate logs and save it with the picture.

suspend_time=30
wait_time=20
if (( wait_time > suspend_time )); then
  echo "Error: wait_time must be less than suspend_time."
  exit 1
fi

iterations=$((15 * 30))  # the 15 hours I'm away from the office
for ((i = 0; i < iterations; i++)); do
  echo "Sleep for ${suspend_time} seconds."
  run_command_on_remote "powerd_dbus_suspend --suspend_for_sec=${suspend_time}"
  echo "Wake up after ${suspend_time} seconds."
  sleep "${wait_time}"

  image_filename=$(capture_photo)
  echo "Photo saved to ${image_filename}"
  brightness_output=$(check_screen_on "${image_filename}")
  average_brightness=$(echo "${brightness_output}" | sed -n '1p')
  echo "Screen Average Brightness: ${average_brightness}"

  screen_status=$(echo "${brightness_output}" | sed -n '2p')
  if [ "${screen_status}" = true ]; then
    echo "Screen is on. Deleting image and continuing."
    rm "${image_filename}"
    rm "${image_filename}_gray.jpg"
    continue
  fi

  # If we get here, the screen is off. Generate logs and report.
  echo "Screen is off. Generating logs and reporting."
  output=$(run_command_on_remote "generate_logs")
  echo "out: ${output}"
  logs_dir=$(process_logs "${output}")
  # check if logs_dir is invalid
  if [ -z "${logs_dir}" ]; then
    echo "Failed to get the logs file directory. SSH output: ${output}"
    continue
  fi

  # Combine the image and logs into a directory
  combined_dir="temp/run_${i}"
  mkdir -p "${combined_dir}"
  mv "${image_filename}" "${combined_dir}"
  scp "${hostname}:${logs_dir}" "${combined_dir}"

done
