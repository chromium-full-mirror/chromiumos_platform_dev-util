# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The local runner on Windows."""

import argparse
import sys

import cros_ca_lib.definition as df
from cros_ca_lib.runner import local_runner
from cros_ca_lib.utils import logging as lg
from cros_ca_win_lib import metrics
from cros_ca_win_lib.host import win_local_host

from bin import utils


logger = lg.logger

# Initialize the metrics here by using the value fetcher implemented by calling
# the Windows specific dependencies.
metrics.init()


def test_win_local():
    """Entry point for test."""
    argv = sys.argv[1:]
    program = utils.TestProgram(df.TEST_WIN_RESULTS_DIR)

    def add_additional_args(
        parser: argparse.ArgumentParser,  # pylint: disable=unused-argument
    ):
        pass

    program.parse_arguments(
        argv, add_additional_args, "Run tests on Windows locally"
    )

    def run():
        dut = win_local_host.WinLocalHost()
        runner = local_runner.LocalRunner(
            dut,
            program.variables,
            program.args.tests,
            program.args.iteration,
            program.result_dir,
            logger,
            program.args.sleep_before_test,
        )
        runner.run_test()

    program.run_tests(run)
