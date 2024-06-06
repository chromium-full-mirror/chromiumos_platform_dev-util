# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Utilities for controling mechanisms."""

from typing import Callable, TypeVar

from cros_ca_lib.utils import logging as lg


logger = lg.logger
T = TypeVar("T")


class RetryableError(Exception):
    """The error can be retried.

    The function that raised this error might succeed after several retries.
    """


def retry(func: Callable[[], T], retry_times=3) -> T:
    """Retry the give function if encounter retryable errors"""
    for i in range(retry_times + 1):
        try:
            return func()
        except RetryableError as e:
            if i == retry_times:
                raise e
            logger.warning("Retry %s for error: %s", i + 1, e)
