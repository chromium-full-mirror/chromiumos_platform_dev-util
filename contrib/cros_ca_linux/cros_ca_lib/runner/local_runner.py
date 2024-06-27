# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Local test running on the DUT."""

import importlib
from pathlib import Path
from typing import List

from cros_ca_lib.runner import base_runner as br
from cros_ca_lib.runner import test_result as tr
from cros_ca_lib.testing.lib import base_test


class LocalRunner(br.BaseRunner):
    """Local test running on the DUT."""

    location = "local"  # Runner location

    def find_test(self, test) -> List[br.ParsedTest]:
        # This is the default implementation for the local runner.
        # Find the local test and local preparation code.
        t = self.find_parsed_test(test, "local", self.location)
        if not t:
            return []
        return [t]

    def run(
        self, parsed_test: br.ParsedTest, test_dir: Path, result: tr.Result
    ):
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
            # Get the class dynamically using getattr. Class name is test
            # case name.
            cls = getattr(module, parsed_test.cls_name)
            prepare_instance: base_test.BasePrepare = cls(
                self.dut,
                parsed_test.param,
                self.variables,
                test_dir,
                self.logger,
            )
            new_vars = prepare_instance.prepare_test()
            for k in new_vars:
                test_vars[k] = new_vars[k]  # Override if alredy exists.

        if parsed_test.test:
            self.logger.info("Running the main test (local)...")
            self.logger.info("Running variables: %s", br.print_vars(test_vars))
            # Import the module dynamically
            module = importlib.import_module(parsed_test.test)
            # Get the class dynamically using getattr. Class name is test
            # case name.
            cls = getattr(module, parsed_test.cls_name)
            run_instance: base_test.BaseTest = cls(
                self.dut, parsed_test.param, test_vars, test_dir, self.logger
            )
            run_instance.run_test()

        if parsed_test.post:
            self.logger.info(f"Running the post test ({self.location})...")
            # Import the module dynamically
            module = importlib.import_module(parsed_test.post)
            # Get the class dynamically using getattr. Class name is test
            # case name.
            cls = getattr(module, parsed_test.cls_name)
            post_instance: base_test.BasePost = cls(
                self.dut, parsed_test.param, test_vars, test_dir, self.logger
            )
            post_instance.post_test()
