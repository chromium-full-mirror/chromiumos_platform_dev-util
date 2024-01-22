#!/bin/bash
#
# Copyright 2020 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.


FINDMISSING_DIR="${HOME}/findmissing_workspace/findmissing"
LOG_FILE="/var/log/findmissing/$(basename -s .sh "$0").log"
LAST_RUN="/tmp/synchronize-lastrun"
RUNNING="/tmp/synchronize-running"

cd "${FINDMISSING_DIR}" || exit

if [[ ! -e env/bin/activate ]]; then
    echo "Virtual environment not set up."
    echo "Setting up virtual environment"
    python3 -m venv env

    # pip install requirements line by line
    env/bin/pip install --require-hashes -r requirements.txt
fi

day="$(date +%e)"
create="False"
if [[ ! -e "${LAST_RUN}" ]]; then
    create="True"
else
    last="$(cat "${LAST_RUN}")"
    if [[ "${last}" != "${day}" ]]; then
        create="True"
    fi
fi
echo "${day}" > "${LAST_RUN}"

{
    echo "Triggered full synchronization at $(date)"
    touch "${RUNNING}"
    env/bin/python3 -c "import main; main.synchronize_and_create_patches(${create})"
    rm "${RUNNING}"
    echo "End of full synchronization at $(date)"
} >> "${LOG_FILE}" 2>&1
