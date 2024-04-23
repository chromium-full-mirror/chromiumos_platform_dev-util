# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Loading metrics"""

from cros_ca_lib.metric import metric


class CPUUSage(metric.Metric[float], simple_name="cpu_usage", units="percent"):
    """Metric of CPU usage"""


class MemoryUsage(
    metric.Metric[float], simple_name="memory_usage", units="percent"
):
    """Metric of memory usage"""
