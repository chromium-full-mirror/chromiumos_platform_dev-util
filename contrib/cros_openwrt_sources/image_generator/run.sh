#!/bin/bash
# Copyright 2025 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

DEVICE=$1
IMAGE_PROFILE=$2
CREATE_ARCHIVE=$3
DEBUG=$4

set -e

OPENWRT_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
cp -r "/build/generated_files/." "${OPENWRT_DIR}/files"
source "${OPENWRT_DIR}/configs/${DEVICE}/run.sh"

# Generate an archive with sysupgrade.bin and several other files.
# This archive could be used to pass into cros_openwrt_image_builder to upload and apply for device
if [ "${CREATE_ARCHIVE}" = true ]; then
    echo "Start creating tar archive"
    TARGETS_DIR="${OPENWRT_DIR}/bin/targets"
    BUILD_INFO_FILE_NAME="cros_openwrt_image_build_info.json"
    sysupgrade_file_path=$(find "${TARGETS_DIR}" \( -name "*-${IMAGE_PROFILE}-squashfs-sysupgrade.itb" -o -name "*-${IMAGE_PROFILE}-squashfs-sysupgrade.bin" \) -print -quit)
    sysupgrade_basename=$(basename "${sysupgrade_file_path}")
    archive_name=${IMAGE_PROFILE}-sysupgrade_$(date +"%Y%m%d-%H%M%S").tar.xz
    sysupgrade_dir=$(dirname "${sysupgrade_file_path}")
    files_to_add=(
        "${sysupgrade_basename}"
        "${BUILD_INFO_FILE_NAME}"
        "version.buildinfo"
        "config.buildinfo"
        "feeds.buildinfo"
        "sha256sums"
        "profiles.json"
    )
    GENERATED_FILES_DIR="/build/generated_files"
    OUTPUT_DIR="${GENERATED_FILES_DIR}/output"
    cp "${GENERATED_FILES_DIR}/etc/cros/${BUILD_INFO_FILE_NAME}" "${sysupgrade_dir}"

    mkdir -p "${OUTPUT_DIR}"
    cd "${OUTPUT_DIR}"
    cp "${GENERATED_FILES_DIR}/etc/cros/${BUILD_INFO_FILE_NAME}" .
    tar -cJf "${archive_name}" -C "${sysupgrade_dir}" "${files_to_add[@]}"
    echo "Tar archive created"
fi

if [ "${DEBUG}" = true ]; then
    echo "Keep container alive"
    /bin/bash
fi