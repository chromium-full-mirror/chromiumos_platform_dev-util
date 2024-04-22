# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The remote runner for CrOS."""

import argparse
from pathlib import Path
import sys


# pylint: disable=wrong-import-position

if __name__ == "__main__" and __package__ is None:
    # Add project root into the python path so lib importing works.
    sys.path.append(str(Path(__file__).resolve().parents[1]))

from bin import utils
from lib import definition as df
import lib.config as cf
from lib.host import cros_paramiko_host
from lib.host import cros_ssh_host
from lib.runner import remote_runner_direct
from lib.utils import logging as lg


logger = lg.logger

# pylint: enable=wrong-import-position


def parse_arguments(argv):
    """Parse command line arguments

    Args:
        argv: argument list to parse

    Returns:
        tuple of parsed arguments

    Raises:
        SystemExit if arguments are malformed, or required arguments
            are not present.
    """
    parser = argparse.ArgumentParser(description="Run tests on CrOS.")
    utils.add_vars(parser)
    utils.add_target(parser)
    utils.add_config(parser)
    utils.add_transport(parser)
    utils.add_tests(parser)
    # With the above config, the program accept these optional arguments:
    # --var=KEY=VALUE       Set a variable with key and value in format
    #                       KEY=VALUE. Can be repeated.
    # --transport=TRANSPORT Set the tranport type. Can be ssh or paramiko.

    args = parser.parse_args(argv)
    try:
        ssh_specs = utils.parse_ssh_host_spec(args.target, platform="chromeos")
    except ValueError as e:
        logger.info("Invalid target: %s", e)
        parser.print_help()
        sys.exit(1)

    return args, ssh_specs


def main(argv):
    """Entry point for test_win script.

    Args:
        argv: arguments list
    """
    arguments, ssh_specs = parse_arguments(argv)
    hostname = ssh_specs["hostname"]
    port = ssh_specs["port"]
    username = ssh_specs["user"]
    variables = utils.parse_vars(arguments.variables)
    cf.args_to_config(arguments)

    if arguments.transport == "ssh":
        # When --transport=ssh argument is given.
        dut = cros_ssh_host.CrOSSSHHost(
            hostname=hostname, port=port, username=username
        )
    else:
        dut = cros_paramiko_host.CrOSParamikoHost(
            hostname=hostname, port=port, username=username
        )

    runner = remote_runner_direct.RemoteRunnerDirect(
        dut, variables, arguments.tests, df.TEST_HOST_RESULTS_DIR, logger
    )
    runner.run_test()


if __name__ == "__main__":
    main(sys.argv[1:])
