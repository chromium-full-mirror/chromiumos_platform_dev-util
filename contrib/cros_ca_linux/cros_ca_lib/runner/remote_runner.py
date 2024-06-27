# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""A test runner program that triggers the test execution on the host."""

import datetime
import importlib
import os
from pathlib import Path
import shutil
from typing import List

import cros_ca_lib.definition as df
from cros_ca_lib.runner import base_runner as br
from cros_ca_lib.runner import test_result as tr

# TODO: remove this after all metrics for in the same format.
from cros_ca_lib.runner.post_processing import metric_process_from_log as mp
from cros_ca_lib.testing.lib import base_test as bt
from cros_ca_lib.utils import accounts


class RemoteRunner(br.BaseRunner):
    """A test runner program that triggers the test execution on the host."""

    location = "remote"  # Runner location

    def find_test(self, test) -> List[br.ParsedTest]:
        tests = []
        # Find remote test and its remote preparation test.
        remote_test = self.find_parsed_test(test, "remote", self.location)
        # Find local test and its remote preparation test.
        local_test = self.find_parsed_test(test, "local", self.location)

        if remote_test:
            tests.append(remote_test)
        if local_test:
            tests.append(local_test)
        return tests

    def add_extra_vars(self):
        """Add extra variables to the runner."""
        # Google gaia account
        a_key = df.VAR_GAIA_ACCOUNT
        p_key = df.VAR_GAIA_PASSWORD
        if a_key not in self.variables or p_key not in self.variables:
            try:
                a, p = accounts.get_gaia_account()
                if a_key not in self.variables:
                    self.variables[a_key] = a
                if p_key not in self.variables:
                    self.variables[p_key] = p
            except Exception as e:
                self.logger.debug(repr(e))
                self.logger.warning(
                    "Failed to load gaia_account and gaia_password. Tests "
                    + "needs these variables will fail"
                )

    def run(
        self, parsed_test: br.ParsedTest, test_dir: Path, result: tr.TestResult
    ):
        self.add_extra_vars()
        test_vars = dict(
            self.variables
        )  # Make a copy so each test can modify it.

        # For each test, the following methods will be called in sequence:
        # -- prepare_test(): some preparation procedures for the test.
        #                    It returns additional variables that can be used
        #                    by the main test.
        # -- run_test(): main test logic.
        # -- post_test(): post test logic.
        if parsed_test.prepare:
            self.logger.info(f"Running the prepare test ({self.location})...")
            # Import the module dynamically
            module = importlib.import_module(parsed_test.prepare)
            # Get the class dynamically using getattr. Class name is test case
            # name.
            cls = getattr(module, parsed_test.cls_name)
            prepare_instance: bt.BasePrepare = cls(
                self.dut,
                parsed_test.param,
                test_vars,
                test_dir,
                self.logger,
            )
            new_vars = prepare_instance.prepare_test()
            for k in new_vars:
                test_vars[k] = new_vars[k]  # Override if alredy exists.

        if parsed_test.test and parsed_test.location == "remote":
            self.logger.info("Running the main test (remote)...")
            self.logger.info("Running variables: %s", br.print_vars(test_vars))
            # Import the module dynamically
            module = importlib.import_module(parsed_test.test)
            # Get the class dynamically using getattr. Class name is test case
            # name.
            cls = getattr(module, parsed_test.cls_name)
            run_instance: bt.BaseRun = cls(
                self.dut, parsed_test.param, test_vars, test_dir, self.logger
            )
            run_instance.run_test()

        elif parsed_test.test and parsed_test.location == "local":
            self.logger.info("Triggering the main test (from remote)...")
            self.call_local_test(parsed_test, test_vars, test_dir, result)

        if parsed_test.post:
            self.logger.info(f"Running the post test ({self.location})...")
            # Import the module dynamically
            module = importlib.import_module(parsed_test.post)
            # Get the class dynamically using getattr. Class name is test case
            # name.
            cls = getattr(module, parsed_test.cls_name)
            post_instance: bt.BasePost = cls(
                self.dut, parsed_test.param, test_vars, test_dir, self.logger
            )
            post_instance.post_test()

        self.logger.info(f"Test results saved in {test_dir}")

    def call_local_test(
        self,
        parsed_test: br.ParsedTest,
        variables: dict,
        test_dir: str,
        result: tr.TestResult,
    ):
        raise NotImplementedError(
            "call_local_test() must be implemented in a subclass"
        )

    # TODO: design a flexible hook for customized post log processor.
    def log_post_process(self, test_log_dir):
        metrics_results = []

        energy = "123"
        duration = "456"
        output = mp.metric_process(energy, duration, test_log_dir)

        if len(output) > 0:
            now = datetime.datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
            filename = f"metrics_{now}.txt"

            # Create the file with appropriate permissions
            filepath = os.path.join(test_log_dir, filename)
            with open(filepath, "w", encoding="utf-8") as file:
                for item in output:
                    file.write(f"{item}\n")
            metrics_results.append(filepath)
        else:
            self.logger.info(f"No metrics found for {test_log_dir}")

        if len(metrics_results) > 0:
            self.logger.info(f"Generated metrics files: {metrics_results}")

    def _merge_test_dir(self, host_dir, dut_dir_name, test_name):
        """Merge the local and remote test directory"""

        # Merge the local test dir with the host dir.
        # Assuming the local test dir structure is:
        #    <root>/
        #      |---- full_local.log
        #      |---- ...
        #      |---- tests/<test>/
        #                     |---- test_local.log
        #                     |---- ...
        def move_dir(src: str, dest: str, excluds: List[str] = None):
            all_moved = True
            if not excluds:
                excluds = []
            for item in os.listdir(src):
                if item in excluds:
                    continue
                source_path = os.path.join(src, item)
                destination_path = os.path.join(dest, item)
                if os.path.exists(destination_path):
                    self.logger.warning(
                        f"{destination_path} found when moving from "
                        + f"{source_path}; sipt it"
                    )
                    all_moved = False
                    continue
                shutil.move(source_path, destination_path)
            return all_moved

        # Record the DUT result dir name with an empty file
        with open(
            os.path.join(host_dir, f"{dut_dir_name}.local_dir"),
            "w",
            encoding="utf-8",
        ):
            pass

        try:
            src_root = os.path.join(host_dir, dut_dir_name)
            # Handle dirs other than "tests"
            all_moved_root = move_dir(src_root, host_dir, ["tests"])

            # Handle the "tests" dir
            src_case_root = os.path.join(
                host_dir, dut_dir_name, "tests", test_name
            )
            if not os.path.exists(src_case_root):
                raise Exception(
                    f"Test case dir tests/{test_name} could not be found in DUT"
                    + " test results"
                )
            all_moved_case = move_dir(src_case_root, host_dir)
            if all_moved_root and all_moved_case:
                shutil.rmtree(src_root)
        except Exception as e:
            self.logger.warning(f"Error merging test results: {e}")
