# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The base runner that defines the functions of a runner.

Runners derived from the based runner can be used to run tests
"""

import datetime
import json
import os
import pathlib
import traceback
from typing import Dict, List, Optional

import cros_ca_lib.definition as df
from cros_ca_lib.host import base_host
from cros_ca_lib.runner import test_result as tr
from cros_ca_lib.utils import logging as lg


TEST_BASE_PATH = "cros_ca_lib/testing"
TEST_BASE_PACKAGE = "cros_ca_lib.testing"


def print_vars(test_vars: Dict) -> str:
    """Remove sensitive information and print the vars."""
    cp = dict(test_vars)  # make a copy.
    cp.pop(df.VAR_BOND_CREDENTIALS, None)
    cp.pop(df.VAR_GAIA_ACCOUNT, None)
    cp.pop(df.VAR_GAIA_PASSWORD, None)
    return f"{cp}"


def merge_local_results(local_results: str, result: tr.TestResult) -> None:
    if result.location == "remote":
        # Only remote test available:
        result.result = result.remote_result
        result.error = result.remote_error
        return
    result.result = tr.Result.NOT_RUN
    result.error = result.remote_error
    if not os.path.exists(local_results):
        # Local result is missing:
        return
    with open(local_results, "r", encoding="utf-8") as f:
        data = json.load(f)
        for item in data:
            lr = tr.from_dict(item)
            if lr.name == result.name:
                result.local_start_time = lr.local_start_time
                result.local_end_time = lr.local_end_time
                result.local_error = lr.local_error
                result.local_result = lr.local_result

                # Use local results as the final results.
                # This might need to be changed later.
                result.result = result.local_result
                result.error = result.local_error
                break
    return


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
        iteration: int,
        results_dir: pathlib.Path,
        logger: lg.logging.Logger,
        sleep_before_test: int = 0,
    ):
        """Initializes a Host RemoteRunner with connection details.

        Args:
            dut: The DUT client object for interacting with the device.
            variables: Variables and values to run the test.
            tests: List of test case names.
            iteration: The number of iterations the test should run.
            results_dir: Directory to put the test results.
            logger: The logger that the test should use.
            sleep_before_test: The sleep time before running the test,
                in minutes.
        """
        self.dut = dut
        self.variables = variables
        self.tests = tests
        self.iteration = iteration
        self.results_dir = results_dir
        self.results_tests_dir = self.results_dir / "tests"
        self.logger = logger
        self.sleep_before_test = sleep_before_test

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
        test_file = df.PROJECT_ROOT_DIR / f"{TEST_BASE_PATH}/{l}/{cat}/{tc}.py"
        if not test_file.exists():
            return None

        tm = f"{TEST_BASE_PACKAGE}.{l}.{cat}.{tc}"  # test module
        prepare_file = (
            df.PROJECT_ROOT_DIR
            / f"{TEST_BASE_PATH}/{l}/{cat}/{pl}_prepare/{tc}.py"
        )
        if not prepare_file.exists():
            pm = ""  # prepare module
        else:
            pm = f"{TEST_BASE_PACKAGE}.{l}.{cat}.{pl}_prepare.{tc}"

        post_file = (
            df.PROJECT_ROOT_DIR
            / f"{TEST_BASE_PATH}/{l}/{cat}/{pl}_post/{tc}.py"
        )
        if not post_file.exists():
            postm = ""  # post module
        else:
            postm = f"{TEST_BASE_PACKAGE}.{l}.{cat}.{pl}_post.{tc}"

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

            results: List[tr.TestResult] = []
            for idx, pt in enumerate(self.parsed_tests):
                results.extend(self._run_test(idx + 1, pt))

            results_file = self.results_dir / df.TEST_RESULTS_FILE
            with open(results_file, "w", encoding="utf-8") as f:
                results_dicts = [r.to_dict() for r in results]
                json.dump(results_dicts, f, indent=4)
            summary_str = ""
            for r in results:
                summary_str += f"\n\t\t\t\t{r.name}:\t{r.result.name}"
            if summary_str:
                self.logger.info("Test summary: %s", summary_str)

        except Exception as e:
            self.logger.error("Execution error: %s", e)
            cb = traceback.format_exc()
            self.logger.info(cb)
        finally:
            self.logger.info(f"Test results saved in {self.results_dir}")

            self.logger.removeHandler(run_file_handler)
            self.dut.close()

    def _run_test(self, test_no: int, pt: ParsedTest) -> List[tr.TestResult]:
        # Set up logs for the test
        test_dir = self.results_tests_dir / pt.test_name
        results: List[tr.TestResult] = []
        if self.iteration > 1:
            for iter_no in range(1, self.iteration + 1):
                results.append(
                    self._run_test_iteration(iter_no, test_no, test_dir, pt)
                )
        else:
            results.append(
                self._run_test_iteration(None, test_no, test_dir, pt)
            )
        return results

    def _sleep(self) -> None:
        if self.sleep_before_test > 0:
            self.logger.info(
                "Sleep %d minutes before running the test",
                self.sleep_before_test,
            )
            self.dut.sleep(self.sleep_before_test * 60)

    def _run_test_iteration(
        self,
        iteration_no: Optional[int],
        test_no: int,
        test_dir: pathlib.Path,
        pt: ParsedTest,
    ) -> tr.TestResult:
        self._sleep()
        if iteration_no is not None:
            test_dir = test_dir / f"iteration_{iteration_no}"
            round_no = (
                f"Test {test_no} / {len(self.parsed_tests)} - "
                f"Iteration {iteration_no} / {self.iteration}"
            )
        else:
            round_no = f"{test_no} / {len(self.parsed_tests)}"

        if not test_dir.exists():
            test_dir.mkdir(parents=True)

        # File handler for a log file
        test_log_file = test_dir / f"test_{self.location}.log"
        test_file_handler = lg.logging.FileHandler(test_log_file)
        test_file_handler.setFormatter(lg.formatter)
        self.logger.addHandler(test_file_handler)

        self.logger.info(
            "[%s] Start test %s (%s)", round_no, pt.test_name, pt.location
        )

        self.logger.info("DUT name: %s", self.dut.name())
        result = tr.TestResult()
        result.name = pt.test_name
        result.iteration = iteration_no
        result.location = pt.location
        if self.location == "remote":
            result.remote_start_time = datetime.datetime.now()
        else:
            result.local_start_time = datetime.datetime.now()
        err = None
        try:
            self.run(pt, test_dir, result)
        except Exception as e:
            err = e
            cb = traceback.format_exc()
            self.logger.info(cb)
        finally:
            if self.location == "remote":
                result.remote_error = str(err) if err else None
                result.remote_result = tr.Result.FAIL if err else tr.Result.PASS
                result.remote_end_time = datetime.datetime.now()

                local_result_path = str(test_dir / df.TEST_RESULTS_FILE)
                merge_local_results(local_result_path, result)

            else:
                result.local_error = str(err) if err else None
                result.local_result = tr.Result.FAIL if err else tr.Result.PASS
                result.local_end_time = datetime.datetime.now()
            if result.error:
                self.logger.error(
                    "Run test %s error: %s", pt.test_name, result.error
                )
            self.logger.info(
                "[%s] Ended test %s. Result: %s",
                round_no,
                pt.test_name,
                result.result.name,
            )
            self.logger.removeHandler(test_file_handler)
        return result

    def run(
        self, parsed_test: ParsedTest, test_dir: pathlib.Path, result: tr.Result
    ):
        raise NotImplementedError("run() must be implemented in a subclass")
