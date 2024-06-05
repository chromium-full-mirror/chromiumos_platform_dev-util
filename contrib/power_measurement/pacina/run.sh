#!/bin/bash
# shellcheck disable=SC2068
# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
set -e
python3 -m pacina.cli $@
