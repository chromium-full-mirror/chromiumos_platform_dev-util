# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Power metrics"""

from cros_ca_lib.metric import metric


class BatteryRemainingCapacity(
    metric.Metric[int], simple_name="battery_remaining_capacity", units="mWh"
):
    """Metric of battery remaining capacity"""


class BatteryDischargeRate(
    metric.Metric[int], simple_name="battery_discharge_rate", units="mW"
):
    """Metric of battery discharge rate"""
