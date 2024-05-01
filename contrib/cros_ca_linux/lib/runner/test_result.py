# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Test result."""

import datetime
import enum


class Result(enum.Enum):
    """Result enum."""

    NOT_RUN = 0
    PASS = 1
    FAIL = 2


class TestResult:
    """The test result."""

    def __init__(self):
        self.name: str = ""
        self.location: str = ""
        self.result: Result = Result.NOT_RUN
        self.error: str = None

        self.remote_result: Result = Result.NOT_RUN
        self.remote_error: str = None
        self.remote_start_time: datetime.datetime = None
        self.remote_end_time: datetime.datetime = None

        self.local_result: Result = Result.NOT_RUN
        self.local_error: str = None
        self.local_start_time: datetime.datetime = None
        self.local_end_time: datetime.datetime = None

    def toDict(self) -> dict:
        return dict(
            name=self.name,
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


def fromDict(data: dict) -> TestResult:
    # Create a TestResult object manually from the dictionary
    r = TestResult()
    r.name = data.get("name")
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
