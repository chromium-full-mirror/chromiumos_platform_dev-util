#!/bin/bash
# Copyright 2025 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

set -e

DEVICE_CONFIG_DIR="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"
OPENWRT_DIR=$(realpath -e "${DEVICE_CONFIG_DIR}/../..")

cd "${OPENWRT_DIR}"
exec > >(tee ./logs/run.log) 2>&1

echo "Start feeds update and install"
./scripts/feeds update -a
./scripts/feeds install -a

echo "Apply config"
cp "${DEVICE_CONFIG_DIR}/defconfig" "${OPENWRT_DIR}/.config"

echo "Remove z_cros_test.sh"
rm "${OPENWRT_DIR}/files/etc/init.d/z_cros_test.sh"

make defconfig

echo "Run make download"
make -j"$(nproc)" V=s download

echo "Run make world -j=$(nproc) V=s"
make -j"$(nproc)" V=s world || (echo "Rerun make -j1 V=sc world" && make -j1 V=sc world)

echo "Images successfully generated"
