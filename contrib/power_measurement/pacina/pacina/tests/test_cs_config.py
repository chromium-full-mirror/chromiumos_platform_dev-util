# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable=W0212,W0621

"""Tests for cs_config.py file."""

from freezegun import freeze_time  # pylint: disable=import-error
import pytest

from .. import conftest
from .. import cs_config


@freeze_time(conftest.FROZEN_TIME)
def test_bus_config__log_continuous(bus_config, log_continuous_data) -> None:
    samples = 3
    # first sample is discarded
    for _ in range(samples + 1):
        bus_config.log_continuous()

    assert bus_config.get_acc_pwr() == [log_continuous_data] * samples


def test_bus_config__parse_addr_and_channel(bus_config) -> None:
    test_cases = [
        ("0x10:1", (16, 1)),
        ("0x0a:0", (10, 0)),
        ("0x00:20", (0, 20)),
        ("1:3", (1, 3)),
    ]
    for test_input, result in test_cases:
        assert bus_config._parse_addr_and_channel(test_input) == result


def test_bus_config__parse_addr_and_channel__raises_on_invalid(
    bus_config,
) -> None:
    with pytest.raises(ValueError):
        assert bus_config._parse_addr_and_channel("hj:2")

    with pytest.raises(ValueError):
        assert bus_config._parse_addr_and_channel("0x00:test")


def test_bus_config_sensors_discovery(bus_config) -> None:
    assert list(bus_config.sensors.keys()) == [0x10, 0x0]
    assert bus_config.sensors[0x10].get_ch_info() == {
        0: {"vname": "CH1", "rsense": 0.02, "nom": 12.0}
    }
    # REFRESH_G sensor
    assert bus_config.sensors[0].get_ch_info() is None


@freeze_time(conftest.FROZEN_TIME)
def test_collect_data_from_bus_configs(
    bus_config, log_continuous_data
) -> None:
    bus_config.log_continuous()  # first sample is discarded
    bus_config.log_continuous()

    data, avg_power = cs_config.collect_data_from_bus_configs([bus_config])
    assert data == [log_continuous_data]
    assert avg_power == {
        conftest.CH_VNAME: {
            "acc_power": conftest.MockedPacDevice.MEASURED_POWER
            * conftest.MockedPacDevice.ACC_COUNT,
            "acc_raw": conftest.MockedPacDevice.ACC_VAL,
            "count": conftest.MockedPacDevice.ACC_COUNT,
            "rsense": conftest.RSENSE,
        }
    }
