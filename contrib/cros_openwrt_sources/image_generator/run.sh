#!/bin/bash
# Copyright 2025 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

DEVICE=$1
IMAGE_PROFILE=$2
CREATE_ARCHIVE=$3
DEBUG=$4

set -e -o pipefail

OPENWRT_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
GENERATED_FILES_DIR="/build/generated_files"
cp -r "${GENERATED_FILES_DIR}/." "${OPENWRT_DIR}/files"
source "${OPENWRT_DIR}/configs/${DEVICE}/run.sh"

# Collect third-party notices and repack rootfs/target images so the gzipped
# notices file is embedded inside the flashable image.
NOTICES_FILE_NAME="THIRD_PARTY_NOTICES.txt"
NOTICES_TXT="${GENERATED_FILES_DIR}/${NOTICES_FILE_NAME}"
NOTICES_GZ="${OPENWRT_DIR}/files/usr/share/licenses/${NOTICES_FILE_NAME}.gz"
python3 "${OPENWRT_DIR}/collect_notices.py" "${OPENWRT_DIR}" "${NOTICES_TXT}" "${NOTICES_GZ}"

make -C "${OPENWRT_DIR}" package/install
make -C "${OPENWRT_DIR}" -j"$(nproc)" target/linux/install
make -C "${OPENWRT_DIR}" json_overview_image_info checksum

# Generate an archive with sysupgrade binary, metadata, and notices.
if [ "${CREATE_ARCHIVE}" = true ]; then
    echo "Start creating tar archive"
    TARGETS_DIR="${OPENWRT_DIR}/bin/targets"
    BUILD_INFO_FILE_NAME="cros_openwrt_image_build_info.json"
    sysupgrade_file_path=$(find "${TARGETS_DIR}" \( -name "*-${IMAGE_PROFILE}-squashfs-sysupgrade.itb" -o -name "*-${IMAGE_PROFILE}-squashfs-sysupgrade.bin" \) -print -quit)
    sysupgrade_basename=$(basename "${sysupgrade_file_path}")
    archive_timestamp=$(date +"%Y%m%d-%H%M%S")
    archive_name="${IMAGE_PROFILE}-sysupgrade_${archive_timestamp}.tar.xz"
    sysupgrade_dir=$(dirname "${sysupgrade_file_path}")
    metadata_files=(
        "${BUILD_INFO_FILE_NAME}"
        "${NOTICES_FILE_NAME}"
        "version.buildinfo"
        "config.buildinfo"
        "feeds.buildinfo"
        "sha256sums"
        "profiles.json"
    )
    OUTPUT_DIR="${GENERATED_FILES_DIR}/output"
    cp "${GENERATED_FILES_DIR}/etc/cros/${BUILD_INFO_FILE_NAME}" "${sysupgrade_dir}"
    cp "${NOTICES_TXT}" "${sysupgrade_dir}"

    mkdir -p "${OUTPUT_DIR}"
    cd "${OUTPUT_DIR}"
    cp "${GENERATED_FILES_DIR}/etc/cros/${BUILD_INFO_FILE_NAME}" .
    tar -cJf "${archive_name}" -C "${sysupgrade_dir}" "${sysupgrade_basename}" "${metadata_files[@]}"
    echo "Tar archive created: ${OUTPUT_DIR}/${archive_name}"

    sources_archive_name="${IMAGE_PROFILE}-sources_${archive_timestamp}.tar.gz"
    python3 "${OPENWRT_DIR}/collect_sources.py" \
        "${OPENWRT_DIR}" "${DEVICE}" "${sysupgrade_dir}" \
        "${OUTPUT_DIR}/${sources_archive_name}" "${metadata_files[@]}"
fi

if [ "${DEBUG}" = true ]; then
    echo "Keep container alive"
    /bin/bash
fi
