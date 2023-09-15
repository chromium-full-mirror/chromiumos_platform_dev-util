#!/bin/bash
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Configures project path and exe and runs the build_in_chroot.sh script from
# within chroot.
# Meant to be run outside of chroot.

set -e

SCRIPT_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"

CHROOT_DIR="${SCRIPT_DIR}/../../../../../chroot"
SCRIPT_DIR_FROM_CHROOT="/mnt/host/source/src/platform/dev/contrib/btpeer_manager"
PROJECT_EXE_NAME="btpeer_manager"

echo "Building ${PROJECT_EXE_NAME} in chroot"
cros_sdk --chroot "${CHROOT_DIR}" --working-dir "${SCRIPT_DIR_FROM_CHROOT}" bash "./build_in_chroot.sh" "${PROJECT_EXE_NAME}"
echo "Successfully built ${PROJECT_EXE_NAME} at '${SCRIPT_DIR}/go/bin/${PROJECT_EXE_NAME}'"
