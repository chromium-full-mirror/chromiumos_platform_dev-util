# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable=W0212,W0621

"""Pytest testing fixtures file."""

import contextlib
import time
import typing as t

from freezegun import freeze_time  # pylint: disable=import-error
import pytest

from . import cs_config
from . import cs_types
from . import cs_util
from .cs_util import _pac19xx
from .cs_util import pac_ctrl


CH_VNAME: t.Final[str] = "CH1"
FROZEN_TIME: t.Final[str] = "1978-01-14"
RSENSE: t.Final[float] = 0.02

LOG_SINGLE_VOLTAGE: t.Final[float] = 12.1796875
LOG_SINGLE_CURRENT: t.Final[float] = 0.00030517578125


@pytest.fixture
def sample_ch_config_entry():
    return lambda vname: {
        "vname": vname,
        "rsense": RSENSE,
        "nom": 12.0,
    }


@pytest.fixture
@freeze_time("1978-01-14")
def log_continuous_data():
    # we freeze time, so it will be always 0
    time_elapsed = 0

    return [
        time.time(),
        time_elapsed,
        CH_VNAME,
        MockedPacDevice.ACC_VAL,
        MockedPacDevice.ACC_COUNT,
        MockedPacDevice.MEASURED_POWER,
    ]


@pytest.fixture
def pac_device():
    return MockedPacDevice()


@pytest.fixture
def pac_drv(pac_device, sample_ch_config_entry) -> pac_ctrl.Pac195xInst:
    return pac_ctrl.Pac195xInst(
        pac_device, ch_config={1: sample_ch_config_entry(CH_VNAME)}
    )


class MockedPacDevice:
    """Mocked PAC device class with hardcoded read behaviour."""

    ACC_VAL = 100_000
    ACC_COUNT = 4
    MEASURED_POWER = 0.003725290298461914

    def write_to(
        self, regaddr: int, out: t.Union[bytes, bytearray, t.Iterable[int]]
    ):
        pass

    def read_from(self, regaddr: int, _: int = 0):
        if regaddr == _pac19xx.ACC_COUNT:
            return bytearray(b"\x00\x00\x00\x04")  # 4

        if _pac19xx.VACC1 <= regaddr <= _pac19xx.VACC4:
            return bytearray(b"\x00\x00\x00\x00\x01\x86\xa0")  # 100_000

        if _pac19xx.VBUS1_AVG <= regaddr <= _pac19xx.VBUS4_AVG:
            return bytearray(b"ap")

        if _pac19xx.VSENSE1_AVG <= regaddr <= _pac19xx.VSENSE4_AVG:
            return bytearray(b"\x00\x04")

        raise NotImplementedError(f"Undefined behaviour on register {regaddr}")

    @property
    def frequency(self) -> float:
        return 1

    @property
    def address(self) -> int:
        return 1


class MockedSamplingDevice:
    """Mocked sampling device returning MockedPacDevice."""

    def get_port(self, _: int) -> cs_types.SamplingDevice:
        return MockedPacDevice()

    @property
    def ftdi(self) -> cs_types.FtdiCtrlInterface:
        return MockeFtdiCtrlInterface


class MockeFtdiCtrlInterface:
    """Mocked FTDI device controller interface."""

    @property
    def usb_dev(self) -> cs_types.UsbDevInterface:
        return MockedUsbDevInterface


class MockedUsbDevInterface:
    """Mocked USB device interface."""

    @property
    def speed(self) -> int:
        import usb  # pylint: disable=import-error

        return usb.util.SPEED_HIGH


@contextlib.contextmanager
def device_controller_const(_: str):
    try:
        yield MockedSamplingDevice()
    finally:
        pass


@pytest.fixture
def sampling_device():
    return MockedSamplingDevice()


@pytest.fixture
def bus_config(sampling_device):
    return cs_config.BusConfig(
        config_fpath="pacina/cpd.tests.py",
        ftdi_inst=sampling_device,
        drvs=cs_util.supported_pns,
    )
