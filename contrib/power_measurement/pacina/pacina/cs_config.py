# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Current sensor related classes"""

import importlib.util
import logging
import os
import time
import typing as t

from pyftdi.i2c import I2cNackError  # pylint: disable=import-error

from . import cs_types


logger = logging.getLogger(__name__)


class CurrentSensor:
    """Current sensor class for channel information."""

    def __init__(self, addr: int, drv: str) -> None:
        self.addr = addr
        self.drv = drv
        self.chs: t.Dict[int, t.Dict[str, t.Any]] = {}
        self.gpio = None
        self.cs = None

    def add_ch_info(self, ch: int, name: str, rsense: float, nom="") -> None:
        if ch in self.chs.keys():
            logger.warning(
                "%d already exists for %s %d. Overwriting.",
                ch,
                self.drv,
                self.addr,
            )

        self.chs[ch] = {"vname": name, "rsense": rsense, "nom": nom}

    def get_ch_info(self) -> t.Optional[t.Dict[int, t.Dict[str, t.Any]]]:
        if not self.chs:
            return None
        return self.chs


class BusConfig:
    """Current sensor bus class for loading config and measurements."""

    def __init__(
        self,
        config_fpath: str,
        ftdi_inst: cs_types.DeviceController,
        drvs: dict,
        polarity: cs_types.Polarity = cs_types.Polarity.UNIPOLAR,
        sample_time: float = 1,
        fast_mode: bool = True,
    ) -> None:
        spec = importlib.util.spec_from_file_location(
            "input_config", config_fpath
        )
        if spec is None or spec.loader is None:
            raise ValueError(f"Could not load: {config_fpath} file.")

        input_config = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(input_config)

        self.sensors: dict = {}
        self.rails: list = []
        self.data: list = []
        self.gpios: dict = {}
        self.gpio_vals: list = []
        self.config_name = os.path.basename(config_fpath)

        try:
            for addr_ch, name, nom, rsense in input_config.PACS:
                self.load_current_sensor_config(
                    "pac1954", addr_ch, name, rsense, nom
                )
            logger.info("Found [PAC] in the config file")
        except AttributeError:
            logger.debug("Could not find [PAC] in the config file")

        try:
            for (
                drv,
                addr_ch,
                name,
                nom,
                rsense,
                unused_mux,
                unused_is_calib,
            ) in input_config.inas:
                self.load_current_sensor_config(
                    drv, addr_ch, name, rsense, nom
                )
            logger.info("Found [inas] in the config file")
        except AttributeError:
            logger.debug("Could not find [inas] in the config file")

        try:
            for rail, parent in input_config.RAILS:
                self.rails.append({"Rail": rail, "Parent": parent})
        except AttributeError:
            logger.debug("Could not find [RAILS] in the config file")

        try:
            for addr, rail in input_config.GPIOS:
                # Assumes that this only applies to PAC1954
                self.gpios[addr] = rail
        except AttributeError:
            logger.debug("Could not find [GPIOS] in the config file")

        # Merge GPIO information with sensor
        for addr, gpio in self.gpios.items():
            if addr not in self.sensors.keys():
                logger.warning(
                    "%d not found for configuring GPIO %s", addr, gpio
                )
            else:
                self.sensors[addr].gpio = gpio

        # Check DRVs, if all PAC19xx enable REFRESH_G
        self.en_refresh_g = all(
            "pac19" in sensor.drv for sensor in self.sensors.values()
        )
        logger.info("Global Refresh - %s", self.en_refresh_g)

        if self.en_refresh_g:
            addr = 0
            self.sensors[addr] = CurrentSensor(addr, "pac1954")

        # Create Current Sensor FTDI Buses
        unreachable_addrs = []
        for addr, sensor in self.sensors.items():
            try:
                sensor.cs = drvs[sensor.drv](
                    ftdi_inst.get_port(addr),
                    sensor.get_ch_info(),
                    polarity=polarity,
                    refresh_global=self.en_refresh_g,
                    sample_time=sample_time,
                    fast_mode=fast_mode,
                    drv=sensor.drv,
                )
                if sensor.gpio:
                    sensor.cs.set_gpio_name(sensor.gpio)
                    self.gpio_vals = []
            except I2cNackError:
                unreachable_addrs.append(addr)
                logger.warning(
                    "Couldn't connect to address: %s; Skipping", hex(addr)
                )

        # Removes unreachable addresses from the sensors list to prevent errors
        for addr in unreachable_addrs:
            del self.sensors[addr]

    def _parse_addr_and_channel(self, addr_ch: str) -> t.Tuple[int, int]:
        if ":" not in addr_ch:
            addr = int(addr_ch, 16)
            ch = 0
        else:
            [addr_str, ch_str] = addr_ch.split(":")
            addr = int(addr_str, 16)
            ch = int(ch_str)
        return addr, ch

    def load_current_sensor_config(
        self, drv: str, addr_ch: str, name: str, rsense: float, nom
    ) -> None:
        addr, ch = self._parse_addr_and_channel(addr_ch)
        if addr not in self.sensors:
            self.sensors[addr] = CurrentSensor(addr, drv)
        self.sensors[addr].add_ch_info(ch, name, rsense, nom)

    def log_single(self) -> None:
        if 0 in self.sensors.keys() and self.en_refresh_g:
            self.sensors[0].cs.log_single()

        for addr, sensor in self.sensors.items():
            if addr == 0:
                continue
            sensor.cs.log_single()
            self.data.extend(sensor.cs.data_single)
            if sensor.gpio:
                self.gpio_vals.append([sensor.gpio, sensor.cs.read_gpio()])

    def log_continuous(self) -> None:
        time_now = time.time()
        if 0 in self.sensors.keys() and self.en_refresh_g:
            self.sensors[0].cs.log_continuous()

        for addr, sensor in self.sensors.items():
            if addr == 0:
                continue
            sensor.cs.log_continuous(time_now)

    def get_acc_pwr(self) -> list:
        for sensor in self.sensors.values():
            self.data.extend(sensor.cs.data)
        return self.data

    def get_avg_pwr(self) -> dict:
        avg_power: dict = {}
        for sensor in self.sensors.values():
            avg_power = {**avg_power, **sensor.cs.acc}
        return avg_power


def collect_data_from_bus_configs(
    bus_configs: t.List[BusConfig],
) -> t.Tuple[list, dict]:
    avg_power: dict = {}
    data: list = []
    for bus_config in bus_configs:
        data.extend(bus_config.get_acc_pwr())
        avg_power = {**avg_power, **bus_config.get_avg_pwr()}
    return data, avg_power
