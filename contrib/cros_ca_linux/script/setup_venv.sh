#! /bin/bash
# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

if [ "${BASH_SOURCE[0]}" -ef "$0" ]; then
  echo "Error: This script is meant to be sourced, not executed!"
  echo "Please use 'source $0 <venv_inst_dir>' to run it."
  exit 1  # Exit with an error code
fi

script_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
project_dir=$(realpath "${script_dir}/..")
echo "CrOS CA project directory is ${project_dir}"

if [ -n "$1" ]; then
  venv_inst_dir=$(realpath "$1")
  echo "Virtual env installation directory is ${venv_inst_dir}"
else
  echo "Virtual env installation directory must be provided."
  echo "Usage: source setup_venv.sh <venv_inst_dir>"
  return
fi
# Exit the current venv just in case.
deactivate
if [ ! -d "${venv_inst_dir}" ]; then
  echo "Virtual env installation directory is not found: ${venv_inst_dir}"
  return
fi
if [[ -d "${venv_inst_dir}/venv" || -L "${venv_inst_dir}/venv" ]]; then
  echo "\"venv\" already exists under installation dir. Remove it."
  rm -rf "${venv_inst_dir}/venv"
fi
if [[ -d "${project_dir}/venv" || -L "${project_dir}/venv" ]]; then
  echo "\"venv\" already exists under project dir. Remove it."
  rm -rf "${project_dir}/venv"
fi
echo "Creating project virtual env..."
python3 -m venv "${venv_inst_dir}/venv"

if [ ! -f "${venv_inst_dir}/venv/bin/activate" ]; then
  echo "Error: virtual env is not ready: ${venv_inst_dir}/venv/bin/activate doesn't exist."
  return
fi
echo "Activating the virtual env..."
# shellcheck source=/dev/null
source "${venv_inst_dir}/venv/bin/activate"

if [ "${project_dir}" != "${venv_inst_dir}" ]; then
  echo "Creating symbolic link of venv to the project directory..."
  ln -s "${venv_inst_dir}/venv" "${project_dir}/venv"
fi

pip install --require-hashes --no-deps -r "${project_dir}/requirements.txt"

echo ""
echo "Project virtual environment is ready. Type \"deactivate\" to exit the virtual environment."
