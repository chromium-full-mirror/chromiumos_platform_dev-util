# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable=W0212,W0621

"""Tests for measure.py file."""

from .. import conftest
from .. import measure


def test_log_power_single(bus_config) -> None:
    data, gpio_data = measure.log_power_single([bus_config])

    assert not gpio_data
    assert data == [
        [
            conftest.CH_VNAME,
            conftest.LOG_SINGLE_VOLTAGE,
            conftest.LOG_SINGLE_CURRENT,
            conftest.LOG_SINGLE_VOLTAGE * conftest.LOG_SINGLE_CURRENT,
        ],
    ]


def test_log_power_continuous(bus_config, log_continuous_data) -> None:
    data, avg_power = measure.log_power_continuous(
        measurements=10 + 1,
        sample_time=0.0001,
        configs=[bus_config],
    )
    data = [d[2:] for d in data]  # skip time values
    assert data == [log_continuous_data[2:]] * 10
    assert avg_power == {
        conftest.CH_VNAME: {
            "acc_power": conftest.MockedPacDevice.MEASURED_POWER
            * conftest.MockedPacDevice.ACC_COUNT
            * 10,
            "acc_raw": conftest.MockedPacDevice.ACC_VAL * 10,
            "count": conftest.MockedPacDevice.ACC_COUNT * 10,
            "rsense": conftest.RSENSE,
        }
    }
