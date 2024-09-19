#!/usr/bin/env python
# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Compares the metrics of BSM and baseline CUJs results-chart.json.

It takes a directory path as an argument and expects to find two folders
within the directory, one named "baseline" and one named "battery_saver".
Each folder should contain a results-chart.json file.
The script then extracts the specified metrics from each file and
generates a markdown table comparing the two sets of metrics.
"""

import argparse
import json
import os


# pylint: disable=line-too-long
def extract_metrics(filepath):
    """Extracts the specified metrics from a results-chart.json file.

    Args:
        filepath: Path to the results-chart.json file.

    Returns:
        A dictionary of metric names to their values.
    """
    with open(filepath, "r", encoding="utf-8") as f:
        data = json.load(f)
    metrics = {
        "TPS.Power.usage": data.get("TPS.Power.usage", {})
        .get("summary", {})
        .get("value"),
        "Graphics.Smoothness.PercentDroppedFrames3.AllSequences.average": data.get(
            "Graphics.Smoothness.PercentDroppedFrames3.AllSequences", {}
        )
        .get("average", {})
        .get("value"),
        "EventLatency.MousePressed.TotalLatency.average": data.get(
            "EventLatency.MousePressed.TotalLatency", {}
        )
        .get("average", {})
        .get("value"),
        "EventLatency.KeyPressed.TotalLatency.average": data.get(
            "EventLatency.KeyPressed.TotalLatency", {}
        )
        .get("average", {})
        .get("value"),
        "PageLoad.PaintTiming.NavigationToFirstContentfulPaint.average": data.get(
            "PageLoad.PaintTiming.NavigationToFirstContentfulPaint", {}
        )
        .get("average", {})
        .get("value"),
        "Power.MinutesBatteryLife": data.get("Power.MinutesBatteryLife", {})
        .get("summary", {})
        .get("value"),
    }
    return metrics


def generate_markdown_table(baseline_metrics, test_metrics, cuj_name):
    """Generates a markdown table comparing baseline and test metrics.

    Args:
        baseline_metrics: A dictionary of baseline metric names to their values.
        test_metrics: A dictionary of test metric names to their values.
        cuj_name: The name of the CUJ.

    Returns:
        A string containing the markdown table.
    """
    table = f"| metric | {cuj_name} value | {cuj_name} battery saver value |\n"
    table += "|---|---|---|\n"
    for metric in baseline_metrics:
        table += f"| {metric} | {baseline_metrics[metric]} | {test_metrics[metric]} |\n"
    return table


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "-d",
        "--directory",
        required=True,
        help="Path to the folder tree which contains the CUJ folders and results-chart.json files.",
    )
    args = parser.parse_args()

    cujs = ["ui.MeetCUJ", "ui.DesksCUJ", "ui.VideoCUJ"]
    results = {}

    for root, _, files in os.walk(args.directory):
        for file in files:
            if file == "results-chart.json":
                for cuj in cujs:
                    if cuj in root:
                        if "battery_saver" in root:
                            results.setdefault(cuj, {}).setdefault(
                                "battery_saver", root
                            )
                        else:
                            results.setdefault(cuj, {}).setdefault(
                                "baseline", root
                            )

    for cuj in cujs:
        if "baseline" in results[cuj] and "battery_saver" in results[cuj]:
            baseline_metrics = extract_metrics(
                os.path.join(results[cuj]["baseline"], "results-chart.json")
            )
            test_metrics = extract_metrics(
                os.path.join(
                    results[cuj]["battery_saver"], "results-chart.json"
                )
            )
            print(generate_markdown_table(baseline_metrics, test_metrics, cuj))


if __name__ == "__main__":
    main()
