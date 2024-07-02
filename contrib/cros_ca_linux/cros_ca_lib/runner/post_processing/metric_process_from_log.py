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
from pathlib import Path
import re
import sys

import numpy as np


# pylint: disable=banned-string-format-function
# pylint: disable=wrong-import-position
if __name__ == "__main__" and __package__ is None:
    # Add project root into the python path so lib importing works.
    sys.path.append(str(Path(__file__).resolve().parents[3]))

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
metric_data_points = {}
metric_average = {}
metric_p50 = {}
metric_p75 = {}
metric_p80 = {}
metric_p90 = {}
metric_p95 = {}
metric_p99 = {}
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


class GlogalInfo:
    """To store global info of the test."""

    perf_cmd_flag = "perfcmd_false"
    recorder_mode = "python"
    board = "NA"
    device_name = "NA"
    typing_delay = "0"
    sentence_cnt = "0"
    testname = "NA"
    total_duration_sec = 0.0
    total_decoded_frames = 0
    total_dropped_frames = 0
    cycles_cnt = "0"
    instructions_cnt = "0"
    cache_ref_cnt = "0"
    cache_miss_cnt = "0"
    bus_cycles_cnt = "0"
    page_fault_cnt = "0"
    build = "NA"


gi = GlogalInfo()


def run(lines, energy, duration, filename):
    for m in METRICS:
        total_sample_count[m] = 0
        metric_data_points[m] = []
        metric_average[m] = 0
        metric_p50[m] = 0
        metric_p75[m] = 0
        metric_p80[m] = 0
        metric_p90[m] = 0
        metric_p95[m] = 0
        metric_p99[m] = 0
        metric_sum[m] = 0
        ux_bucket_max[m] = 0
        ux_bucket_min[m] = 0

    output = []

    # looking for element from the end to avoid change in folder structure
    # Example of expected format:
    # ./craask_dut18/20240627-024407/tests/cuj.GoogleMeet.16p
    if "galnat" in filename:
        gi.board = "dedede"
        gi.device_name = "galnat"
    elif "casta" in filename or "hpstream14" in filename:
        gi.board = "octopus"
        gi.device_name = "casta"
    elif "craask" in filename:
        gi.board = "nissa"
        gi.device_name = "craask"
    else:
        logger.warning(
            "Unkown board and device from result dir: %s", filename
        )

    if "16p" in filename:
        gi.testname = "meet16p"
    elif "2p" in filename:
        gi.testname = "meet2p"
    elif "Video" in filename:
        gi.testname = "video"
    else:
        logger.warning("Unkown testname from result dir: %s", filename)

    # Note that there is an issue with log that sentence_cnt appears after all
    # the final metric delta. The workaround is to scan the log twice.
    # First scan of the lines (lines are info from the log)
    for line in lines:
        if "Final counter value after typing" in line:
            words = line.split()
            gi.sentence_cnt = words[-1]
        else:
            pass

    # Second scan of the lines
    for line in lines:
        if "NotAvailble" in line:
            pass
        elif "Duration:" in line:
            words = line.split()

            tmpStr = words[-7]
            tmpStr = tmpStr.replace("'", "").replace(",", "")
            duration_sec = float(tmpStr)

            tmpStr = words[-3]
            tmpStr = tmpStr.replace("'", "").replace(",", "")
            decoded_frames = float(tmpStr)

            tmpStr = words[-1]
            tmpStr = tmpStr.replace("'", "").replace(",", "")
            dropped_frames = float(tmpStr)

            gi.total_duration_sec += duration_sec
            gi.total_decoded_frames += decoded_frames
            gi.total_dropped_frames += dropped_frames
        elif "Running variables:" in line:
            print(f"Running variables line: {line}")
            words = line.split()
            index = 0
            for word in words:
                if "typing_delay" in word:
                    index += 1
                    typing_delay = words[index]
                    gi.typing_delay = typing_delay.replace("'", "").replace(
                        ",", ""
                    )
                    break
                else:
                    index += 1

        elif "Final counter value after typing" in line:
            # Note that processing of sentence_cnt is moved to
            # first scan of the lines
            pass
        elif "DUT name:" in line:
            if "CrOS" in line:
                gi.build = "cros"
            elif "Windows" in line:
                gi.build = "win11"
            else:
                logger.error("Unsupported dut name: %s", line)
                return
        else:
            delimiter = "--"
            parts = line.split(delimiter)
            if len(parts) < 2:
                continue
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

                    # Store all data point
                    for i in range(bucket["count"]):
                        metric_data_points[metric_name].append(bucket["low"])

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
                    # compute percentile for all data points
                    metric_p50[metric_name] = np.percentile(
                        metric_data_points[metric_name], 50
                    )
                    metric_p75[metric_name] = np.percentile(
                        metric_data_points[metric_name], 75
                    )
                    metric_p80[metric_name] = np.percentile(
                        metric_data_points[metric_name], 80
                    )
                    metric_p90[metric_name] = np.percentile(
                        metric_data_points[metric_name], 90
                    )
                    metric_p95[metric_name] = np.percentile(
                        metric_data_points[metric_name], 95
                    )
                    metric_p99[metric_name] = np.percentile(
                        metric_data_points[metric_name], 99
                    )

                t_count = total_sample_count[metric_name]
                if t_count > 0:
                    for i in range(6):
                        ux_buckets_per[i] = (
                            ux_buckets_count[metric_name][i] / t_count
                        )

            # Parameter typing_delay and sentence_cnt are only meaning full for
            # meet test
            if gi.total_decoded_frames != 0:
                gi.typing_delay = "0"
                gi.sentence_cnt = "0"

            metric_line = (
                "%s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s"
                + " %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s"
                + " %s %s %s %s"
            ) % (
                short_name[metric_name],
                gi.recorder_mode,
                gi.device_name,
                gi.board,
                gi.build,
                gi.testname,
                energy,
                duration,
                gi.typing_delay,
                gi.sentence_cnt,
                0,
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
                "{:.1f}".format(gi.total_decoded_frames),
                "{:.1f}".format(gi.total_dropped_frames),
                gi.cycles_cnt,
                gi.instructions_cnt,
                gi.cache_ref_cnt,
                gi.cache_miss_cnt,
                gi.bus_cycles_cnt,
                gi.page_fault_cnt,  # print("matching lines: ", line)
                "{:.2f}".format(metric_p50[metric_name]),
                "{:.2f}".format(metric_p75[metric_name]),
                "{:.2f}".format(metric_p80[metric_name]),
                "{:.2f}".format(metric_p90[metric_name]),
                "{:.2f}".format(metric_p95[metric_name]),
                "{:.2f}".format(metric_p99[metric_name]),
                filename,
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
        + r"DUT name:|"
        + r"Running variables:|"
        + r"Final counter value after typing|"
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
    if len(argv) > 3:
        processing_option = argv[3]

    # Determine perfcmd_flag, recorder_mode, device_name, and testname
    # from the result_dri
    parts = result_dir.rstrip("/").split("/")

    # Remove test_local.log if it is part the string
    # ./craask_dut18/20240627-024407/tests/cuj.GoogleMeet.16p/test_local.log
    if "test_local.log" in parts:
        parts.remove("test_local.log")
        result_dir = "/".join(parts)

    output = metric_process(energy, duration, result_dir)

    if "log_only" in processing_option:
            for item in output:
                print(item)
    else:
        if len(output) > 0:
            now = datetime.datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
            filename = f"metrics_{now}.txt"

            # Create the file with appropriate permissions
            filepath = os.path.join(result_dir, filename)
            with open(filepath, "w", encoding="utf-8") as file:
                for item in output:
                    file.write(f"{item}\n")
            logger.info("Metrics file generated in %s", filepath)
        else:
            logger.info("No metrics found for %s", result_dir)


if __name__ == "__main__":
    main(sys.argv[1:])
