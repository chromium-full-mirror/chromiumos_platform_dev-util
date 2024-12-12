#!/bin/bash
# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Script to build docker image for ChromeOS Mobly.

set -e

# Switch to script directory.
HERE="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
cd "${HERE}" || exit

# Keep temp files after script finishes.
KEEP_TMP_FILES=0

DOCKERFILE_MOBLY="Dockerfile_Mobly"

DOCKER_TAG_MOBLY="us-docker.pkg.dev/cros-registry/test-services/foil-test:prodMobly"

# Default docker container
DOCKER_BASE="gcr.io/dockerized-tradefed/tradefed:latest"

MOBLY_ARTIFACTS_BUCKET="gs://mobly_priv_artifacts"
MOBLY_ARTIFACTS_VERSION="out/billyzhao_mobly_tests_20241211_1122_RC00" # Make this a variable

CROS_TEST_BUILD_PATH="../../../../go.chromium.org/chromiumos/test/execution/cmd/cros-test/."
GOPATH_VALS=$(realpath -e "../../../../../../../../chroot/usr/lib/gopath/"):$(realpath -e "../../../../..")

PLATFORM_TOOLS_URL="https://dl.google.com/android/repository/platform-tools-latest-linux.zip"

while [[ $# -gt 0 ]]; do
  case $1 in
    -t|--tag)
      shift
      DOCKER_TAG="$1"
      shift
      ;;
    -b|--base)
      shift
      DOCKER_BASE="$1"
      shift
      ;;
    *)
      echo "Unknown option $1"
      exit 1
      ;;
  esac
done

TMPDIR_METADATA=$(mktemp -d -p . -t mobly_metadata_XXXXXXXXXX) || exit 1
TMPDIR_ARTIFACTS=$(mktemp -d -p . -t mobly_artifacts_XXXXXXXXXX) || exit 1

# Fetch Mobly artifacts
echo "Fetching Mobly artifacts from ${MOBLY_ARTIFACTS_BUCKET}/${MOBLY_ARTIFACTS_VERSION} to ${TMPDIR_ARTIFACTS}..."
gsutil -m cp -r "${MOBLY_ARTIFACTS_BUCKET}/${MOBLY_ARTIFACTS_VERSION}/*" "${TMPDIR_ARTIFACTS}"

# Move *.txtpb files to TMPDIR_METADATA
echo "Moving *.txtpb files to ${TMPDIR_METADATA}..."
find "${TMPDIR_ARTIFACTS}" -name "*.txtpb" -exec mv {} "${TMPDIR_METADATA}" \;

# Download platform tools (adb, etc.)
echo "Downloading platform tools..."
wget -q --retry-connrefused -O platform-tools.zip "${PLATFORM_TOOLS_URL}"
unzip -jq platform-tools.zip -d "${TMPDIR_ARTIFACTS}"
rm platform-tools.zip

# Build cros-test binary.
echo "Building cros-test binary..."
echo "Using GOPATH=${GOPATH_VALS}"
GOPATH="${GOPATH_VALS}" CGO_ENABLED=0 GO111MODULE=off go build "${CROS_TEST_BUILD_PATH}"
if [[ ! -f "./cros-test" ]]; then
  echo "Test binary [cros-test] not found or could not be built."
  exit 2
fi

# Build Docker image.
TAG=${DOCKER_TAG:-${DOCKER_TAG_MOBLY}}
echo "Building Mobly docker image: ${TAG}"
set -x
docker buildx build -f "${DOCKERFILE_MOBLY}" --no-cache=true -t "${TAG}" \
  --build-arg METADATA_DIR="${TMPDIR_METADATA}" \
  --build-arg ARTIFACT_DIR="${TMPDIR_ARTIFACTS}" \
  --build-arg BASE_CONTAINER="${DOCKER_BASE}" .
set +x

# Remove temporary files.
if [[ "${KEEP_TMP_FILES}" -eq 0 ]]; then
  rm -fR "${TMPDIR_METADATA}"
  rm -fR "${TMPDIR_ARTIFACTS}"
  rm ./cros-test
fi
