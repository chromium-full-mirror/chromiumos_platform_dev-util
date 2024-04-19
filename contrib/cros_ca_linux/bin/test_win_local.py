# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The local runner on Windows."""

import argparse
import os
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
    parser = argparse.ArgumentParser(
        description="Run tests on Windows locally."
    )
    utils.add_vars(parser)
    utils.add_tests(parser)
    utils.add_run_id(parser)

    return parser.parse_args(argv)


def main(argv):
    """Entry point for test.

    Args:
        argv: arguments list
    """
    arguments = parse_arguments(argv)
    variables = utils.parse_vars(arguments.variables)

    start_file = f"{df.PROJECT_ROOT_DIR}/{df.TEST_START_FILE}"
    end_file = f"{df.PROJECT_ROOT_DIR}/{df.TEST_END_FILE}"
    if arguments.run_id:
        # Run ID provided. Log the start / end with the run_id under
        # df.PROJECT_ROOT_DIR.
        try:
            os.remove(end_file)
        except OSError:
            pass
        with open(start_file, "w", encoding="utf-8") as f:
            f.write(f"{arguments.run_id}")
    try:
        dut = win_local_host.WinLocalHost()
        runner = local_runner.LocalRunner(
            dut, variables, arguments.tests, df.TEST_WIN_RESULTS_DIR, logger
        )
        runner.run_test()
    finally:
        if arguments.run_id:
            # Run ID provided. Log the start / end with the run_id
            # under df.PROJECT_ROOT_DIR.
            with open(end_file, "w", encoding="utf-8") as f:
                f.write(f"{arguments.run_id}")


if __name__ == "__main__":
    main(sys.argv[1:])
