#!/bin/bash

# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Simple script to download and unpack boot logs from OpenWrt routers quickly.
#
# Usage: bash ./collect_boot_logs.sh <host1> [<host2>...]

SCRIPT_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
COLLECTION_TIMESTAMP=$(date +%s)
OUTPUT_BASE_DIR="${SCRIPT_DIR}/logs/cros_boot_log_${COLLECTION_TIMESTAMP}"

for HOST in "$@"; do
  echo "Downloading logs from ${HOST}"
  LOG_DIR="${OUTPUT_BASE_DIR}/${HOST}/"
  mkdir -p "${LOG_DIR}"
  scp -O -r "${HOST}:/root/cros_boot_log/*" "${LOG_DIR}"
  echo "Unpacking logs from ${HOST}"
  cd "${LOG_DIR}" && find . -name "*.tar.xz" \
  -exec echo Unpacking {} \; \
  -exec tar -xf {} \; \
  -exec rm {} \;
  echo -en "Successfully downloaded and unpacked boot logs from host ${HOST}\n\n"
done

echo "Log collection complete and saved to ${OUTPUT_BASE_DIR}"
