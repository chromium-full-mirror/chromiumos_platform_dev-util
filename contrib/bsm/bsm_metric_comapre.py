#!/usr/bin/env python
# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Compares BSM and baseline CUJs results-chart.json metrics.

Expects a directory with "baseline" and "battery_saver" subfolders,
each containing a results-chart.json.  Generates a markdown table
comparing specified metrics.
"""

import argparse
import json
import pathlib


POWER_METRICS = frozenset(["TPS.Power.usage", "Power.MinutesBatteryLife"])
CUJS = frozenset(["ui.MeetCUJ", "ui.DesksCUJ", "ui.VideoCUJ"])


def extract_metric(data, metric_name):
    """Extracts a single metric from results-chart.json data."""
    metric = data.get(metric_name, {}).get(
        "summary" if metric_name in POWER_METRICS else "average", {}
    )
    return {
        "value": metric.get("value"),
        "units": metric.get("units"),
        "direction": metric.get("improvement_direction"),
    }


def extract_metrics(filepath):
    """Extracts specified metrics from a results-chart.json file."""
    with open(filepath, "r", encoding="utf-8") as f:
        data = json.load(f)

    metrics = {}
    for metric_name in (
        "TPS.Power.usage",
        "Graphics.Smoothness.PercentDroppedFrames3.AllSequences",
        "EventLatency.MousePressed.TotalLatency",
        "EventLatency.KeyPressed.TotalLatency",
        "PageLoad.PaintTiming.NavigationToFirstContentfulPaint",
        "Power.MinutesBatteryLife",
    ):
        metrics[metric_name] = extract_metric(data, metric_name)
    return metrics


def is_power_saving(baseline, bsm, metric_name):
    """Checks if BSM shows power saving compared to baseline."""
    if metric_name not in POWER_METRICS:
        return ""

    if baseline["direction"] == "up":
        return "OK" if bsm["value"] > baseline["value"] else "FAIL"
    if baseline["direction"] == "down":
        return "OK" if bsm["value"] < baseline["value"] else "FAIL"
    return ""


def generate_markdown_table(baseline_metrics, test_metrics, cuj_name):
    """Generates a markdown comparison table."""
    # pylint: disable=line-too-long
    table = f"| metric | {cuj_name} value | {cuj_name} battery saver value | power saving |\n"
    table += "|---|---|---|---|\n"
    for metric, baseline in baseline_metrics.items():
        bsm = test_metrics[metric]
        power_saving = is_power_saving(baseline, bsm, metric)
        table += f"| {metric} ({baseline['units']}) | {baseline['value']:.2f} | {bsm['value']:.2f} | {power_saving} |\n"
    return table


def find_results(directory):
    """Finds results-chart.json files for each CUJ."""
    results = {}
    for path in pathlib.Path(directory).rglob("results-chart.json"):
        for cuj in CUJS:
            if cuj in str(path):
                mode = (
                    "battery_saver"
                    if "battery_saver" in str(path)
                    else "baseline"
                )
                results.setdefault(cuj, {})[mode] = path
                break  # Assume only one matching CUJ per path
    return results


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "-d",
        "--directory",
        required=True,
        help="Path to the folder tree which contains the CUJ \
               folders and results-chart.json files.",
    )
    args = parser.parse_args()

    results = find_results(args.directory)

    for cuj, paths in results.items():
        if "baseline" in paths and "battery_saver" in paths:
            baseline_metrics = extract_metrics(paths["baseline"])
            test_metrics = extract_metrics(paths["battery_saver"])
            print(generate_markdown_table(baseline_metrics, test_metrics, cuj))


if __name__ == "__main__":
    main()
