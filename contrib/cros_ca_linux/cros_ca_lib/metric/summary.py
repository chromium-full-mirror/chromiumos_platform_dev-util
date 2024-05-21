# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Summary classes that summarize metric"""

import abc
import dataclasses
from typing import Any, Dict, List, Optional, Type, TypeVar

from cros_ca_lib.metric import metric
from cros_ca_lib.utils import logging as lg
from typing_extensions import Self


logger = lg.logger

T = TypeVar("T")


@dataclasses.dataclass
class HistogramData:
    """Class to represnet Histogram data.

    Attributes:
        units: The unit of the histogram value.
        improvement_direction: Improvement direction of the histogram.
        extra: Extra attributes of the histogram.
    """

    units: str
    improvement_direction: str
    extra: dataclasses.InitVar[Dict[str, Any]] = {}

    def __post_init__(self, extra: Dict[str, Any]) -> None:
        self._extra = extra

    def to_json_dict(self) -> Dict[str, Any]:
        """Convert the object to dictionary for formating json."""
        result = dataclasses.asdict(self)
        result.update(self._extra)
        return result


@dataclasses.dataclass
class HistogramValue(HistogramData):
    """Class to represent a histogram data with single value.

    Attributes:
        type: The type of the value, default to scalar.
        value: The single value of the histogram.
    """

    type: str = "scalar"
    value: Optional[Any] = None


@dataclasses.dataclass
class HistogramValues(HistogramData):
    """Class to represent a histogram data with a list of values.

    Attributes:
        type: The type of the value, default to list_of_scalar_values.
        values: The list of values of the histogram.
    """

    type: str = "list_of_scalar_values"
    values: Optional[List[Any]] = None


class Histogram:
    """Class to represent a histogram.

    The attribute "type" denotes the type of the histogram.
    Set it to a proper value to clearly explain the type of the histogram value.
    ex. summary, average.

    For Summary class, the type attribute are "summary".
    The exported json looks like:
    system.loading.cpu_usage": {
        "summary": {
        "units": "percent",
        ...
        }
    }

    Attributes:
        name: The name of the histogram.
        type: The type of the histogram, ex. summary, average, etc.
        data: The data object of the histogram.
    """

    def __init__(
        self, name: str, histogram_type: str, data: HistogramData
    ) -> None:
        self.name = name
        self.type = histogram_type
        self.data = data

    def to_json_dict(self) -> Dict[str, Any]:
        """Convert the object to dictionary for formating json."""
        return {self.name: {self.type: self.data.to_json_dict()}}


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

    def summarize(self, extra_attrs: Dict[str, Any]) -> Histogram:
        """Get the result metrics.

        Args:
            extra_attrs: The extra attributes to form the result.

        Returns:
            The result metrics.
        """
        return Histogram(self.name, "summary", self._summarize(extra_attrs))

    @abc.abstractmethod
    def _summarize(self, extra_attrs: Dict[str, Any]) -> HistogramData:
        pass

    def record(self, *_args, **_kwargs) -> None:
        """Record the current metric value"""
        value = self._metric.value_fetcher()
        logger.debug("append value=%s", value)
        self._values.append(value)


class ListSummary(Summary):
    """Class for List style summary"""

    def _summarize(self, extra_attrs: Dict[str, Any]) -> HistogramValues:
        return HistogramValues(
            self._metric.units, "down", extra_attrs, values=self._values
        )


class DifferenceSummary(Summary):
    """Class for difference style summary"""

    def _summarize(self, extra_attrs: Dict[str, Any]) -> HistogramValue:
        value = self._values[0] - self._values[-1]
        return HistogramValue(self._metric.units, "down", value=value)
