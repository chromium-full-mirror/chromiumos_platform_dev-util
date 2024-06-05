# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Common types used by pacina.py"""

import enum
import typing as t


ALL_CHANNELS_ENABLED: t.Final[int] = 0b0000
ALL_CHANNELS_DISABLED: t.Final[int] = 0b1111


class Polarity(enum.Enum):
    """Polarity options for current sensor ICs."""

    UNIPOLAR = "unipolar"
    BIPOLAR = "bipolar"

    def __str__(self) -> str:
        return str(self.value)


class PowerState(enum.Enum):
    """Power state options."""

    UNDEFINED = "undefined"
    Z5 = "z5"
    Z2 = "z2"
    Z1 = "z1"
    S5 = "s5"
    S4 = "s4"
    S3 = "s3"
    S0IX = "s0ix"
    PLT_1H = "plt-1h"
    PLT_10H = "plt-10h"

    def __str__(self) -> str:
        return str(self.value)


class SamplingDevice(t.Protocol):
    """Generic protocol for power measurement device interaction."""

    def write_to(
        self, regaddr: int, out: t.Union[bytes, bytearray, t.Iterable[int]]
    ):
        ...

    def read_from(self, regaddr: int, readlen: int = 0):
        ...

    @property
    def frequency(self) -> float:
        raise NotImplementedError

    @property
    def address(self) -> int:
        raise NotImplementedError


class UsbDevInterface(t.Protocol):
    """Protocol for USB device."""

    @property
    def speed(self) -> int:
        raise NotImplementedError


class FtdiCtrlInterface(t.Protocol):
    """Protocol for FTDI device controller."""

    @property
    def usb_dev(self) -> UsbDevInterface:
        raise NotImplementedError


class DeviceController(t.Protocol):
    """Generic protocol for sampling device controller."""

    def get_port(self, address: int) -> SamplingDevice:
        ...

    @property
    def ftdi(self) -> FtdiCtrlInterface:
        raise NotImplementedError
