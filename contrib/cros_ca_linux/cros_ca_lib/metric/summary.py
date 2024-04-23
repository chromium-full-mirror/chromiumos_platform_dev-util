# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Summary classes that summarize metric"""

import abc
from typing import Any, Dict, List, Optional, Type, TypeVar

from cros_ca_lib.metric import metric
from cros_ca_lib.utils import logging as lg
from typing_extensions import Self


logger = lg.logger

T = TypeVar("T")


class Summary(abc.ABC):
    """Class to summarize recorded metric

    Attributes:
        name: The name of this summary
    """

    def __new__(
        cls, metric_cls: Type[metric.Metric[T]], *_args, **_kwargs
    ) -> Optional[Self]:
        if metric_cls.metric is None:
            logger.warning(
                "Skip the metric %s, which is not implemented or initialized",
                metric_cls.name,
            )
            return None
        return super().__new__(cls)

    def __init__(
        self,
        metric_cls: Type[metric.Metric[T]],
        simple_name: Optional[str] = None,
    ):
        self._metric = metric_cls.metric
        if simple_name is None:
            self.name = self._metric.name
        else:
            self.name = metric.format_name(metric_cls.base_name(), simple_name)

        self._values: List[T] = []

    def summarize(self, extra_attrs: Dict[str, Any]) -> Dict[str, Any]:
        """Get the result metrics.

        Args:
            extra_attrs: The extra attributes to form the result.

        Returns:
            The result metrics.
        """
        return {self.name: {"summary": self._summarize(extra_attrs)}}

    def record(self, *_args, **_kwargs) -> None:
        """Record the current metric value"""
        value = self._metric.value_fetcher()
        logger.debug("append value=%s", value)
        self._values.append(value)

    @abc.abstractmethod
    def _summarize(self, extra_attrs: Dict[str, Any]) -> Dict[str, Any]:
        pass


class ListSummary(Summary):
    """Class for List style summary"""

    def _summarize(self, extra_attrs: Dict[str, Any]):
        return {
            "units": self._metric.units,
            **extra_attrs,
            "values": self._values,
        }


class DifferenceSummary(Summary):
    """Class for difference style summary"""

    def _summarize(self, extra_attrs: Dict[str, Any]):
        return {
            "units": self._metric.units,
            **extra_attrs,
            "type": "scalar",
            "value": self._values[0] - self._values[-1],
        }
