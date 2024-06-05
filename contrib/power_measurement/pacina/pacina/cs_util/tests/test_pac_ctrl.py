# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable=W0212,W0621

"""Tests for pac_ctrl.py file."""

from freezegun import freeze_time  # pylint: disable=import-error
import pytest

from .. import pac_ctrl
from ... import conftest


def test_pac195x_refresh_wait_time(pac_device, sample_ch_config_entry) -> None:
    pac_drv = pac_ctrl.Pac195xInst(
        pac_device, ch_config={1: sample_ch_config_entry("1")}
    )
    assert pac_drv._refresh_wait_time == pytest.approx(0.0004)

    pac_drv = pac_ctrl.Pac195xInst(
        pac_device,
        ch_config={
            1: sample_ch_config_entry("1"),
            3: sample_ch_config_entry("3"),
        },
    )
    assert pac_drv._refresh_wait_time == pytest.approx(0.0006)

    pac_drv = pac_ctrl.Pac195xInst(
        pac_device,
        fast_mode=False,
        ch_config={
            1: sample_ch_config_entry("1"),
            3: sample_ch_config_entry("3"),
        },
    )
    assert pac_drv._refresh_wait_time == pytest.approx(0.001)


def test_pac195x_derive_disabled_channels(pac_drv) -> None:
    assert pac_drv._derive_disabled_channels([0, 1, 2, 3]) == 0b0000
    assert pac_drv._derive_disabled_channels([0]) == 0b0111
    assert pac_drv._derive_disabled_channels([1]) == 0b1011
    assert pac_drv._derive_disabled_channels([2, 3]) == 0b1100
    assert pac_drv._derive_disabled_channels([]) == 0b1111


def test_pac195x_derive_disabled_channels__raises_on_high_channel_num(
    pac_drv,
) -> None:
    with pytest.raises(AssertionError):
        pac_drv._derive_disabled_channels([4])


def test_pac195x_total_num_of_channels(pac_drv) -> None:
    assert pac_drv._total_num_of_channels == 4

    pac_drv.drv = "pac1952"
    assert pac_drv._total_num_of_channels == 2


def test_pac195x_dump_accumulator(pac_drv) -> None:
    reg, count = pac_drv._dump_accumulator(ch=1)
    assert (reg, count) == (pac_drv.device.ACC_VAL, pac_drv.device.ACC_COUNT)


@freeze_time(conftest.FROZEN_TIME)
def test_log_continuous(pac_drv, log_continuous_data) -> None:
    samples = 3
    # first sample is discarded
    for _ in range(samples + 1):
        pac_drv.log_continuous()

    assert pac_drv.data == [log_continuous_data] * samples


def test_log_single(pac_drv) -> None:
    pac_drv.log_single()
    assert pac_drv.data_single == [
        [
            conftest.CH_VNAME,
            conftest.LOG_SINGLE_VOLTAGE,
            conftest.LOG_SINGLE_CURRENT,
            conftest.LOG_SINGLE_VOLTAGE * conftest.LOG_SINGLE_CURRENT,
        ]
    ]


def test_failing_assertion() -> None:
    # TODO: should not be merged, added only to assert that tests are running
    assert False
