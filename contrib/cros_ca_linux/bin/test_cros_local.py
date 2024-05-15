# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The local runner on CrOS."""

import argparse
from pathlib import Path
import sys


# pylint: disable=wrong-import-position
if __name__ == "__main__" and __package__ is None:
    # Add project root into the python path so lib importing works.
    sys.path.append(str(Path(__file__).resolve().parents[1]))

import cros_ca_lib.definition as df

# Need to move the import of bin.utils before cros_ca_lib.host.cros_local_host
# Becuase the autologin import by cros_local_host adds the path
# "usr/local/autotest/bin" to sys.path, that conflict with the package "bin"
# and the "cros format" will always move this bin import to the last.
from bin import utils


# TODO: remove this after selenium is latest in CrOS
venv_package = str(df.TEST_CROS_VENV_DIR / "lib/python3.10/site-packages/")
for child in df.TEST_CROS_VENV_DIR.glob("**/*"):
    if child.is_dir() and child.name == "site-packages":
        venv_package = str(child)
        break
sys.path.insert(0, venv_package)

from cros_ca_lib.host import cros_local_host
from cros_ca_lib.runner import local_runner
import cros_ca_lib.utils.logging as lg


# pylint: enable=wrong-import-position

logger = lg.logger


def main(argv):
    """Entry point for test.

    Args:
        argv: arguments list
    """
    program = utils.TestProgram(df.TEST_CROS_RESULTS_DIR)

    def add_additional_args(
        parser: argparse.ArgumentParser,  # pylint: disable=unused-argument
    ):
        pass

    program.parse_arguments(
        argv, add_additional_args, "Run tests on CrOS locally"
    )

    def run():
        dut = cros_local_host.CrosLocalHost()
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


if __name__ == "__main__":
    main(sys.argv[1:])
