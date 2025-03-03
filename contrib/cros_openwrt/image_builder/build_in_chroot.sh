#!/bin/bash
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Configure project go environment, runs tests, and builds executable.
# Meant to be run from within chroot.

set -e

PROJECT_EXE_NAME="$1"
GO_VERSION="go1.23.1"

SCRIPT_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
PROJECT_GOPATH="${SCRIPT_DIR}/go"
PROJECT_EXE_PATH="${PROJECT_GOPATH}/bin/${PROJECT_EXE_NAME}"
GOBIN="${GOBIN:-"${PROJECT_GOPATH}/bin"}"
GO_CMD="${GOBIN}/${GO_VERSION}"

# Configure project go version.
if ! command -v "${GO_CMD}"; then
  export GOBIN
  go install "golang.org/dl/${GO_VERSION}@latest"
  "${GO_CMD}" download
fi

# Configure go environment.
export GOPATH="${PROJECT_GOPATH}"
export GO111MODULE=on

set -x

"${GO_CMD}" version
# Get packages, format files, run tests, and build executable.
cd "${PROJECT_GOPATH}/src" && "${GO_CMD}" mod tidy
cd "${PROJECT_GOPATH}/src" && "${GO_CMD}" fmt ./...
cd "${PROJECT_GOPATH}/src" && "${GO_CMD}" test ./...
cd "${PROJECT_GOPATH}/src" && "${GO_CMD}" build -o "${PROJECT_EXE_PATH}"
