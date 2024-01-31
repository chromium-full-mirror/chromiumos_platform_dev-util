# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Script contains decorator to measure the time per some function."""
import inspect
import time

execution_times = []


def time_measure(special_arg_index: int = None):
    """Helper decorator used to calculate time used by specific
    execution function that annotated with it.

    Args:
        special_arg_index: Index of function argument passed
            that should be presents in collected time.
    """

    def decorator(func):
        def wrapper(*args, **kwargs):
            start_time = time.time()
            result = func(*args, **kwargs)
            end_time = time.time()
            callback_stack = inspect.stack()
            for i, _ in enumerate(callback_stack):
                if "execute" in callback_stack[i][3]:
                    execution_time = end_time - start_time
                    special_arg = ""
                    if special_arg_index is not None:
                        special_arg = " for " + args[special_arg_index]
                    execution_times.append(
                        (func.__name__ + special_arg, execution_time)
                    )
                    break
            return result

        return wrapper

    return decorator
