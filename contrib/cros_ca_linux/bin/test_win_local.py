# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The local runner on Windows."""

import argparse
from pathlib import Path
import sys


# pylint: disable=wrong-import-position

if __name__ == "__main__" and __package__ is None:
    # Add project root into the python path so lib importing works.
    sys.path.append(str(Path(__file__).resolve().parents[1]))

from bin import utils
import lib.definition as df
from lib.host import win_local_host
from lib.runner import local_runner
from lib.utils import logging as lg


# pylint: enable=wrong-import-position

logger = lg.logger


def main(argv):
    """Entry point for test.

    Args:
        argv: arguments list
    """
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
            program.result_dir,
            logger,
        )
        runner.run_test()

    program.run_tests(run)


if __name__ == "__main__":
    main(sys.argv[1:])
