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
    * Documentation: go/cros-openwrt
    * Device: ${DEVICE_NAME}
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
        "buildProfile": "${IMAGE_PROFILE}",
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

export DEBUG_MODE=false
export CREATE_ARCHIVE=false

parse_args() {
    local arg

    # Loop through all arguments passed to this function
    while [[ "$#" -gt 0 ]]; do
        arg="$1" # Get the current argument

        case "${arg}" in
            --debug)
                DEBUG_MODE=true
                echo "DEBUG: Debug mode enabled."
                ;;
            --create_archive)
                CREATE_ARCHIVE=true
                echo "INFO: Archive creation enabled."
                ;;
            -*)
                echo "ERROR: Unknown option '${arg}'"
                exit 1
                ;;
            *)
                # Handle unexpected positional arguments if any
                # For this script, we don't expect any positional args after flags
                echo "ERROR: Unexpected argument '${arg}'"
                exit 1
                ;;
        esac
        shift
    done
}

build() {
    set -e

    parse_args "$@"

    LIB_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
    CROS_OPENWRT_DIR="${LIB_DIR}"/../..

    cd "${CROS_OPENWRT_DIR}"/image_generator

    local DOCKER_TAG="openwrt-builder:${IMAGE_PROFILE}-${OPENWRT_COMMIT}"

    local DOCKER_BUILD_ARGS=(
    -t "${DOCKER_TAG}"
    --build-arg OPENWRT_COMMIT="${OPENWRT_COMMIT}"
    --build-arg MTK_OPENWRT_FEEDS_COMMIT="${MTK_OPENWRT_FEEDS_COMMIT}"
    --build-arg DEVICE="${DEVICE}"
    )

    # Build Docker container with OpenWRT and MTK feed repos inside it
    docker build "${DOCKER_BUILD_ARGS[@]}" .
    START_TIME="$(date +%Y%m%d-%H%M%S)"
    BUILD_DIR="${CROS_OPENWRT_DIR}/build_dir/${DEVICE}/${IMAGE_NAME}/${START_TIME}"

    mkdir -p "${BUILD_DIR}"
    ln -sfn "${BUILD_DIR}" "$(dirname ${BUILD_DIR})/latest"

    # Pre-create the volume folders on the host so the Docker daemon doesn't create them as root.
    mkdir -p "${BUILD_DIR}/logs"
    mkdir -p "${BUILD_DIR}/bin"
    mkdir -p "${BUILD_DIR}/generated_files"

    # Make them globally writable so the container's internal 'openwrtuser' has clearance.
    chmod 777 "${BUILD_DIR}/logs" "${BUILD_DIR}/bin" "${BUILD_DIR}/generated_files"

    generate_cros_files "${BUILD_DIR}"

    # --- Debug Mode Container Lifecycle ---
    if [ "${DEBUG_MODE}" = true ]; then
        local CONTAINER_NAME="openwrt_debug_${DEVICE}"

        # Clean up any lingering container from an older aborted run
        docker rm -f "${CONTAINER_NAME}" 2>/dev/null || true

        echo "DEBUG MODE: Launching persistent container in background..."
        docker run -d \
            --name "${CONTAINER_NAME}" \
            -v "${BUILD_DIR}/logs:/build/openwrt/logs" \
            -v "${BUILD_DIR}/bin:/build/openwrt/bin" \
            -v "${BUILD_DIR}/generated_files:/build/generated_files" \
            "${DOCKER_TAG}" \
            tail -f /dev/null

        echo "DEBUG MODE: Executing build layer..."
        set +e  # Temporarily disable immediate script exit to catch the fail
        docker exec -it "${CONTAINER_NAME}" /build/openwrt/run.sh "${DEVICE}" "${IMAGE_PROFILE}" "${CREATE_ARCHIVE}" "${DEBUG_MODE}"
        BUILD_EXIT_CODE=$?
        set -e  # Re-enable error catching

        if [ ${BUILD_EXIT_CODE} -ne 0 ]; then
            echo "--------------------------------------------------------"
            echo "🔴 BUILD FAILED inside container (Exit Code: ${BUILD_EXIT_CODE})"
            echo "As requested via --debug, the container has been kept alive."
            echo "Run this command to drop into the shell right where it died:"
            echo ""
            echo "    docker exec -it ${CONTAINER_NAME} bash"
            echo ""
            echo "When done investigating, destroy the container using:"
            echo "    docker rm -f ${CONTAINER_NAME}"
            echo "--------------------------------------------------------"
            exit ${BUILD_EXIT_CODE}
        else
            echo "🟢 Debug Build Successful. Tearing down container environments."
            docker rm -f "${CONTAINER_NAME}"
        fi

    # --- Standard Mode Container Lifecycle (Original) ---
    else
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
        "${CREATE_ARCHIVE}"
        "${DEBUG_MODE}"
        )
        docker run "${DOCKER_RUN_ARGS[@]}"
    fi
}
