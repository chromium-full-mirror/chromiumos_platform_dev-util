# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Main file for pacina utility"""

import contextlib
import csv
import logging
import pathlib
import sys
import time
import typing as t

from . import cs_config
from . import cs_types
from . import cs_util
from . import measure


logger = logging.getLogger(__name__)

MEASUREMENTS_LOG_HEADER: t.Final[t.List[str]] = [
    "time_abs",
    "time_relative",
    "rail",
    "accum",
    "count",
    "power",
]
LOG_SINGLE_HEADER: t.Final[t.List[str]] = [
    "Rail",
    "Voltage (V)",
    "Current (A)",
    "Power (W)",
]


def _get_log_prefix_from_dut_info(
    dut_info: dict, power_state: cs_types.PowerState, start_time
) -> str:
    return (
        "_".join(
            [
                dut_info["model"],
                dut_info["build_phase"],
                dut_info["sku"],
                dut_info["dut_id"],
                dut_info["os_version"],
                str(power_state),
                time.strftime("%Y%m%d_%H%M%S", time.localtime(start_time)),
            ]
        )
        + "_"
    )


def dump_time_log_to_csv(
    data: list,
    log_path: pathlib.Path,
    log_prefix: str,
) -> None:
    time_log_path = log_path / (log_prefix + "timeLog.csv")

    with open(time_log_path, mode="w", newline="", encoding="utf-8") as file:
        writer = csv.writer(file)
        writer.writerow(MEASUREMENTS_LOG_HEADER)
        writer.writerows(data)

    logger.info("Output written to %s", time_log_path)


def dump_log_single_data_to_csv(
    data: list,
    log_path: pathlib.Path,
) -> None:
    with open(log_path, mode="w", newline="", encoding="utf-8") as file:
        writer = csv.writer(file)
        writer.writerow(LOG_SINGLE_HEADER)
        writer.writerows(data)

    logger.info("Output written to %s", log_path)


def prepare_output_dir(
    dut_info: t.Optional[dict],
    log_prefix: str,
    output: pathlib.Path,
    start_time: float,
) -> pathlib.Path:
    if dut_info and log_prefix:
        log_path = output / dut_info["program"]
    else:
        log_path = output / time.strftime(
            "%Y%m%d_%H%M%S", time.localtime(start_time)
        )

    log_path.mkdir(parents=True, exist_ok=True)
    return log_path


def run_measurements(
    ftdi_urls: t.List[str],
    configs: t.List[str],
    power_state: cs_types.PowerState = cs_types.PowerState.UNDEFINED,
    output: pathlib.Path = pathlib.Path("./results"),
    dut_info: t.Optional[dict] = None,
    fast_mode: bool = True,
    polarity: cs_types.Polarity = cs_types.Polarity.UNIPOLAR,
    single_mode: bool = False,
    sample_time: float = 1,
    total_time: float = 0,
    measurements: int = 0,
    generate_viz: bool = False,
    device_controller_const: t.Callable[
        [str], contextlib.AbstractContextManager
    ] = measure.managed_ftdi,
    initialize_ftdi: t.Callable[[t.List[str]], None] = measure.initialize_ftdi,
) -> None:
    if len(ftdi_urls) != len(configs):
        logger.error(
            "Number of config files needs to match number of FTDI URLs"
        )
        sys.exit(1)

    initialize_ftdi(ftdi_urls)
    start_time = time.time()

    log_prefix = (
        _get_log_prefix_from_dut_info(
            dut_info,
            power_state,
            start_time,
        )
        if dut_info
        else ""
    )

    log_path = prepare_output_dir(dut_info, log_prefix, output, start_time)

    # based on the config files, generate bus config instances
    bus_configs: t.List[cs_config.BusConfig] = []
    config_rails: t.List[dict] = []

    with contextlib.ExitStack() as stack:
        for ftdi_url, input_config in zip(ftdi_urls, configs):
            logger.info(
                "Measuring power from %s using %s", ftdi_url, input_config
            )
            ftdi = stack.enter_context(device_controller_const(ftdi_url))
            config = cs_config.BusConfig(
                config_fpath=input_config,
                ftdi_inst=ftdi,
                drvs=cs_util.supported_pns,
                polarity=polarity,
                sample_time=sample_time,
                fast_mode=fast_mode,
            )

            bus_configs.append(config)
            config_rails.extend(config.rails)

        if single_mode:
            data, gpio_data = measure.log_power_single(bus_configs)
            single_log_path = log_path / (log_prefix + "singleLog.csv")
            dump_log_single_data_to_csv(data, single_log_path)

            if generate_viz:
                from . import viz

                viz.show_log_single_data(data, gpio_data, LOG_SINGLE_HEADER)

            return

        data, avg_power = measure.log_power_continuous(
            bus_configs,
            sample_time,
            total_time,
            measurements,
        )
        dump_time_log_to_csv(data, log_path, log_prefix)

    if generate_viz:
        from . import viz

        viz.generate_reports(
            start_time,
            log_path,
            log_prefix,
            dut_info,
            data,
            avg_power,
            config_rails,
            power_state,
            MEASUREMENTS_LOG_HEADER,
        )
