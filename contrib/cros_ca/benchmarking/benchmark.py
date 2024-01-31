# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable= W1203
"""Script contains Benchmark class."""

from benchmarking.process_measurements import ProcessMeasurements
from common.constants import LoggingConfig

LOGGER = LoggingConfig.get_logger()
RESULTS_LOGGER = LoggingConfig.get_logger(False)


class Benchmark:
    """Benchmark class used to benchmark set of process
    used while run the use cases.
    """

    def __init__(self):
        """Constructor of Benchmark class to have
        empty set of ProcessMeasurements type at initialization.
        """
        self.__process_measurements_set = set()

    def get_process_measurements_set(self):
        """
        Getter method for __process_measurements_set.
        Returns:
            __process_measurements_set
        """
        return self.__process_measurements_set

    def update_process_measurements_set(self, proces_measure):
        """Method to update __process_measurements_set by add
        the ProcessMeasurements to the set.

        Args:
            proces_measure: ProcessMeasurements to add to the set.
        """
        self.get_process_measurements_set().add(proces_measure)

    def measure(self, proces_measurements: ProcessMeasurements):
        """Method to run measurements operation on ProcessMeasurements set.
        if ProcessMeasurements passed by parameter not exist in the set before,
        then it will take the measurements for it and added it to the set,
        else it will take the measurements for current process in the set.

        Args:
            proces_measurements: ProcessMeasurements to run the measure for it.
        """
        for current_proces_measure in self.get_process_measurements_set():
            if current_proces_measure == proces_measurements:
                current_proces_measure.appends_process_measurements()
                break
        else:
            proces_measurements.appends_process_measurements()
            self.update_process_measurements_set(proces_measurements)

    def print_average(self):
        """Method to print result of measurements for
        the __process_measurements_set.
        """
        LOGGER.info(f"start print_average method in {self.__class__.__name__}.")
        for proces_measurement in self.get_process_measurements_set():
            proces_measurement.print_average_with_outlier()

        app_cpu_percent_sum = 0.0
        app_memory_percent_sum = 0.0
        app_user_cpu_times_sum = 0.0
        app_system_cpu_times_sum = 0.0

        for proces_measurement in self.get_process_measurements_set():
            average_dict = proces_measurement.average_dict
            cpu_percent = average_dict.get("cpu_percent")
            memory_percent = average_dict.get("memory_percent")
            user_cpu_times = average_dict.get("user_cpu_times")
            system_cpu_times = average_dict.get("system_cpu_times")
            app_cpu_percent_sum += cpu_percent if cpu_percent > 0 else 0
            app_memory_percent_sum += (
                memory_percent if memory_percent > 0 else 0
            )
            app_user_cpu_times_sum += (
                user_cpu_times if user_cpu_times > 0 else 0
            )
            app_system_cpu_times_sum += (
                system_cpu_times if system_cpu_times > 0 else 0
            )
        RESULTS_LOGGER.info(f"{15*'-'}sum per all process{15*'-'}")
        RESULTS_LOGGER.info(
            f"sum of app cpu percent = ({app_cpu_percent_sum}) %"
        )
        RESULTS_LOGGER.info(
            f"sum of app memory percent = ({app_memory_percent_sum}) %"
        )
        RESULTS_LOGGER.info(
            f"sum of app user cpu time = ({app_user_cpu_times_sum}) seconds"
        )
        RESULTS_LOGGER.info(
            f"sum of app system cpu time = ({app_system_cpu_times_sum}) "
            "seconds"
        )
        LOGGER.info(
            f"end of print_average method in {self.__class__.__name__}."
        )
