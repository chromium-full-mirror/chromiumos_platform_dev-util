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

from cros_ca_lib import definition as df
from cros_ca_lib.host import cros_paramiko_host
from cros_ca_lib.host import cros_ssh_host
from cros_ca_lib.runner import remote_runner_direct
from cros_ca_lib.utils import logging as lg

from bin import utils


# pylint: enable=wrong-import-position


logger = lg.logger


def main(argv):
    """Entry point for test.

    Args:
        argv: arguments list
    """
    program = utils.TestProgram(df.TEST_HOST_RESULTS_DIR)

    def add_additional_args(parser: argparse.ArgumentParser):
        utils.add_transport(parser)
        utils.add_result_dir(parser)
        utils.add_target(parser)

    program.parse_arguments(argv, add_additional_args, "Run tests on CrOS")

    try:
        ssh_specs = utils.parse_ssh_host_spec(
            program.args.target, platform="chromeos"
        )
    except ValueError as e:
        logger.info("Invalid target: %s", e)
        program.parser.print_help()
        sys.exit(1)
    hostname = ssh_specs["hostname"]
    port = ssh_specs["port"]
    username = ssh_specs["user"]

    def run():
        arguments = program.args
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
            dut,
            program.variables,
            arguments.tests,
            arguments.iteration,
            program.result_dir,
            logger,
            program.args.sleep_before_test,
        )
        runner.run_test()

    program.run_tests(run)


if __name__ == "__main__":
    main(sys.argv[1:])
