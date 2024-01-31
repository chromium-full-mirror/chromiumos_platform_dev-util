# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable = R0902, W1203
"""
Script contain ProcessMeasurements class.
"""

import time

import psutil
from psutil import NoSuchProcess

from common.common_utility import remove_multiple_from_list
from common.constants import LoggingConfig
from common.iqr import calculate_iqr_outliers

RESULTS_LOGGER = LoggingConfig.get_logger(False)


class ProcessMeasurements:
    """ProcessMeasurements class that responsible to handle
    measurements of process.
    """

    def __init__(self, proces: psutil.Process):
        """Constructor of ProcessMeasurements class."""
        self._proces = proces
        self._proces_id = proces.pid
        self._proces_name = proces.name()
        self._cpu_percent = []
        self._memory_percent = []
        self._user_cpu_times = []
        self._system_cpu_times = []
        self._last_sys_cpu_times = None
        self._last_proc_cpu_times = None
        self.average_dict = None

    def appends_process_measurements(self):
        """Method to take measurements of proces and appends them
        to the instance variable lists.
        """
        proces = self._proces
        try:
            if proces is not None and proces.is_running():
                cpu = self.calculate_cpu_percent()
                mem = proces.memory_percent()
                cpu_times = proces.cpu_times()
                user_cpu_times = cpu_times.user
                system_cpu_times = cpu_times.system
                self._cpu_percent.append(cpu)
                self._memory_percent.append(mem)
                self._user_cpu_times.append(user_cpu_times)
                self._system_cpu_times.append(system_cpu_times)
        except NoSuchProcess:
            pass

    def print_average_with_outlier(self):
        """Method to print the average of instance variable lists after
        pass the lists to calculate_iqr_outliers method to determine
        the outliers value should be removed from original list before
        calculating the average, and save those average in average_dict
        dictionary to be used in Benchmarking class when print the summation.
        """
        cpu = self._cpu_percent
        mem = self._memory_percent
        user_cpu_times = self._user_cpu_times
        system_cpu_times = self._system_cpu_times
        process_name = self._proces_name
        process_id = self._proces_id

        if len(cpu) > 0:
            remove_multiple_from_list(
                cpu, calculate_iqr_outliers(cpu)["Outliers"]
            )
        if len(mem) > 0:
            remove_multiple_from_list(
                mem, calculate_iqr_outliers(mem)["Outliers"]
            )
        if len(user_cpu_times) > 0:
            remove_multiple_from_list(
                user_cpu_times,
                calculate_iqr_outliers(user_cpu_times)["Outliers"],
            )
        if len(system_cpu_times) > 0:
            remove_multiple_from_list(
                system_cpu_times,
                calculate_iqr_outliers(system_cpu_times)["Outliers"],
            )

        average_dict = {
            "cpu_percent": sum(cpu) / len(cpu) if len(cpu) > 0 else -1,
            "memory_percent": sum(mem) / len(mem) if len(mem) > 0 else -1,
            "user_cpu_times": (
                sum(user_cpu_times) / len(user_cpu_times)
                if len(user_cpu_times) > 0
                else -1
            ),
            "system_cpu_times": (
                sum(system_cpu_times) / len(system_cpu_times)
                if len(system_cpu_times) > 0
                else -1
            ),
        }
        self.average_dict = average_dict
        RESULTS_LOGGER.info(
            f"average results for process name = {process_name} and pid = {process_id} is: "
        )
        RESULTS_LOGGER.info(
            f"cpu percent = ({average_dict.get('cpu_percent')}) %"
        )
        RESULTS_LOGGER.info(
            f"memory percent = ({average_dict.get('memory_percent')}) %"
        )
        RESULTS_LOGGER.info(
            f"user cpu time = ({average_dict.get('user_cpu_times')}) seconds"
        )
        RESULTS_LOGGER.info(
            f"system cpu time = ({average_dict.get('system_cpu_times')}) seconds"
        )

        RESULTS_LOGGER.info(f"{15*'-'}**********{15*'-'}")

    def calculate_cpu_percent(self):
        """Return a float representing the current process CPU
        utilization as a percentage.
        it compares process times to system CPU times elapsed
        since last call, returning immediately (non-blocking).
        That means that the first time this is called it will
        return a meaningful 0.0 value.

        Returns:
            Current process CPU utilization as a percentage.
        """
        num_cpus = psutil.cpu_count() or 1

        def timer():
            _timer = getattr(time, "monotonic", time.time)
            return _timer() * num_cpus

        st1 = self._last_sys_cpu_times
        pt1 = self._last_proc_cpu_times
        st2 = timer()
        pt2 = self._proces.cpu_times()
        if st1 is None or pt1 is None:
            self._last_sys_cpu_times = st2
            self._last_proc_cpu_times = pt2
            return 0.0
        delta_proc = (pt2.user - pt1.user) + (pt2.system - pt1.system)
        delta_time = st2 - st1
        # reset values for next call in case of interval == None
        self._last_sys_cpu_times = st2
        self._last_proc_cpu_times = pt2

        try:
            # This is the utilization split evenly between all CPUs.
            # E.g. a busy loop process on a 2-CPU-cores system at this
            # point is reported as 50% instead of 100%.
            overall_cpus_percent = (delta_proc / delta_time) * 100
        except ZeroDivisionError:
            return 0.0

        single_cpu_percent = overall_cpus_percent * num_cpus
        return round(single_cpu_percent, 1)

    def __eq__(self, other):
        """Overloading of equal method for ProcessMeasurements class
        to make the set comparison depends on the proces rather
        than all instance variable in the class.

        Args:
            other: Other ProcessMeasurements instance.

        Returns:
            True if ProcessMeasurements proces isinstance or equal
                to other ProcessMeasurements proces, else false.
        """
        if not isinstance(other, ProcessMeasurements):
            return False
        return self._proces == other._proces

    def __hash__(self):
        """Overloading of hash method for ProcessMeasurements class
        to make the set comparison depends on the proces rather
        than all instance variable in the class.

        Returns:
            Hash of ProcessMeasurements proces.
        """
        return hash(self._proces)

    def __repr__(self):
        """Overloading of representation method for ProcessMeasurements class.

        Returns:
            Specific format of ProcessMeasurements class when it needs to
                be presented.
        """
        return (
            f"( proces = {self._proces} \n "
            + f"cpu_percent = {self._cpu_percent} \n"
            + f"memory_percent = {self._memory_percent} \n "
            + f"user_cpu_times = {self._user_cpu_times} \n"
            + f"system_cpu_times = {self._system_cpu_times})"
        )
