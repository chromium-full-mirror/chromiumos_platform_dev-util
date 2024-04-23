# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The base test class definition."""

from pathlib import Path

from cros_ca_lib.host import base_host
from cros_ca_lib.utils import logging as lg


class BaseTest:
    """Base class for CrOS CA test."""

    required_vars = {}
    optional_vars = {}
    parameters = {}  # Test parameters

    def __init__(
        self,
        dut: base_host.BaseHost,
        param: str,
        variables: dict,
        test_dir: Path,
        logger: lg.logging.Logger,
    ):
        """Initializes the BaseTest instance.

        Args:
            dut: The DUT client for interacting with the device.
            param: Test parameters.
            variables: Variables and values to run the test.
            test_dir: The path for logs and results.
            logger: Logger instance.
        """
        self.test_name = self.__class__.__name__
        self.dut = dut
        self.param = param
        self.variables = variables
        self.logger = logger

        self.test_dir = test_dir
        self.screenshots_dir = test_dir / "screenshots"
        self.histogram_dir = test_dir / "histograms"

        self.check_parameters()
        self.check_vars()

    def run(self) -> None:
        raise NotImplementedError("run() must be implemented in a subclass")

    def run_test(self) -> None:
        self.run()

    def prepare(self) -> dict:
        """Prepare the test.

        Returns:
            dict with extra key / value pairs.
        """
        raise NotImplementedError("prepare() must be implemented in a subclass")

    def prepare_test(self) -> dict:
        """Prepare the test.

        Returns:
            dict with extra key / value pairs.
        """
        return self.prepare()

    def post(self) -> dict:
        """Run post procedure of the test."""
        raise NotImplementedError("prepare() must be implemented in a subclass")

    def post_test(self) -> dict:
        """Run post procedure of the test."""
        self.post()

    def check_vars(self):
        vars_missing = []
        if self.required_vars:
            for key, value in self.required_vars.items():
                if key not in self.variables:
                    vars_missing.append(key)
                    self.logger.warning(
                        "Missing required var %s: (%s). "
                        + "Specify it with --var=%s=[VALUE]",
                        key,
                        value,
                        key,
                    )

        if self.optional_vars:
            for key, value in self.optional_vars.items():
                if key not in self.variables:
                    self.logger.debug(
                        "Optional var is not given: %s: (%s). "
                        + "Specify it with --var=%s=[VALUE]",
                        key,
                        value,
                        key,
                    )

        if len(vars_missing) > 0:
            raise ValueError(f"Missing required variables: {vars_missing}")

    def check_parameters(self):
        if len(self.parameters) == 0 and self.param == "":
            return
        if self.param not in self.parameters:
            raise ValueError(f'Test parameter is not defined: "{self.param}"')

    @property
    def parameter(self):
        return self.parameters[self.param]

    def var(self, name: str) -> str:
        """Get the var.

        Args:
            name: The var name.

        Returns:
            The value of the vr.

        Raises:
            ValueError: if the var is not provided.
        """
        if name in self.variables:
            return self.variables[name]
        else:
            raise ValueError(f"Missing var: {name}")
