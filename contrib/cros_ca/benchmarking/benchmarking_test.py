# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable= W0718
"""Script contains BenchmarkingTest class used to be
inherited by project use cases."""
import traceback

from common.constants import LoggingConfig

LOGGER = LoggingConfig.get_logger()


class BenchmarkingTest:
    """Class used to be inherited by project use cases."""

    def set_up(self):
        """Method will be run before executing main logic of use case,
        to prepare required use case data."""

    def execute(self):
        """Method responsible for executing core logic of the use case."""

    def teardown(self):
        """Method will be run after executing main logic of use case,
        to delete use case data."""

    def run_test(self):
        """Method to run the test based on special order"""
        LOGGER.info("start running test")
        try:
            self.set_up()
            self.execute()
        except Exception as e:
            LOGGER.exception(e)
            traceback.print_exception(type(e), e, e.__traceback__)
        finally:
            try:
                self.teardown()
            except Exception as e:
                LOGGER.exception(e)
                traceback.print_exception(type(e), e, e.__traceback__)
        LOGGER.info("end of running test")
