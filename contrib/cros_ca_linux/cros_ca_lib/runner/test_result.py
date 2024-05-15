# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Test result."""

import datetime
import enum
from typing import Any, Dict, Optional


class Result(enum.Enum):
    """Result enum."""

    NOT_RUN = 0
    PASS = 1
    FAIL = 2


class TestResult:
    """The test result."""

    def __init__(self):
        self.name: str = ""
        self.iteration: Optional[int] = None
        self.location: str = ""
        self.result: Result = Result.NOT_RUN
        self.error: Optional[str] = None

        self.remote_result: Result = Result.NOT_RUN
        self.remote_error: Optional[str] = None
        self.remote_start_time: Optional[datetime.datetime] = None
        self.remote_end_time: Optional[datetime.datetime] = None

        self.local_result: Result = Result.NOT_RUN
        self.local_error: Optional[str] = None
        self.local_start_time: Optional[datetime.datetime] = None
        self.local_end_time: Optional[datetime.datetime] = None

    def to_dict(self) -> Dict[str, Any]:
        """Convert to a json ready dict"""
        return dict(
            name=self.name,
            iteration=self.iteration,
            location=self.location,
            result=self.result.name,
            error=self.error,
            remote_result=self.remote_result.name,
            remote_error=self.remote_error,
            remote_start_time=(
                self.remote_start_time.isoformat()
                if self.remote_start_time
                else None
            ),
            remote_end_time=(
                self.remote_end_time.isoformat()
                if self.remote_end_time
                else None
            ),
            local_result=self.local_result.name,
            local_error=self.local_error,
            local_start_time=(
                self.local_start_time.isoformat()
                if self.local_start_time
                else None
            ),
            local_end_time=(
                self.local_end_time.isoformat() if self.local_end_time else None
            ),
        )


def string_to_result(name: str) -> Result:
    """Converts string representation of a enum member to the enum object."""
    if not name:
        return Result.NOT_RUN
    if name not in Result.__members__:
        raise ValueError(f"Invalid result name: {name}")
    return Result[name]


def from_dict(data: Dict[str, Any]) -> TestResult:
    """Create a TestResult object manually from the dictionary.

    Args:
        data: The data to create from.
    """
    r = TestResult()
    r.name = data.get("name")
    r.iteration = data.get("iteration")
    r.location = data.get("location")
    r.result = string_to_result(data.get("result"))
    r.error = data.get("error")

    r.remote_result = string_to_result(data.get("remote_result"))
    r.remote_error = data.get("remote_error")
    r.remote_start_time = None
    if data.get("remote_start_time"):
        r.remote_start_time = datetime.datetime.fromisoformat(
            data["remote_start_time"]
        )

    r.remote_end_time = None
    if data.get("remote_end_time"):
        r.remote_end_time = datetime.datetime.fromisoformat(
            data["remote_end_time"]
        )

    r.local_result = string_to_result(data.get("local_result"))
    r.local_error = data.get("local_error")
    r.local_start_time = None
    if data.get("local_start_time"):
        r.local_start_time = datetime.datetime.fromisoformat(
            data["local_start_time"]
        )
    r.local_end_time = None
    if data.get("local_end_time"):
        r.local_end_time = datetime.datetime.fromisoformat(
            data["local_end_time"]
        )
    return r
