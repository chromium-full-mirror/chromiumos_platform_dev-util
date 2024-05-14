# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Calculate "minutes_battery_life"."""

import json

from cros_ca_lib.utils import logging as lg
from win.cros_ca_win_lib.utils import sys_monitor


logger = lg.logger


def calculate_battery_life(metricfile, time_metric):
    """Calculate "minutes_battery_life" using data in metricfile.

    Read results-chart.json file content and extract
    "battery_remaining_capacity" data to calculate "minutes_battery_life".

    Args:
        metricfile: results-chart.json file path
        time_metric: the metric name for time recording, such as "Browsing.t".
                    All other metrics that use this timeline will be saved into
                    the HTML chart.
    """

    with open(metricfile, "r+", encoding="utf-8") as f:
        data = json.load(f)
        runtime_minutes = (
            data[time_metric]["summary"]["values"][-1]
            - data[time_metric]["summary"]["values"][0]
        ) / 60
        if data["system.power.battery_remaining_capacity"] is not None:
            battery_remaining_capacity = data[
                "system.power.battery_remaining_capacity"
            ]["summary"]["values"]
            logger.info(
                battery_remaining_capacity[0],
                battery_remaining_capacity[-1],
            )
            low_battery_shutdown = 4
            energy_available = sys_monitor.get_battery_max_capacity() - (
                (100 - low_battery_shutdown) / 100
            )
            energy_used = (
                battery_remaining_capacity[0] - battery_remaining_capacity[-1]
            )
            minutes_battery_life = (
                energy_available / energy_used * runtime_minutes
            )
            m = {
                "minutes_battery_life": {
                    "summary": {
                        "units": "minute",
                        "improvement_direction": "up",
                        "type": "scalar",
                        "value": minutes_battery_life,
                    }
                }
            }
            data.update(m)
        f.seek(0)
        json.dump(data, f, indent=2)
