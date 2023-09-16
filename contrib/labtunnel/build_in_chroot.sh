#!/bin/bash
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Configure project go environment, runs tests, and builds executable.
# Meant to be run from within chroot.

set -e

PROJECT_EXE_NAME="$1"
SCRIPT_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
PROJECT_GOPATH="${SCRIPT_DIR}/go"
PROJECT_EXE_PATH="${PROJECT_GOPATH}/bin/${PROJECT_EXE_NAME}"

# Configure go environment.
export GOPATH="${PROJECT_GOPATH}"
export GO111MODULE=on

set -x

# Get packages, format files, run tests, and build executable.
cd "${PROJECT_GOPATH}/src" && go mod tidy
cd "${PROJECT_GOPATH}/src" && go fmt ./...
cd "${PROJECT_GOPATH}/src" && go test ./...
cd "${PROJECT_GOPATH}/src" && go build -o "${PROJECT_EXE_PATH}"
