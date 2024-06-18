# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Interface for initializing measurements on the devices."""

import contextlib
import logging
import signal
import sys
import time
import typing as t

from pyftdi.ftdi import Ftdi  # pylint: disable=import-error
from pyftdi.i2c import I2cController  # pylint: disable=import-error
import usb  # pylint: disable=import-error

from . import cs_config
from . import cs_types


logger = logging.getLogger(__name__)

# The default frequency is 100 kHz. PAC supports up to
# - I2C Fast Mode Plus (1MHz)
# - I2C High-Speed mode (3.4 MHz)
# Due to observed i2c clock stretching on PAC side, most likely values higher
# than 1 MHz won't yield positive results.
I2C_CLOCK_FREQUENCY: t.Final[int] = 1 * 1000 * 1000  # MHz

# Please read '3 Buffers and the Latency Timer' section of the following doc:
# https://ftdichip.com/wp-content/uploads/2020/08/AN232B-04_DataLatencyFlow.pdf
# before attempting to change this value.
#
# Due to USB frame size being 1 ms in low speed and full speed, it cannot be
# lower than 2ms. However, in case of USB high speed, the protocol operates on
# microframes of size 125 µs, hence, making the 1 ms available.
FTDI_LATENCY_TIMER__USB_STD_SPEED_MODE: t.Final[int] = 2  # ms
FTDI_LATENCY_TIMER__USB_HIGH_SPEED_MDOE: t.Final[int] = 1  # ms

SAMPLING_WAIT_SLEEP_TIME: t.Final[float] = 0.050  # s


class SignalHandler:
    """Signal Handler that sets a flag."""

    def __init__(self) -> None:
        self.terminate_signal = False
        signal.signal(signal.SIGINT, self._signal_handler)

    def _signal_handler(self, signum, unused_frame) -> None:
        """Define a signal handler for record so we can stop on CTRL-C.

        Autotest can call subprocess.kill which will make us stop waiting and
        dump the rest of the log.
        """
        logger.warning("Signal handler called with signal %d", signum)
        self.terminate_signal = True

    def __bool__(self) -> bool:
        return self.terminate_signal


def is_usb_high_speed_mode(controller: cs_types.DeviceController) -> bool:
    return controller.ftdi.usb_dev.speed == usb.util.SPEED_HIGH


@contextlib.contextmanager
def managed_ftdi(ftdi_url: str) -> t.Iterator[cs_types.DeviceController]:
    """Managed FTDI resource."""
    ftdi_inst = I2cController()
    ftdi_inst.configure(ftdi_url, frequency=I2C_CLOCK_FREQUENCY)

    if is_usb_high_speed_mode(ftdi_inst):
        logger.info("USB High speed mode detected")
        latency_timer = FTDI_LATENCY_TIMER__USB_HIGH_SPEED_MDOE
    else:
        latency_timer = FTDI_LATENCY_TIMER__USB_STD_SPEED_MODE

    ftdi_inst.ftdi.set_latency_timer(latency_timer)

    try:
        yield ftdi_inst
    finally:
        ftdi_inst.close()


def initialize_ftdi(ftdi_urls: t.List[str]) -> None:
    # Adding PacDebugger V1 VID PID
    Ftdi.add_custom_vendor(0x18D1, "Google")
    Ftdi.add_custom_product(0x18D1, 0x5211, "PacDebuggerV1")

    ftdis = Ftdi.list_devices()
    if len(ftdis) == 0:
        logger.error("No FTDIs found. Aborting!")
        sys.exit(1)

    for ftdi_url in ftdi_urls:
        Ftdi.get_device(ftdi_url)


def log_power_continuous(
    configs: t.List[cs_config.BusConfig],
    sample_time: float,
    log_duration: float = 0,
    measurements: int = 0,
) -> t.Tuple[list, dict]:
    assert (log_duration != 0) ^ (
        measurements != 0
    ), "Specify EITHER log_duration or number of measurements"

    logger.info("Measuring Power for %ss.", log_duration)
    terminate_signal = SignalHandler()

    stime = time.time()
    timeout = stime + log_duration

    # Current sensors get reset by the first read (setting t=0)
    tprev_sample = 0.0
    sample_index = 0

    while True:
        tcur_sample = time.time()
        if log_duration and tcur_sample > timeout:
            if sample_index > 1:
                break
        else:
            if tprev_sample == 0:
                tprev_sample = tcur_sample
            elif tcur_sample - tprev_sample < sample_time:
                time_left = sample_time - (tcur_sample - tprev_sample)
                if time_left > 4 * SAMPLING_WAIT_SLEEP_TIME:
                    time.sleep(SAMPLING_WAIT_SLEEP_TIME)
                continue
            else:
                tprev_sample = tcur_sample
        if terminate_signal:
            break
        if measurements and measurements <= sample_index:
            break
        if log_duration:
            print(
                "Logging: %.2f / %.2f s..."
                % (tcur_sample - stime, log_duration),
                end="\r",
            )
        else:
            print(
                "Logging: %.2f / %.2f" % (sample_index, measurements),
                end="\r",
            )

        for config in configs:
            config.log_continuous()
        sample_index = sample_index + 1

    logger.info("Completed Logging")
    return cs_config.collect_data_from_bus_configs(configs)


def log_power_single(
    configs: t.List[cs_config.BusConfig],
) -> t.Tuple[list, list]:
    data = []
    gpio_data = []
    logger.info(
        "Taking a single voltage, current power measurement of all rails"
    )

    for config in configs:
        config.log_single()

    for config in configs:
        data.extend(config.data)
        if config.gpio_vals:
            gpio_data.extend(config.gpio_vals)
    return data, gpio_data
