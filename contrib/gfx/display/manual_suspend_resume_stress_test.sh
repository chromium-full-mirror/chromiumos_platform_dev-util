#!/bin/bash

# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Function to display script usage
usage() {
  echo "Usage: $0 [OPTIONS] HOSTNAME"
  echo "Run a command on a remote host using SSH."
  echo "OPTIONS:"
  echo "  -h, --help     Display this help message"
  exit 1
}

# Parse command-line arguments
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

# Connect to the remote host using SSH
ssh "${hostname}" "stop ui"

#TODO(markyacoub): Implement the rest of the script.
