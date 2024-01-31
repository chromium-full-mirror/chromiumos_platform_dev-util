# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable= W1203
"""Script contains performance decorator used to running
benchmarking thread.
"""

import platform
import threading

import psutil
from psutil import NoSuchProcess

from benchmarking.benchmark import Benchmark
from benchmarking.benchmarking_exception import BenchmarkingException
from benchmarking.process_measurements import ProcessMeasurements
from common.constants import ProcessNames, LoggingConfig

LOGGER = LoggingConfig.get_logger()


def measurements_thread(process_name, stop_event, benchmarking):
    """Helper function that's responsible for measuring the device
    resource usage in parallel with the current process.

    Args:
        benchmarking: Instance of Benchmark class used to benchmark
            set of process used while run the use cases.
        process_name: Tuple of process name needs to take measurements for it.
        stop_event: A flag to stop the measurement.
    """

    def performance_measurements():
        """Thread function responsible to take measurements every 100ms
        for pre-defined process name.
        """
        try:
            for proc in psutil.process_iter():
                for proces_name in process_name:
                    p_name = proc.name()
                    if (
                        proces_name in p_name
                        and p_name
                        not in ProcessNames.PROCESS_NAME_TO_BE_EXCLUDE
                    ):
                        benchmarking.measure(ProcessMeasurements(proc))
        except NoSuchProcess as exc:
            stop_event.set()  # Stop the measurement if the process is not found
            LOGGER.info(
                f"stop performance_measurements thread, "
                f"because proces not found with exception : {exc}"
            )

        if not stop_event.is_set():
            threading.Timer(
                interval=0.1, function=performance_measurements
            ).start()

    # Start the initial measurement
    LOGGER.info("start performance_measurements thread.")
    performance_measurements()


def validate_process_name(process_name):
    """Function to validate process name passed by running platform.

    Args:
        process_name: Tuple of process name needs to take measurements for it.

    Raises:
        BenchmarkingException: In case process name passed is None
            or empty or if proces name not contains .exe for Windows
            platform or proces name contains .exe for linux platform.
    """
    LOGGER.info("start validate_process_name method.")
    if process_name is None or len(process_name) == 0:
        LOGGER.info("validate_process_name failed with BenchmarkingException.")
        raise BenchmarkingException("process_names should not be empty.")
    windows_app_extension = ".exe"
    for proc in process_name:
        if platform.system() == ProcessNames.WINDOWS_PLATFORM_TYPE:
            if windows_app_extension not in proc.lower():
                LOGGER.info(
                    "validate_process_name failed with BenchmarkingException."
                )
                raise BenchmarkingException(
                    "Windows proces name should ends with .exe"
                )
        else:
            if windows_app_extension in proc.lower():
                LOGGER.info(
                    "validate_process_name failed with BenchmarkingException."
                )
                raise BenchmarkingException(
                    "Linux proces name should not ends with .exe"
                )
    LOGGER.info("end of validate_process_name method.")


def performance(process_name: tuple = None):
    """Decorator function that's responsible for measuring the device
    resource usage in parallel with the current process."""

    def decorator(func):
        def wrapper(*args, **kwargs):
            LOGGER.info(
                "start executing performance decorator with "
                f"process_name = {process_name}."
            )
            validate_process_name(process_name)
            benchmarking = Benchmark()
            stop_event = threading.Event()
            measurement_thread = threading.Thread(
                target=measurements_thread,
                args=(process_name, stop_event, benchmarking),
            )
            measurement_thread.start()
            # Run the function
            result = func(*args, **kwargs)
            stop_event.set()
            measurement_thread.join()
            benchmarking.print_average()
            LOGGER.info("ending of execute performance decorator.")
            return result

        return wrapper

    return decorator
