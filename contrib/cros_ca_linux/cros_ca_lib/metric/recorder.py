# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Classes about recorder"""

import abc
import concurrent.futures
import datetime
import json
from pathlib import Path
import time
import types
from typing import Any, Dict, List, Optional, Type

from cros_ca_lib.metric import summary
from cros_ca_lib.utils import logging as lg
from cros_ca_lib.utils import timer


logger = lg.logger


class Recorder(abc.ABC):
    """Class to represent a recorder.

    Attributes:
        name: The name of this recorder.
        results: The recorded results.
        summaries: The summaries to record
    """

    def __init__(
        self,
        name: str,
        *summaries: summary.Summary,
    ) -> None:
        self.name = name
        self._summaries: List[summary.Summary] = [
            s for s in summaries if s is not None
        ]
        self.results: Dict[str, Any] = {}

    def start(self) -> None:
        self._start()

    def _start(self) -> None:
        pass

    def stop(self) -> None:
        self._stop()

    def _stop(self) -> None:
        self._summarize()

    def _record(self) -> None:
        for m in self._summaries:
            m.record()

    def _extra_attrs(self) -> Dict[str, Any]:
        """All summaries recorded will have these extra attributes."""
        return {}

    def add_result(self, result: Dict[str, Any]) -> None:
        """Add result to the recorder summary"""
        self.results.update(result)

    def add_histograms(self, histograms: List[summary.Histogram]) -> None:
        """Add histogram to the recorder summary"""
        for histogram in histograms:
            self.results.update(histogram.to_json_dict())

    def _summarize(self) -> None:
        """Summarize using all the summaries"""
        for s in self._summaries:
            self.add_histograms([s.summarize(self._extra_attrs())])

    def save(self, path: Path) -> None:
        """Save the summarized result to path/results-chart.json"""
        metrics_file = path / "results-chart.json"
        with open(metrics_file, "w", encoding="utf-8") as f:
            json.dump(self.results, f, indent=2)

    def __enter__(self) -> "TimerRecorder":
        self._start()
        return self

    def __exit__(
        self,
        _exc_type: Optional[Type[BaseException]],
        _exc_value: Optional[BaseException],
        _traceback: Optional[types.TracebackType],
    ) -> bool:
        try:
            self._stop()
        except Exception:
            logger.exception("Fail to stop %s", self)

        # Return False to leave the exception handling to the caller.
        return False

    def __str__(self) -> None:
        return f"{self.__class__.__name__}[{self.name}]"


class SimpleRecorder(Recorder):
    """Recorder to record nothing except the data added by add_result."""

    # disable pylint due to the pylint bug
    # https://github.com/pylint-dev/pylint/issues/2270
    def __init__(  # pylint: disable=useless-super-delegation
        self, name: str
    ) -> None:
        super().__init__(name)


class BeginEndRecorder(Recorder):
    """Recorder to record at begin and end."""

    def _start(self) -> None:
        self._record()

    def _stop(self) -> None:
        self._record()
        super()._stop()


class TimerRecorder(Recorder):
    """Recorder to record at fix interval.

    Attributes:
        wait_stop: Wait for all worker threads to stop when stop the recorder.
    """

    def __init__(
        self,
        name: str,
        *summaries: summary.Summary,
        delay: datetime.timedelta = datetime.timedelta(seconds=1),
        interval: datetime.timedelta = datetime.timedelta(seconds=1),
        max_workers: int = 5,
        wait_stop=True,
    ) -> None:
        super().__init__(name, *summaries)
        self._delay = delay
        self._interval = interval
        self._max_workers = max_workers
        self.wait_stop = wait_stop

        self._timer: timer.Timer = None
        self._workers: concurrent.futures.ThreadPoolExecutor = None
        self._begin = None
        self._timestamps = []

    def _extra_attrs(self) -> Dict[str, Any]:
        """Add interval attribute to all recorded summaries."""
        return {"interval": self._timestamp_name()}

    def _timestamp_name(self) -> str:
        return f"{self.name}.t"

    def _start(self) -> None:
        name = f"Recorder-{self.name}"
        self._workers = concurrent.futures.ThreadPoolExecutor(
            self._max_workers, thread_name_prefix=name
        )
        self._timer = timer.Timer(name)

        self._timer.schedule_at_fixed_rate(
            self.name, self._record, self._delay, self._interval
        )
        self._begin = time.time()

    def _record(self) -> None:
        self._timestamps.append(time.time() - self._begin)
        for m in self._summaries:
            self._workers.submit(m.record)

    def _stop(self) -> None:
        if self._timer is not None:
            self._timer.shutdown()
        if self._workers is not None:
            self._workers.shutdown(self.wait_stop)
        self.add_histograms([self._timestamp_histogram()])
        super()._stop()

    def _timestamp_histogram(self) -> summary.Histogram:
        data = summary.HistogramValues("s", "down", values=self._timestamps)
        return summary.Histogram(self._timestamp_name(), "summary", data)
