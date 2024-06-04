#!/bin/bash
# shellcheck disable=SC2248
# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
set -e

# -- Benchmarking pacina -- #
#
# This script generates benchmarks to assess the capabilities under a number of
# requested sampling frequencies and SPS (Samples Per Second) configurations.
# For each SPS x Hz config, it takes 10_000 measurements and saves the results
# to a timeLog.csv in a directory structure depicted below. Currently, it does
# not benchmark w.r.t to different voltage / power supply, as the goal was to
# asssess measurement time and sampling time variance and reliability.
#
# The following directories structure is generated:
#
# results
# ├── SPS_1024
# │     ├── 128hz
# │     ├── 16hz
# │     ├── 256hz
# │     ├── 32hz
# │     └── 64hz
# └── SPS_2560
#     ├── 128hz
#     ├── 16hz
#     ├── 256hz
#     ├── 32hz
#     └── 64hz
#
# See "Selecting frequencies" section of go/pac-power-measurement
# for a detailed explanation of the selected numbers
#
# in seconds
sample_times=(
  "0.00390625"
  "0.0078125"
  "0.015625"
  "0.03125"
  "0.0625"
)
output_dirs=(
  "256hz"
  "128hz"
  "64hz"
  "32hz"
  "16hz"
)
fast_modes=(
  "true"
  "false"
)

for fast_mode in "${fast_modes[@]}"; do
    for i in "${!sample_times[@]}"; do
      sample_time=${sample_times[${i}]}
      output_dir=${output_dirs[${i}]}

      if [[ "${fast_mode}" == "true" ]]; then
        results_dir="results/SPS_2560"
        no_fast_mode_flag=""
      else
        results_dir="results/SPS_1024"
        no_fast_mode_flag="--no-fast-mode"
      fi

      echo "Running pacina with sample-time=${sample_time} and output-dir=${results_dir}/${output_dir}"

      # The following actions were taken to ensure timely fashion of measurements:
      # - Process scheduling policy was set to SCHED_FIFO (-F)
      # - Priority is set to a high value (-p)
      # - And we pinned the process CPU affinity (-a)
      # https://manpages.ubuntu.com/manpages/trusty/man8/schedtool.8.html
      sudo schedtool \
          -F \
          -p 70 \
          -a 0x2 \
          -e \
          ./run.sh \
          -m 10000 \
          --output "${results_dir}" \
          --sample-time "${sample_time}" \
          --configs cpd.py \
          --dir "${output_dir}" \
          --ftdi-urls ftdi:///2 \
          ${no_fast_mode_flag}
    done
done

echo "Pacina benchmarks completed successfully."
