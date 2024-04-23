# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Classes about metric"""

from typing import Callable, Generic, TypeVar

from cros_ca_lib import metric as mtc
from cros_ca_lib.utils import logging as lg


logger = lg.logger

METRIC_BASE_NAME = mtc.__name__

T = TypeVar("T")


class Metric(Generic[T]):
    """Base class for all metrics.

    Almost all metrics depend on platform (e.g. Windows) specific dependcies
    that can only be included and installed in the DUT project.
    So we leave the metric value_fetcher, i.e. the cpu usage metric calls
    platform specific API to get the value, for DUT proejct to assign when
    creating the metric.
    That helps to decouple the common Metric from depending on platform
    specific depenencies.

    The DUT proejct needs to implement the metric value fetcher by importing and
    calling the platform dependent API for creating the singlon metric instance.
    The metric instance can then be used to collect the running metrics by
    summary and recorder.

    Attributes:
        name: The name of this metric.
        units: The unit of this metric.
        metric: The singlton instance of an metric
        value_fetcher: The function to generate the value of this metric.
    """

    name: str
    units: str
    metric: "Metric[T]" = None

    def __init__(self, value_fetcher: Callable[[], T]) -> None:
        self.value_fetcher = value_fetcher
        self.__class__.metric = self

    # disable pylint due to the pylint bug
    # https://github.com/pylint-dev/pylint/issues/6644
    # https://github.com/pylint-dev/pylint/issues/8919
    def __init_subclass__(
        cls, simple_name, units, **kwargs
    ) -> (
        None
    ):  # pylint: disable=unexpected-special-method-signature, arguments-differ
        super().__init_subclass__(**kwargs)
        cls.name = format_name(cls.base_name(), simple_name)
        cls.units = units

    @classmethod
    def base_name(cls) -> str:
        """The Base part of the metric name.

        Resolve the base name of all subclasses automatically.
        The base name of all metrics inherited from this class is the module
        name after this module.
        """
        module_name = cls.__module__
        if not module_name.startswith(METRIC_BASE_NAME):
            raise ValueError(f"Metric class should be under {METRIC_BASE_NAME}")

        return module_name.replace(METRIC_BASE_NAME, "", 1).lstrip(".")


def format_name(base_name: str, simple_name: str) -> str:
    """The metric name, composed of the base name and the simple name.

    Args:
        simple_name:
            The last part of the metric name.
        base_name:
            The base part, aka the part before the last part,
            of the metric name.

    Returns:
        The full metric name.
    """
    return f"{base_name}.{simple_name}"
