# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The utilities used by the runners."""

import argparse
import datetime
import json
import os
from pathlib import Path
import re
import time
from typing import Callable, List

import cros_ca_lib.config as cf
import cros_ca_lib.definition as df
import cros_ca_lib.utils.logging as lg


# pylint: enable=wrong-import-position
logger = lg.logger
config = cf.config


def add_vars(parser: argparse.ArgumentParser):
    """Add var optional argument to parser"""
    parser.add_argument(
        "--var",
        dest="variables",
        action="append",
        type=str,
        metavar=("KEY=VALUE"),
        help="Set a variable with key and value in format KEY=VALUE. "
        + "Can be repeated.",
    )


def parse_vars(arg_vars: List[str]) -> dict:
    """Parse the variables list into key / value dictionary"""
    variables = {}
    if arg_vars:
        for var in arg_vars:
            key, value = var.split("=", 1)
            variables[key] = value
    return variables


def add_run_id(parser: argparse.ArgumentParser):
    """Add run_id optional argument to parser"""
    parser.add_argument(
        "--run_id",
        action="store",
        type=str,
        help="The unique id for this test run.",
    )


def add_console_output(parser: argparse.ArgumentParser):
    """Add no_console_output optional argument to parser"""
    parser.add_argument(
        "--no_console_output",
        action="store_true",
        help="Don't log to console.",
    )


def add_transport(parser: argparse.ArgumentParser):
    """Add transport optional argument to parser"""
    parser.add_argument(
        "--transport",
        action="store",
        type=str,
        help="The transport for this test run. Can be ssh or paramiko.",
    )


def add_verbose(parser: argparse.ArgumentParser):
    """Add verbose optional argument to parser"""
    parser.add_argument(
        "--verbose",
        dest="verbose",
        action="store_true",
        help="Verbose / debug logging mode.",
    )


def add_result_dir(parser: argparse.ArgumentParser):
    """Add result_dir optional argument to parser"""
    parser.add_argument(
        "--result_dir",
        type=str,
        action="store",
        help="The directory of the test logs",
    )


def add_sleep_before_test(parser: argparse.ArgumentParser):
    """Add sleep before test optional arument to parser"""
    parser.add_argument(
        "--sleep_before_test",
        type=int,
        action="store",
        default=0,
        help=(
            "The sleep time in minutes before running each tests and"
            " iterations"
        ),
    )


def add_target(parser: argparse.ArgumentParser):
    """Add target argument to parser"""
    parser.add_argument(
        "target",
        type=str,
        action="store",
        help='The DUT SSH connection spec of the form "[user@]host[:port]"',
    )


def add_tests(parser: argparse.ArgumentParser):
    """Add tests argument to parser"""
    parser.add_argument(
        "tests", metavar="test", type=str, nargs="+", help="test case name"
    )


def add_iteration(parser: argparse.ArgumentParser):
    """Add iteration times artument to parser"""
    parser.add_argument(
        "--iteration",
        type=int,
        default=1,
        help="Iteration times for each test case.",
    )


def parse_ssh_host_spec(host_spec, platform="ChromeOS"):
    """Parses an SSH host spec string into a dictionary.

    It contains user, hostname, and port information.

    Args:
        host_spec: The SSH host spec string (e.g., hostname, user@hostname,
                   user@hostname:port)
        platform: ChromeOS or Windows.

    Returns:
        A dictionary with keys 'user' (optional), 'hostname',
            and 'port' (optional).
    """
    default_user = "root"
    if platform.lower() in ["win", "windows"]:
        default_user = "administrator"

    match = re.match(r"^(?:(.+)@)?([^@:]+)(?:[:](\d+))?$", host_spec)
    if match:
        user = match.group(1)
        hostname = match.group(2)
        port = match.group(3)
        return {
            "user": user if user else default_user,
            "hostname": hostname,
            "port": int(port) if port else 22,  # Default SSH port
        }
    else:
        raise ValueError(f"invalid SSH host spec: {host_spec}")


class TestProgram:
    """The test program."""

    def __init__(self, result_dir: Path):
        self.result_dir: Path = result_dir / datetime.datetime.now().strftime(
            "%Y%m%d-%H%M%S"
        )
        self.args: cf.Config = None
        self.parser: argparse.ArgumentParser = None
        self.variables: dict = {}

    def run_tests(self, run_func: Callable) -> None:
        start_file = f"{df.PROJECT_LOG_DIR}/{df.TEST_START_FILE}"
        end_file = f"{df.PROJECT_LOG_DIR}/{df.TEST_END_FILE}"
        try:
            os.remove(end_file)
        except OSError as e:
            logger.debug("Failed to remove end file: %s", e)
        # Log the start with the run_id under.
        start_json = dict(
            run_id=self.args.run_id,
            start_time=time.time(),
            result_dir=str(self.result_dir),
        )
        with open(start_file, "w", encoding="utf-8") as f:
            json.dump(start_json, f)

        err = None
        try:
            run_func()
        except Exception as e:
            err = e
            raise
        finally:
            end_json = dict(
                run_id=self.args.run_id,
                end_time=time.time(),
                error=f"{err}" if err else None,
                result_dir=str(self.result_dir),
            )
            with open(end_file, "w", encoding="utf-8") as f:
                json.dump(end_json, f)

    def parse_arguments(
        self, argv: List[str], add_additional_args: callable, description: str
    ) -> None:
        """Parse command line arguments

        Args:
            argv: argument list to parse.
            add_additional_args: func to add more args by the caller.
            description: description of the program.

        Raises:
            SystemExit if arguments are malformed, or required arguments
                are not present.
        """
        parser = argparse.ArgumentParser(description=description)
        # Add common arguments
        add_verbose(parser)
        add_run_id(parser)
        add_console_output(parser)
        add_vars(parser)
        add_iteration(parser)
        add_sleep_before_test(parser)
        # Add program specific arguments:
        add_additional_args(parser)
        add_tests(parser)  # Tests are the last arguments.

        self.parser = parser
        self.args = parser.parse_args(argv, cf.Config())
        self.variables = parse_vars(self.args.variables)
        cf.args_to_config(self.args)

        if hasattr(self.args, "result_dir") and self.args.result_dir:
            result_dir = Path(self.args.result_dir)
            if not result_dir.exists():
                raise ValueError(f"result_dir {result_dir} doesn't exist")
            self.result_dir = result_dir / datetime.datetime.now().strftime(
                "%Y%m%d-%H%M%S"
            )
