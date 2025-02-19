#!/bin/bash
# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Script to build docker image for ChromeOS AOSP Tradefed.

set -e

# Switch to script directory.
HERE="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
cd "${HERE}" || exit

# What docker image to build: 0: AOSP TF, 1: internal TF
INTERNAL_TF=0

# To use the preset GOPATH or build it
MYGOPATH=0

# Keep temp files after script finishes.
KEEP_TMP_FILES=0

TRADEFED_ZIP="tradefed.zip"

DOCKERFILE_AOSP="Dockerfile_TF_AOSP"
DOCKERFILE_INTERNAL="Dockerfile_AL_INTERNAL"

DOCKER_TAG_AOSP="us-docker.pkg.dev/cros-registry/test-services/cros-test:prodAospTF"
DOCKER_TAG_INTERNAL="us-docker.pkg.dev/cros-registry/test-services/cros-test:prodTF"

TEST_FINDER_TAG_INTERNAL="us-docker.pkg.dev/cros-registry/test-services/cros-test-finder:ALTF"

TEST_FINDER_FILE="Dockerfile_Combined_Finder"



METADATA_BUCKET="gs://cros-xts-metadata"
METADATA_COMPILE_SCRIPT="../../python/src/tools/compile_xts_metadata.py"

CROS_TEST_BUILD_PATH="../../../../go.chromium.org/chromiumos/test/execution/cmd/cros-test/."
CROS_TEST_FINDER_BUILD_PATH="../../../../go.chromium.org/chromiumos/test/test_finder/cmd/cros-test-finder/."


PLATFORM_TOOLS_URL="https://dl.google.com/android/repository/platform-tools-latest-linux.zip"

function download_artifact () {
  local SUITE=$1
  local BUILD=$2
  local TARGET=$3

  local ZIP_FILE="android-${SUITE}.zip"

  echo "Downloading ${ZIP_FILE} for build ${BUILD} to ${TMPDIR_ARTIFACTS}..."
  set -x
  /google/data/ro/projects/android/fetch_artifact --bid "${BUILD}" \
    --target "${TARGET}" "${ZIP_FILE}" "${TMPDIR_ARTIFACTS}"
  set +x
  if [[ ! -f "${TMPDIR_ARTIFACTS}/${ZIP_FILE}" ]]; then
    echo "Error fetching artifact: ${TMPDIR_ARTIFACTS}/${ZIP_FILE}"
    exit 2
  fi

  # Unpack the test archive (only the needed files).
  echo "Unpacking test cases from ${TMPDIR_ARTIFACTS}/${ZIP_FILE}..."
  BASE_DIR="$(basename "${ZIP_FILE}" .zip)"
  unzip -q "${TMPDIR_ARTIFACTS}/${ZIP_FILE}" \
    "${BASE_DIR}/testcases/*" \
    "${BASE_DIR}/tools/${SUITE}-tradefed.jar" \
    -d "${TMPDIR_ARTIFACTS}"
  if [[ ! -d "${TMPDIR_ARTIFACTS}/${BASE_DIR}" ]]; then
    echo "Error unpacking artifact: ${TMPDIR_ARTIFACTS}/${ZIP_FILE}"
    exit 2
  fi
  mv "${TMPDIR_ARTIFACTS}/${BASE_DIR}/tools/${SUITE}-tradefed.jar" "${TMPDIR_ARTIFACTS}/"
  rm "${TMPDIR_ARTIFACTS}/${ZIP_FILE}"
  rm -fR "${TMPDIR_ARTIFACTS}/${BASE_DIR}/tools"
}

if [[ ! -f "./${DOCKERFILE_AOSP}" && ! -f "./${DOCKERFILE_INTERNAL}" ]]; then
  echo "Tradefed docker file must be present in the current folder."
  exit 2
fi

METADATA=()
while [[ $# -gt 0 ]]; do
  case $1 in
    -m|--metadata)
      shift
      METADATA+=("$1")
      shift
      ;;
    -i|--internal)
      INTERNAL_TF=1
      echo "Building internal TF docker..."
      shift
      ;;
    -k|--keep-tmp-files)
      KEEP_TMP_FILES=1
      shift
      ;;
    -t|--tag)
      shift
      DOCKER_TAG="$1"
      shift
      ;;
    -p|--MYGOPATH)
      MYGOPATH=1
      shift
      ;;
    *)
      echo "Unknown option $1"
      exit 1
      ;;
  esac
done

# ###############################################################################
# Handle Tradefed metadata and tests
# ###############################################################################

if [[ "${MYGOPATH}" -eq 0 ]]; then
  echo "configuring gopath for you"
  GOPATH_VALS=$(realpath -e "../../../../../../../../chroot/usr/lib/gopath/"):$(realpath -e "../../../../..")
else
  echo "using preset gopath"
  GOPATH_VALS=${GOPATH:?}
fi

echo "Using ${GOPATH_VALS} for gopath"

TMPDIR_METADATA=$(mktemp -d -p . -t tf_metadata_XXXXXXXXXX) || exit 1
TMPDIR_ARTIFACTS=$(mktemp -d -p . -t tf_artifacts_XXXXXXXXXX) || exit 1
if [[ "${INTERNAL_TF}" -eq 0 ]]; then
  TMPDIR_TF=$(mktemp -d -p . -t tf_dir_XXXXXXXXXX) || exit 1
fi

if [[ "${#METADATA[@]}" -eq 0 ]]; then
  echo "Metadata arguments (--metadata FILE1 --metadata FILE2...) not provided."
  exit 2
fi

