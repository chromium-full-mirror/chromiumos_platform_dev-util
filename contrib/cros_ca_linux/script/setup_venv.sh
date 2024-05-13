#! /bin/bash
# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

if [ -n "$1" ]; then
  project_dir=$(realpath "$1")
  echo "Project directory is ${project_dir}"
else
  echo "Project directory must be provided."
  echo "Usage: source setup_venv.sh <project_dir>"
  return
fi

cd "${project_dir}" || { echo "Project directory is not found: ${project_dir}"; return; }

if [ -f "${project_dir}/venv/bin/activate" ]; then
  echo "Virtual env already exists. Remove it."
  rm -rf "${project_dir}/venv/"
fi
echo "Creating project virtual env..."
python3 -m venv "${project_dir}/venv"

if [ ! -f "${project_dir}/venv/bin/activate" ]; then
  echo "Error: virtual env is not ready: ${project_dir}/venv/bin/activate doesn't exist."
  return
fi
echo "Activating the virtual env..."
# shellcheck source=/dev/null
source "${project_dir}/venv/bin/activate"

pip install --require-hashes -r "${project_dir}/requirements.txt"

echo ""
echo "Project virtual environment is ready. Type \"deactivate\" to exit the virtual environment."
