#!/bin/bash
# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

ScriptsPath=$(dirname "$0")
echo "${ScriptsPath}"
tar_dut() {
    tar -cvzf "${ScriptsPath}/../win_config.tgz" -C "${ScriptsPath}/win/" "config"
}

tar_dut
