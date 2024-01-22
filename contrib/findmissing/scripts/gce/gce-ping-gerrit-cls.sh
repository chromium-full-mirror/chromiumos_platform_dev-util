#!/bin/bash
#
# Copyright 2022 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.


FINDMISSING="$(cd "$(dirname "$0")/../.." || exit; pwd)"
cd "${FINDMISSING}" || exit
LOG_FILE="/var/log/findmissing/$(basename -s .sh "$0").log"

if [[ ! -e env/bin/activate ]]; then
    echo "Virtual environment not set up."
    echo "Setting up virtual environment"
    python3 -m venv env

    # pip install requirements line by line
    env/bin/pip install --require-hashes -r requirements.txt
fi

{
    echo "Triggered ping Gerrit CLs at $(date)"
    env/bin/python3 -u ping.py
    echo "End of ping Gerrit CLs at $(date)"
} >> "${LOG_FILE}" 2>&1
