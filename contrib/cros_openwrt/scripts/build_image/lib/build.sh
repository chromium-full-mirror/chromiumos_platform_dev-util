#!/bin/bash
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Paths vars.
SCRIPT_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
CROS_OPENWRT_DIR="${SCRIPT_DIR}/../../.."
BUILD_DIR="${CROS_OPENWRT_DIR}/build"
DO_BUILD='y'
BUILD_SUBCOMMAND='all'
source "${SCRIPT_DIR}/builder_options.sh"

function compile_builder() {
  echo "Compiling cros_openwrt_image_builder"
  bash "${CROS_OPENWRT_DIR}/image_builder/build.sh"
}

function validate_build_vars() {
  if [ "${image_profile}" = "" ]; then
    echo "Missing required build var 'image_profile'"
    exit 1
  fi
  if [ "${sdk_url}" = "" ]; then
    echo "Missing required build var 'sdk_url'"
    exit 1
  fi
  if [ "${image_builder_url}" = "" ]; then
    echo "Missing required build var 'image_builder_url'"
    exit 1
  fi
  if [ "${extra_image_name}" = "" ]; then
    echo "Missing required build var 'extra_image_name'"
    exit 1
  fi
}

function build() {
  set -e

  # Prepare for build.
  validate_build_vars
  compile_builder
  mkdir -p "${BUILD_DIR}"

  # Finalize arguments based on options.
  local BUILDER_ARGS=(
    build
    "${BUILD_SUBCOMMAND}"
    --working_dir "${BUILD_DIR}/${image_profile}/cros_openwrt"
    --sdk_url "${sdk_url}"
    --image_builder_url "${image_builder_url}"
    --image_profile "${image_profile}"
    --extra_image_name "${extra_image_name}"
  )
  for VAL in "${image_feature[@]}"; do
    local BUILDER_ARGS+=(--image_feature "${VAL}")
  done
  for VAL in "${sdk_config[@]}"; do
    local BUILDER_ARGS+=(--sdk_config "${VAL}")
  done
  for VAL in "${sdk_make[@]}"; do
    local BUILDER_ARGS+=(--sdk_make "${VAL}")
  done
  for VAL in "${include_custom_package[@]}"; do
    local BUILDER_ARGS+=(--include_custom_package "${VAL}")
  done
  for VAL in "${include_official_package[@]}"; do
    local BUILDER_ARGS+=(--include_official_package "${VAL}")
  done
  for VAL in "${exclude_package[@]}"; do
    local BUILDER_ARGS+=(--exclude_package "${VAL}")
  done
  for VAL in "${disable_service[@]}"; do
    local BUILDER_ARGS+=(--disable_service "${VAL}")
  done
  for VAL in "${EXTRA_ARGS[@]}"; do
    local BUILDER_ARGS+=("${VAL}")
  done

  # Print and run build command.
  echo "Building standard OpenWRT image for profile ${image_profile} in ${BUILD_DIR}"
  echo cros_openwrt_image_builder "${BUILDER_ARGS[@]}"
  if [ "${DO_BUILD}" = "y" ]; then
    cros_openwrt_image_builder "${BUILDER_ARGS[@]}"
  fi
}
