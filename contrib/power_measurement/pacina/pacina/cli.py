#!/usr/bin/env python3
# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Command line interface for Pacina"""

import argparse
import json
import logging
import pathlib
import sys
import typing as t

from . import cs_types
from . import pacina


logger = logging.getLogger(__name__)


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
        required=True,
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
        choices=[str(state) for state in cs_types.PowerState],
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


def main(argv: t.Optional[t.List[str]] = None):
    args = parse_args(argv)
    logging.basicConfig(level=args.loglevel)

    dut_info = None

    if args.dut_info and not args.dut:
        raise ValueError("Specify target DUT.")

    if args.dut_info and args.dut:
        dut_info = json.load(args.dut_info)

        if args.dut not in dut_info.keys():
            raise ValueError(args.dut + " not found in " + args.dut_info.name)

        dut_info = dut_info[args.dut]
        args.ftdi_urls = dut_info["ftdi_urls"]
        args.configs = dut_info["configs"]

    pacina.run_measurements(
        ftdi_urls=args.ftdi_urls,
        configs=args.configs,
        power_state=args.power_state,
        output=args.output,
        dut_info=dut_info,
        fast_mode=args.fast_mode,
        polarity=args.polarity,
        single_mode=args.single,
        sample_time=args.sample_time,
        total_time=args.time,
        measurements=args.measurements,
        generate_viz=True,
    )


if __name__ == "__main__":
    main(sys.argv[1:])
