# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# TODO: Remove this file after the metric from test is handled in json
# Currently this file reads the metric from the test log, and extract the
# information it needs.

"""Process the log and write the information we are interested to a file"""

import datetime
import json
import os
import re
import sys

from cros_ca_lib.utils import logging as lg


logger = lg.logger


m_df = "Graphics.Smoothness.PercentDroppedFrames3.AllSequences"
m_kp = "EventLatency.KeyPressed.TotalLatency"
m_mp = "EventLatency.MousePressed.TotalLatency"
m_tt = "PageLoad.InteractiveTiming.TimeToNextPaint"
m_j3 = "Graphics.Smoothness.Jank3.AllSequences"
m_id = "PageLoad.InteractiveTiming.InputDelay3"
METRICS = [m_df, m_kp, m_mp, m_tt, m_j3, m_id]

total_sample_count = {}
metric_average = {}
metric_sum = {}
ux_bucket_max = {}
ux_bucket_min = {}

short_name = {
    m_df: "DF",
    m_kp: "KP",
    m_mp: "MP",
    m_tt: "TT",
    m_j3: "J3",
    m_id: "ID",
}

ux_buckets_count = {}
ux_buckets_high = {}

for mtc in METRICS:
    ux_buckets_count[mtc] = [0, 0, 0, 0, 0, 0]

ux_buckets_high[m_df] = [5, 10, 20, 40, 80, 90]
ux_buckets_high[m_kp] = [25000, 50000, 75000, 100000, 200000, 300000]
ux_buckets_high[m_mp] = [25000, 50000, 75000, 100000, 200000, 300000]
ux_buckets_high[m_tt] = [25, 50, 75, 100, 250, 500]
ux_buckets_high[m_j3] = [2, 5, 10, 20, 30, 100]
ux_buckets_high[m_id] = [5, 10, 20, 40, 80, 100]


def run(lines, energy, duration, filename):
    for m in METRICS:
        total_sample_count[m] = 0
        metric_average[m] = 0
        metric_sum[m] = 0
        ux_bucket_max[m] = 0
        ux_bucket_min[m] = 0

    output = []

    for line in lines:
        if "NotAvailble" in line:
            pass
        elif "Duration:" in line:
            # Skip the line for DecodedFrames for now.
            pass
        else:
            delimiter = "--"
            parts = line.split(delimiter)
            metric = parts[1].strip()
            # metric = {'histogram': {'buckets': [
            # {'count': 4, 'high': 3, 'low': 2},
            # {'count': 3, 'high': 4, 'low': 3},
            # {'count': 1, 'high': 5, 'low': 4},
            # {'count': 1, 'high': 17, 'low': 16}], 'count': 9,
            # 'name': 'Graphics.Smoothness.PercentDroppedFrames3.AllSequences',
            # 'sum': 37}}
            metric = metric.replace("'", '"')

            metric_name = None
            for m in METRICS:
                if m in parts[0]:
                    metric_name = m
            if not metric_name:
                # Metric that is not interesting to us.
                continue
            histogram = {"sum": 0}
            ux_buckets_per = [0, 0, 0, 0, 0, 0]
            if metric != "None":
                # Got the value for metric
                metric_dict = json.loads(metric)
                histogram = metric_dict["histogram"]
                buckets = histogram["buckets"]

                metric_name = histogram["name"]

                for bucket in buckets:
                    total_sample_count[metric_name] = (
                        total_sample_count[metric_name] + bucket["count"]
                    )

                    if bucket["low"] >= ux_buckets_high[metric_name][4]:
                        ux_buckets_count[metric_name][5] += bucket["count"]
                    elif bucket["low"] >= ux_buckets_high[metric_name][3]:
                        ux_buckets_count[metric_name][4] += bucket["count"]
                    elif bucket["low"] >= ux_buckets_high[metric_name][2]:
                        ux_buckets_count[metric_name][3] += bucket["count"]
                    elif bucket["low"] >= ux_buckets_high[metric_name][1]:
                        ux_buckets_count[metric_name][2] += bucket["count"]
                    elif bucket["low"] >= ux_buckets_high[metric_name][0]:
                        ux_buckets_count[metric_name][1] += bucket["count"]
                    else:
                        ux_buckets_count[metric_name][0] += bucket["count"]

                    # track min and max value per metric
                    if ux_bucket_max[metric_name] < bucket["high"]:
                        ux_bucket_max[metric_name] = bucket["high"]

                    if ux_bucket_min[metric_name] > bucket["low"]:
                        ux_bucket_min[metric_name] = bucket["low"]

                    # compute total sum of all buckets
                    metric_sum[metric_name] = metric_sum[metric_name] + (
                        (bucket["high"] + bucket["low"]) / 2 * bucket["count"]
                    )
                    # compute average
                    metric_average[metric_name] = (
                        metric_sum[metric_name]
                        / total_sample_count[metric_name]
                    )
                # compute average
                if total_sample_count[metric_name] == 0:
                    metric_average[metric_name] = 0
                else:
                    metric_average[metric_name] = (
                        histogram["sum"] / total_sample_count[metric_name]
                    )

                t_count = total_sample_count[metric_name]
                if t_count > 0:
                    for i in range(6):
                        ux_buckets_per[i] = (
                            ux_buckets_count[metric_name][i] / t_count
                        )

            metric_line = (
                "%s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s"
                % (
                    short_name[metric_name],
                    energy,
                    duration,
                    "%.3f" % metric_average[metric_name],
                    total_sample_count[metric_name],
                    histogram["sum"],
                    ux_bucket_min[metric_name],
                    ux_bucket_max[metric_name],
                    "%.2f" % ux_buckets_per[0],
                    "%.2f" % ux_buckets_per[1],
                    "%.2f" % ux_buckets_per[2],
                    "%.2f" % ux_buckets_per[3],
                    "%.2f" % ux_buckets_per[4],
                    "%.2f" % ux_buckets_per[5],
                    ux_buckets_count[metric_name][0],
                    ux_buckets_count[metric_name][1],
                    ux_buckets_count[metric_name][2],
                    ux_buckets_count[metric_name][3],
                    ux_buckets_count[metric_name][4],
                    ux_buckets_count[metric_name][5],
                    filename,
                )
            )
            output.append(metric_line)
    return output


def metric_process(energy, duration, directory):
    suffix = "test_local.log"
    matching_files = []
    for root, _, filenames in os.walk(directory):
        for filename in filenames:
            if filename.endswith(suffix):
                file_path = os.path.join(root, filename)
                matching_files.append(file_path)
    output = []
    pattern = (
        r"DecodedFrames|"
        + r"histogram snapshot \(after crosvideo playback - delta\)|"
        + r"histogram snapshot \(after typing - delta\)"
    )
    for f in matching_files:
        logger.debug("Processing %s", f)
        lines = []
        with open(f, "r", encoding="utf-8") as file:
            for line in file:
                if re.search(pattern, line):
                    lines.append(line)
        output += run(lines, energy, duration, f)
    return output


def main(argv):
    energy = argv[0]
    duration = argv[1]
    result_dir = argv[2]

    output = metric_process(energy, duration, result_dir)

    if len(output) > 0:
        now = datetime.datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
        filename = f"metrics_{now}.txt"

        # Create the file with appropriate permissions
        filepath = os.path.join(result_dir, filename)
        with open(filepath, "w", encoding="utf-8") as file:
            for item in output:
                file.write(f"{item}\n")
        print(f"Metrics file generated in {filepath}")
    else:
        print(f"No metrics found for {result_dir}")


if __name__ == "__main__":
    main(sys.argv[1:])
