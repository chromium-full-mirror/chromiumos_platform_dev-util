# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The utilities used by the runners."""

import argparse
import re
from typing import List

import lib.config as cf


config = cf.config


def add_vars(parser: argparse.ArgumentParser):
    """Add var optionatl argument to parser"""
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
            key, value = var.split("=")
            variables[key] = value
    return variables


def add_run_id(parser: argparse.ArgumentParser):
    """Add run_id optionatl argument to parser"""
    parser.add_argument(
        "--run_id",
        action="store",
        type=str,
        help="The unique id for this test run.",
    )


def add_transport(parser: argparse.ArgumentParser):
    """Add host_type optionatl argument to parser"""
    parser.add_argument(
        "--transport",
        action="store",
        type=str,
        help="The transport for this test run. Can be ssh or paramiko.",
    )


def add_config(parser: argparse.ArgumentParser):
    """Add config optionatl argument to parser"""
    parser.add_argument(
        "--verbose",
        dest="verbose",
        action="store_true",
        help="Verbose / debug logging mode.",
    )
    # Add other confiugrations after this.


def add_target(parser: argparse.ArgumentParser):
    """Add target argument to parser"""
    parser.add_argument(
        "target",
        type=str,
        action="store",
        help='the DUT SSH connection spec of the form "[user@]host[:port]"',
    )


def add_tests(parser: argparse.ArgumentParser):
    """Add tests argument to parser"""
    parser.add_argument(
        "tests", metavar="test", type=str, nargs="+", help="test case name"
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
