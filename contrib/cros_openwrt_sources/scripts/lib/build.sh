#!/bin/bash
# Copyright 2025 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

export OPENWRT_COMMIT=""
export MTK_OPENWRT_FEEDS_COMMIT=""
export DEVICE=""
export DEVICE_NAME=""
export IMAGE_PROFILE=""
export IMAGE_NAME='standard-0.0.0'
export ROUTER_FEATURES=(
  'WIFI_ROUTER_FEATURE_IEEE_802_11_B'
  'WIFI_ROUTER_FEATURE_IEEE_802_11_G'
  'WIFI_ROUTER_FEATURE_IEEE_802_11_N'
  'WIFI_ROUTER_FEATURE_IEEE_802_11_A'
  'WIFI_ROUTER_FEATURE_IEEE_802_11_AC'

  # This is performance based and expected that all OpenWrt models can handle
  # it. If this changes later, this should be moved to per-model build scripts.
  'WIFI_ROUTER_FEATURE_DOUBLE_BRIDGE_OVER_VETH'
)
export IMAGE_BUILDER_VERSION="2.0.0"

CURRENT_TIME=$(date -u +"%Y-%m-%dT%H:%M:%S.%N")
IMAGE_UUID=$(uuidgen)

generate_cros_files() {
    BUILD_DIR=$1
    CROS_FOLDER="${BUILD_DIR}/generated_files/etc/cros"
    mkdir -p "${CROS_FOLDER}"

    generate_ssh_banner
    generate_cros_openwrt_image_build_info_file
}

generate_ssh_banner() {
    ROOTFS_SSH_BANNER_FILE="${CROS_FOLDER}/ssh_banner.txt"
    echo "
    ------------------------------------------------------------
    ChromeOS Test OpenWRT
    * Device: ${DEVICE}
    * ImageUUID: ${IMAGE_UUID}
    ------------------------------------------------------------
    " > "${ROOTFS_SSH_BANNER_FILE}"
}

generate_cros_openwrt_image_build_info_file() {
    ROOTFS_INFO_FILE="${CROS_FOLDER}/cros_openwrt_image_build_info.json"

    json_features_string=""

    for feature in "${ROUTER_FEATURES[@]}"; do
        # If json_features_string is not empty, add a comma before appending the next feature
        if [[ -n "${json_features_string}" ]]; then
            json_features_string+=","
        fi
        # Append the feature, enclosed in double quotes
        json_features_string+="\"${feature}\""
    done

    cat << EOF > "${ROOTFS_INFO_FILE}"
{
    "imageUuid": "${IMAGE_UUID}",
    "customImageName": "${IMAGE_NAME}",
    "osRelease": {},
    "standardBuildConfig": {
        "openwrtRevision": "",
        "openwrtBuildTarget": "",
        "buildProfile": "",
        "deviceName": "${DEVICE_NAME}",
        "buildTargetPackages": [],
        "profilePackages": [],
        "supportedDevices": []
    },
    "routerFeatures": [${json_features_string}],
    "buildTime": "${CURRENT_TIME}Z",
    "crosOpenwrtImageBuilderVersion": "${IMAGE_BUILDER_VERSION}",
    "customIncludedFiles": {},
    "customPackages": {},
    "extraIncludedPackages": [],
    "excludedPackages": [],
    "disabledServices": [],
    "reservedInterfaces": []
}
EOF
    echo
}

build() {
    set -e

    LIB_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
    CROS_OPENWRT_DIR="${LIB_DIR}"/../..

    cd "${CROS_OPENWRT_DIR}"/image_generator

    local DOCKER_TAG="openwrt-builder:${DEVICE}-${OPENWRT_COMMIT}"

    local DOCKER_BUILD_ARGS=(
    -t "${DOCKER_TAG}"
    --build-arg OPENWRT_COMMIT="${OPENWRT_COMMIT}"
    --build-arg MTK_OPENWRT_FEEDS_COMMIT="${MTK_OPENWRT_FEEDS_COMMIT}"
    --build-arg DEVICE="${DEVICE}"
    )

    #Build Docker container with OpenWRT and MTK feed repos inside it
    docker build "${DOCKER_BUILD_ARGS[@]}" .
    START_TIME="$(date +%Y%m%d-%H%M%S)"
    BUILD_DIR="${CROS_OPENWRT_DIR}/build_dir/${DEVICE}/${IMAGE_NAME}/${START_TIME}"

    mkdir -p "${BUILD_DIR}"
    ln -sfn "${BUILD_DIR}" "$(dirname ${BUILD_DIR})/latest"

    generate_cros_files "${BUILD_DIR}"

    local DOCKER_RUN_ARGS=(
    -it
    --rm
    -v "${BUILD_DIR}/logs:/build/openwrt/logs"
    -v "${BUILD_DIR}/bin:/build/openwrt/bin"
    -v "${BUILD_DIR}/generated_files:/build/generated_files"
    "${DOCKER_TAG}"
    "/build/openwrt/run.sh"
    "${DEVICE}"
    "${IMAGE_PROFILE}"
    "BUILD_ARCHIVE"
    "DEBUG"
    )
    docker run "${DOCKER_RUN_ARGS[@]}"
}