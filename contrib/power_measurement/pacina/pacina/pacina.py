#!/usr/bin/env python3
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Main file for pacina utility"""

import argparse
import contextlib
import csv
import json
import logging
import pathlib
import sys
import time
import typing as t

from pyftdi.ftdi import Ftdi  # pylint: disable=import-error

from . import cs_config
from . import cs_types
from . import cs_util
from . import measure
from . import viz


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


def parse_args(argv) -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "-s",
        "--single",
        help=(
            "Use to take a single voltage, current "
            "power measurement of all rails"
        ),
        action="store_true",
    )
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument(
        "-t",
        "--time",
        help="Time to capture in seconds. Mutually exclusive with -m",
        default=0.0,
        type=float,
    )
    group.add_argument(
        "-m",
        "--measurements",
        help="Number of measurements to capture. Mutually exclusive with -t",
        default=0,
        type=int,
    )
    parser.add_argument(
        "--configs",
        nargs="*",
        default=[],
        help=(
            "Current sensor configuration files. "
            "Supports both servod and pacman formats. "
            "Number of config files needs to match number of "
            "number of FTDI URLs."
        ),
    )
    parser.add_argument(
        "--power_state",
        default="undefined",
        choices=[
            "undefined",
            "z5",
            "z2",
            "z1",
            "s5",
            "s4",
            "s3",
            "s0ix",
            "plt-1h",
            "plt-10h",
        ],
        help="Power State Information",
    )
    parser.add_argument(
        "-O",
        "--output",
        type=pathlib.Path,
        default="./results",
        help="Path for log files",
    )
    parser.add_argument(
        "-p",
        "--polarity",
        type=cs_types.Polarity,
        default=cs_types.Polarity.UNIPOLAR,
        choices=cs_types.Polarity,
        help="Measurements can either be unipolar or bipolar",
    )
    parser.add_argument(
        "--ftdi-urls",
        nargs="*",
        default=["ftdi:///"],
        help="FTDI URLs. Number of URLs needs to match number of config files",
    )
    parser.add_argument(
        "--dut-info",
        type=argparse.FileType("r"),
        help="JSON file containing DUT related information",
    )
    parser.add_argument(
        "--dut",
        default="",
        help="Target DUT. Only used when --dut_info is used.",
    )
    parser.add_argument(
        "--sample-time",
        default=1,
        type=float,
        help="Target sample time in seconds",
    )
    parser.add_argument(
        "-d",
        "--debug",
        help="Print debug messages",
        action="store_const",
        dest="loglevel",
        const=logging.DEBUG,
        default=logging.WARNING,
    )
    parser.add_argument(
        "-v",
        "--verbose",
        help="Print verbose messages",
        action="store_const",
        dest="loglevel",
        const=logging.INFO,
    )
    parser.add_argument(
        "--no-fast-mode",
        dest="fast_mode",
        default=True,
        action="store_false",
        help="Disable PAC Fast Mode sampling",
    )
    return parser.parse_args(argv)


def _get_log_prefix_from_dut_info(
    dut_info: dict, power_state: str, start_time
) -> str:
    return (
        "_".join(
            [
                dut_info["model"],
                dut_info["build_phase"],
                dut_info["sku"],
                dut_info["dut_id"],
                dut_info["os_version"],
                power_state,
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


def validate_args(args: argparse.Namespace) -> None:
    if args.dut_info and not args.dut:
        raise ValueError("Specify target DUT.")

    if args.dut_info and args.dut:
        dut_info = json.load(args.dut_info)
        if args.dut not in dut_info.keys():
            raise ValueError(args.dut + " not found in " + args.dut_info.name)

    if not args.configs:
        raise ValueError("Current Sensor Configuration(s) Required")

    if len(args.ftdi_urls) != len(args.configs):
        logger.error(
            "Number of config files needs to match number of FTDI URLs"
        )


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


def main(argv: t.Optional[t.List[str]] = None) -> None:
    args = parse_args(argv)
    validate_args(args)

    logging.basicConfig(level=args.loglevel)
    measure.initialize_ftdi()
    start_time = time.time()

    dut_info = None
    log_prefix = ""

    if args.dut_info and args.dut:
        dut_info = json.load(args.dut_info)
        dut_info = dut_info[args.dut]
        args.ftdi_urls = dut_info["ftdi_urls"]
        args.configs = dut_info["configs"]
        log_prefix = _get_log_prefix_from_dut_info(
            dut_info, args.power_state, start_time
        )

    for ftdi_url in args.ftdi_urls:
        Ftdi.get_device(ftdi_url)

    log_path = prepare_output_dir(
        dut_info, log_prefix, args.output, start_time
    )

    # based on the config files, generate cs instances
    configs: t.List[cs_config.BusConfig] = []
    config_rails: t.List[dict] = []

    with contextlib.ExitStack() as stack:
        for ftdi_url, input_config in zip(args.ftdi_urls, args.configs):
            logger.info(
                "Measuring power from %s using %s", ftdi_url, input_config
            )
            ftdi = stack.enter_context(measure.managed_ftdi(ftdi_url))
            config = cs_config.BusConfig(
                config_fpath=input_config,
                ftdi_inst=ftdi,
                drvs=cs_util.supported_pns,
                polarity=args.polarity,
                sample_time=args.sample_time,
                fast_mode=args.fast_mode,
            )

            configs.append(config)
            config_rails.extend(config.rails)

        if args.single:
            data, gpio_data = measure.log_power_single(configs)
            single_log_path = log_path / (log_prefix + "singleLog.csv")
            dump_log_single_data_to_csv(data, single_log_path)
            viz.show_log_single_data(data, gpio_data, LOG_SINGLE_HEADER)
            return

        data, avg_power = measure.log_power_continuous(
            configs,
            args.sample_time,
            args.time,
            args.measurements,
        )

    dump_time_log_to_csv(data, log_path, log_prefix)
    viz.generate_reports(
        start_time,
        log_path,
        log_prefix,
        dut_info,
        data,
        avg_power,
        config_rails,
        args.power_state,
        MEASUREMENTS_LOG_HEADER,
    )


if __name__ == "__main__":
    main(sys.argv[1:])
