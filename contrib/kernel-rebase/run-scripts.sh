#!/bin/bash
# Copyright 2021 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

progdir=$(dirname "$0")
cd "${progdir}" || exit 1

if [[ ! -d env ]]; then
    python3 -m venv env
    env/bin/pip -q install google_auth_oauthlib python-Levenshtein fuzzywuzzy rich google-api-python-client sh
fi

source "env/bin/activate"

baseline="$(python3 -c 'from config import rebase_baseline_branch; print(rebase_baseline_branch)')"
target="$(python3 -c 'from config import rebase_target; print(rebase_target)')"

logfile="/tmp/run-scripts.${baseline}-to-${target}.$(date +%F).log"

./setup.sh > "${logfile}" 2>&1
./genspreadsheet.py >> "${logfile}" 2>&1
./genstats.py >> "${logfile}" 2>&1
