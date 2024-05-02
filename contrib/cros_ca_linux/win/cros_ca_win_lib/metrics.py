# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Windows metrics initialization"""

from cros_ca_lib.metric.system import loading
from cros_ca_lib.metric.system import power
from win.cros_ca_win_lib.utils import sys_monitor


def init() -> None:
    """Initialize metrics.

    Because the metric value fetchers need the Windows specific denpendencies
    to work, i.e. the sys_monitor.get_cpu_usage relies on the API from pythoncom
    and wmi.
    So we need to assign the Windows specific value fetcher API here in windows
    DUT project instead of the common Metric to sovle the dependencies issue.
    """
    loading.CPUUSage(sys_monitor.get_cpu_usage)
    loading.MemoryUsage(lambda: sys_monitor.get_memory().percent)
    power.BatteryRemainingCapacity(
        sys_monitor.get_battery_remaining_capacity,
    )
    power.BatteryDischargeRate(sys_monitor.get_battery_discharge_rate)
