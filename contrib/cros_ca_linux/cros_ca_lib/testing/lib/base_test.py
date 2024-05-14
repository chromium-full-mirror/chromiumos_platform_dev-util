# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The base test class definition."""

from pathlib import Path
from typing import Any, Dict

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
        variables: Dict[str, Any],
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
        self.metric_file = test_dir / "results-chart.json"

        self._check_parameters()
        self._check_vars()

    def _check_vars(self):
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

    def _check_parameters(self):
        if len(self.parameters) == 0 and self.param == "":
            return
        if self.param not in self.parameters:
            raise ValueError(f'Test parameter is not defined: "{self.param}"')

    @property
    def parameter(self):
        """Get the test case specific parameter value by the param."""
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


class BaseRun(BaseTest):
    """Base class for running test."""

    def run_test(self) -> None:
        """Running the test."""
        self._run()

    def _run(self) -> None:
        raise NotImplementedError("run() must be implemented in a subclass")


class BasePrepare(BaseTest):
    """Base class for preparing test."""

    def prepare_test(self) -> Dict[str, Any]:
        """Prepare the test.

        Returns:
            dict with extra key / value pairs.
        """
        return self._prepare()

    def _prepare(self) -> Dict[str, Any]:
        raise NotImplementedError("prepare() must be implemented in a subclass")


class BasePost(BaseTest):
    """Base class for post processing after runing the test."""

    def post_test(self) -> Dict[str, Any]:
        """Run post procedure of the test.

        Returns:
            dict with extra key / value pairs.
        """
        self._post()

    def _post(self) -> Dict[str, Any]:
        raise NotImplementedError("prepare() must be implemented in a subclass")
