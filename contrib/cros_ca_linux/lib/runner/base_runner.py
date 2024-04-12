# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The base runner that defines the functions of a runner.

Runners derived from the based runner can be used to run tests
"""

import datetime
import pathlib
import traceback
from typing import List, Optional

import lib.definition as df
from lib.host import base_host
from lib.utils import logging as lg


class ParsedTest:
    """Contains the test information."""

    def __init__(
        self,
        location: str,
        test_name: str,
        cls_name: str,
        param: str,
        test: str,
        prepare="",
        post="",
    ):
        """Initialize the ParsedTest.

        Args:
            location: Remote or local test.
            test_name: Test name.
            cls_name: The test class name.
            param: The test parameter.
            test: The python module that contains the test case.
            prepare: The python module that contains the prepare case.
            post: The python module that contains the post case.
        """
        self.location = location  # "remote" or "local"
        self.test_name = test_name
        self.cls_name = cls_name
        self.param = param
        self.test = test
        self.prepare = prepare
        self.post = post


class BaseRunner:
    """A class representing the base test runner"""

    parsed_tests: List[ParsedTest] = []  # Store the parsed tests object
    location = "local"  # Runner location

    def __init__(
        self,
        dut: base_host.BaseHost,
        variables: dict,
        tests: List[str],
        results_dir: pathlib.Path,
        logger: lg.logging.Logger,
    ):
        """Initializes a Host RemoteRunner with connection details.

        Args:
            dut: The DUT client object for interacting with the device.
            variables: Variables and values to run the test.
            tests: List of test case names.
            results_dir: Directory to put the test results.
            logger: The logger that the test should use.
        """
        self.dut = dut
        self.variables = variables
        self.tests = tests
        self.results_dir = results_dir / datetime.datetime.now().strftime(
            "%Y%m%d-%H%M%S"
        )
        self.results_tests_dir = self.results_dir / "tests"
        self.logger = logger

    def find_parsed_test(
        self, test, location, prepare_location
    ) -> Optional[ParsedTest]:
        # Test case in category.name.parameter format. parameter part is
        # optional.
        parts = test.split(".")
        if len(parts) < 2:
            return None
        cat = parts[0]  # Category
        tc = parts[1]  # Test case / class name
        param = ""  # Test parameter.
        if len(parts) > 2:
            param = ".".join(parts[2:])
        l = location
        pl = prepare_location
        test_file = df.PROJECT_ROOT_DIR / f"lib/testing/{l}/{cat}/{tc}.py"
        if not test_file.exists():
            return None

        tm = f"lib.testing.{l}.{cat}.{tc}"  # test module
        prepare_file = (
            df.PROJECT_ROOT_DIR / f"lib/testing/{l}/{cat}/{pl}_prepare/{tc}.py"
        )
        if not prepare_file.exists():
            pm = ""  # prepare module
        else:
            pm = f"lib.testing.{l}.{cat}.remote_prepare.{tc}"

        post_file = (
            df.PROJECT_ROOT_DIR / f"lib/testing/{l}/{cat}/{pl}_post/{tc}.py"
        )
        if not post_file.exists():
            postm = ""  # post module
        else:
            postm = f"lib.testing.{l}.{cat}.{pl}_post.{tc}"

        return ParsedTest(l, test, tc, param, tm, pm, postm)

    def find_test(self, test) -> List[ParsedTest]:
        # Depending on the runner's location (local or remote), find tests in
        # different package.
        raise NotImplementedError(
            "find_test() must be implemented in a subclass"
        )

    def run_test(self):
        # Create the results dir using the current time for this run.
        if not self.results_tests_dir.exists():
            self.results_tests_dir.mkdir(parents=True)
        # Set up logs for this run.
        run_log_file = self.results_dir / f"full_{self.location}.log"
        run_file_handler = lg.logging.FileHandler(run_log_file)
        run_file_handler.setFormatter(lg.formatter)
        self.logger.addHandler(run_file_handler)
        try:
            tests_not_found = []
            for test in self.tests:
                found_tests = self.find_test(test)
                if len(found_tests) == 0:
                    tests_not_found.append(test)
                else:
                    self.parsed_tests.extend(found_tests)

            if len(tests_not_found) > 0:
                raise Exception(f"Tests are not found: {tests_not_found}")

            has_local_tests = False
            for t in self.parsed_tests:
                if t.location == "local":
                    has_local_tests = True
                    break

            self.dut.connect()
            if has_local_tests:
                self.dut.deploy()
                self.dut.update_src_version()
                self.dut.update_dependency()

            total = len(self.parsed_tests)
            for idx, pt in enumerate(self.parsed_tests):
                tst = pt.test_name
                # Set up logs for the test
                test_dir = self.results_tests_dir / tst
                if not test_dir.exists():
                    test_dir.mkdir(parents=True)
                # File handler for a log file
                test_log_file = test_dir / f"test_{self.location}.log"
                test_file_handler = lg.logging.FileHandler(test_log_file)
                test_file_handler.setFormatter(lg.formatter)
                self.logger.addHandler(test_file_handler)

                self.logger.info(
                    "[%d / %d] Start test %s (%s)",
                    idx + 1,
                    total,
                    tst,
                    pt.location,
                )
                try:
                    self.run(pt, test_dir)
                except Exception as e:
                    self.logger.error(f"Run test {tst} error: {e}")
                    cb = traceback.format_exc()
                    self.logger.info(cb)
                finally:
                    self.logger.info(
                        "[%d / %d] Ended test %s", idx + 1, total, tst
                    )
                    self.logger.removeHandler(test_file_handler)

        except Exception as e:
            self.logger.error("Execution error: %s", e)
            cb = traceback.format_exc()
            self.logger.info(cb)
        finally:
            self.logger.info(f"Test results saved in {self.results_dir}")
            self.logger.removeHandler(run_file_handler)
            self.dut.close()

    def run(self, parsed_test: ParsedTest, test_dir: pathlib.Path):
        raise NotImplementedError("run() must be implemented in a subclass")