SUITES=()
for i in "${METADATA[@]}"; do
  # Fetch and compile metadata
  PB_NAME="$(basename "${i}" .json).pb"
  DIGEST_NAME="$(basename "${i}" .json)-digest.json"
  echo "Fetching metadata ${i} to ${TMPDIR_METADATA}..."
  gsutil -q cp "${METADATA_BUCKET}/${i}" "${TMPDIR_METADATA}"
  jq '{suite, branch, build_id, target: .targets[] | select(index("x86")) }' \
    "${TMPDIR_METADATA}/${i}" > "${TMPDIR_METADATA}/${DIGEST_NAME}"
  echo "Compiling metadata ${i}..."
  "${METADATA_COMPILE_SCRIPT}" --output_file "${TMPDIR_METADATA}/${PB_NAME}" "${TMPDIR_METADATA}/${i}"

  # Fetch and unpack test suite artifact.
  set +e
  # shellcheck disable=SC2002
  read -d "\n" -r SUITE BRANCH BUILD TARGET \
    <<<"$(cat "${TMPDIR_METADATA}/${DIGEST_NAME}" | jq -r '.[]')"
  set -e
  echo "Metadata suite=${SUITE} branch=${BRANCH} bid=${BUILD} target=${TARGET}"
  if [[ "${INTERNAL_TF}" -eq 0 ]]; then
    download_artifact "${SUITE}" "${BUILD}" "${TARGET}"
  fi
  SUITES+=("${SUITE}")
done

echo "Test suites to be included in docker image: [${SUITES[*]}]..."

# Fetch and unpack tradefed console.
if [[ "${INTERNAL_TF}" -eq 0 ]]; then
  echo "Downloading Tradefed console for build ${BUILD} and target ${TARGET}..."
  set -x
  /google/data/ro/projects/android/fetch_artifact --bid "${BUILD}" \
    --target "${TARGET}" "${TRADEFED_ZIP}" "${TMPDIR_TF}"
  set +x
  echo "Unpacking Tradefed console to ${TMPDIR_TF}..."
  unzip -q "${TMPDIR_TF}/${TRADEFED_ZIP}" -d "${TMPDIR_TF}"
  rm "${TMPDIR_TF}/${TRADEFED_ZIP}"
  # Download platform tools (adb, etc.) to the tradefed dir.
  wget -q --retry-connrefused -O platform-tools.zip "${PLATFORM_TOOLS_URL}"
  unzip -jq platform-tools.zip -d "${TMPDIR_TF}"
  rm platform-tools.zip
fi


# ###############################################################################
# Handle Mobly GRTE dependency
# ###############################################################################

MOBLY_ARTIFACTS_BUCKET="gs://mobly_priv_artifacts"
MOBLY_ARTIFACTS_VERSION="grte" # Make this a variable

# Fetch Mobly artifacts.
echo "Fetching Mobly artifacts from ${MOBLY_ARTIFACTS_BUCKET}/${MOBLY_ARTIFACTS_VERSION} to ${TMPDIR_ARTIFACTS}..."
gsutil -m cp -r "${MOBLY_ARTIFACTS_BUCKET}/${MOBLY_ARTIFACTS_VERSION}/*" "${TMPDIR_ARTIFACTS}"

# ###############################################################################
# Handle Docker image
# ###############################################################################

# Build cros-test binary.
echo "Building cros-test binary..."
echo "Using GOPATH=${GOPATH_VALS}"
GOPATH="${GOPATH_VALS}" CGO_ENABLED=0 GO111MODULE=off go build "${CROS_TEST_BUILD_PATH}"
if [[ ! -f "./cros-test" ]]; then
  echo "Test binary [cros-test] not found or could not be built."
  exit 2
fi
GOPATH="${GOPATH_VALS}" CGO_ENABLED=0 GO111MODULE=off go build "${CROS_TEST_FINDER_BUILD_PATH}"
if [[ ! -f "./cros-test" ]]; then
  echo "Test binary [cros-test] not found or could not be built."
  exit 2
fi


# Build Docker images.
if [[ "${INTERNAL_TF}" -eq 0 ]]; then
  TAG=${DOCKER_TAG:-${DOCKER_TAG_AOSP}}
  echo "Building AOSP docker image: ${TAG}"
  set -x
  docker buildx build -f "${DOCKERFILE_AOSP}" --no-cache=true -t "${TAG}" \
    --build-arg TRADEFED_DIR="${TMPDIR_TF}" \
    --build-arg METADATA_DIR="${TMPDIR_METADATA}" \
    --build-arg ARTIFACT_DIR="${TMPDIR_ARTIFACTS}" .
  set +x
else
  TAG=${DOCKER_TAG:-${DOCKER_TAG_INTERNAL}}
  echo "Building docker image: ${TAG}"
  set -x
  docker buildx build -f "${DOCKERFILE_INTERNAL}" --no-cache=true -t "${TAG}" \
    --build-arg METADATA_DIR="${TMPDIR_METADATA}" \
    --build-arg ARTIFACT_DIR="${TMPDIR_ARTIFACTS}" .

  set +x
fi

set -x
docker buildx build -f "${TEST_FINDER_FILE}" --no-cache=true -t "${TEST_FINDER_TAG_INTERNAL}" \
    --build-arg METADATA_DIR="${TMPDIR_METADATA}" .
set +x


# Remove temporary files.
if [[ "${KEEP_TMP_FILES}" -eq 0 ]]; then
  rm -fR "${TMPDIR_METADATA}"
  if [[ "${INTERNAL_TF}" -eq 0 ]]; then
    rm -fR "${TMPDIR_ARTIFACTS}"
    rm -fR "${TMPDIR_TF}"
  fi
  rm ./cros-test
  rm ./cros-test-finder
fi
