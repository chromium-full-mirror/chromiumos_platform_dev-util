# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Timer utility"""

import abc
import datetime
import sched
import threading
import time
from typing import Callable

from cros_ca_lib.utils import logging as lg


logger = lg.logger


class TimerException(Exception):
    """Timer Exception.

    Attributes:
        msg: The message describes the exception.
    """

    def __init__(self, msg: str) -> None:
        super().__init__()
        self._msg = msg

    def __str__(self) -> str:
        return self._msg


class TimerTask(abc.ABC):
    """The class represents a scheduled timer task.

    User should not new this class directly.
    The API in Timer such as Timer.schedule_at_fixed_rate will return a
    TimerTask object.

    Attributes:
        _timer: The timer that schedule this task.
        name: The name of this task.
        _action: The action being called when the task is fired.
        _delay: The delayed seconds before the first run.
        _interval: The interval, in seconds, between each run.
        _event: The current event scheduled for running.
        _next_fire_time: The next time this task will be fired.
        _lock: The lock to protect the internal states.
    """

    def __init__(
        self,
        timer: "Timer",
        name: str,
        action: Callable,
        delay: float,
        interval: float,
    ) -> None:
        self._timer = timer
        self.name = name
        self._action = action
        self._delay = delay
        self._interval = interval
        self._event: sched.Event = None
        self._next_fire_time: float = None
        self._lock = threading.Lock()

    def _run(self) -> None:
        with self._lock:
            self._event = None
            if self.is_cancelled():
                logger.debug("task is cancelled, ignore %s", self)
                return
            if self._timer.is_shutdown():
                logger.debug("timer is shutdown, ignore %s", self)
                return
        logger.debug("%s fired", self)
        try:
            self._action()
        except:
            logger.exception("Fail to run %s", self)
            raise

        with self._lock:
            if not self.is_cancelled():
                self._timer._schedule(self)  # pylint: disable=protected-access

    def _calculate_next_delay(self) -> float:
        now = self._timer.now()
        if self._next_fire_time is None:
            # never ran before, use initial delay.
            self._next_fire_time = now + self._delay
        else:
            self._calculate_next_fire_time()
        return self._next_fire_time - now

    @abc.abstractmethod
    def _calculate_next_fire_time(self) -> None:
        pass

    def is_cancelled(self) -> bool:
        """Whether the task has been cancelled."""
        return self._timer is None

    def cancel(self) -> bool:
        """Cancel a scheduled task.

        It will not interrupt the task if it is running.

        Returns:
            True: Cancel the task successfully.
            False: Ignored, the task has already been cancelled.
        """
        logger.debug("cancel %s", self)
        with self._lock:
            if self.is_cancelled():
                logger.debug("%s has been cancelled before", self)
                return False
            if self._event is None:
                logger.debug("%s is running, just marked as cancelled", self)
            else:
                self._timer._cancel(  # pylint: disable=protected-access
                    self._event
                )
                self._event = None
            self._timer = None
            return True

    def __str__(self) -> str:
        return (
            f"{self.__class__.__name__}"
            f"[name: {self.name}, timer: {self._timer}, "
            f"next_fire_time: {self._next_fire_time}]"
        )


class _FixRateTask(TimerTask):
    """This class represents a scheduled fixed-rate timer task."""

    def _calculate_next_fire_time(self) -> None:
        self._next_fire_time += self._interval


class Timer:
    """The class represents a timer.

    An timer object schedules and runs tasks on a single thread.

    Attributes:
        name: The name of this timer.
        _scheduler: The sched.scheduler used for this timer.
        _thread: The thread running this timer.
        _condition: The condition used to protect the internal states.
        _shutdown: A flag indicating whether this timer is shutdown.
    """

    _DEFAUT_PRIORITY = 1

    def __init__(self, name: str) -> None:
        self.name = name
        self._scheduler = sched.scheduler(time.monotonic, time.sleep)
        self._thread: threading.Thread = None
        self._condition = threading.Condition()
        self._shutdown = False

        self._start()

    def now(self) -> float:
        """Get current time by the same API of the scheduler."""
        return self._scheduler.timefunc()

    def schedule_at_fixed_rate(
        self,
        task_name: str,
        action: Callable,
        delay: datetime.timedelta,
        interval: datetime.timedelta,
    ) -> TimerTask:
        """Schedule a timer task fired at fixed rate.

        Args:
            task_name: The name of this task.
            action: The action will be called when the task is fired.
            delay: The delay before the first run.
            interval: The interval between each run.

        Returns:
            A TimerTask object.
        """
        task = _FixRateTask(
            self,
            task_name,
            action,
            delay.total_seconds(),
            interval.total_seconds(),
        )
        logger.info("schedule %s", task)

        if not self._schedule(task):
            raise TimerException(f"{self} is shutdown")
        return task

    def _schedule(self, task: TimerTask) -> bool:
        if self._shutdown:
            logger.debug("%s is shutdown, ignore scheduling %s", self, task)
            return False
        # pylint: disable=protected-access
        delay = task._calculate_next_delay()
        task._event = self._scheduler.enter(
            delay, self._DEFAUT_PRIORITY, task._run
        )
        with self._condition:
            self._condition.notify()
        logger.debug("fire %s in %s seconds", task, delay)
        return True

    def _cancel(self, event: sched.Event) -> None:
        self._scheduler.cancel(event)

    def _run(self) -> None:
        logger.debug("starting timer: %s", self)
        while not self._shutdown:
            try:
                sleep = self._scheduler.run(blocking=False)
            except Exception as e:
                logger.warning(
                    "Continue %s with error: %s", self, e, exc_info=True
                )
            with self._condition:
                if not self._shutdown:
                    self._condition.wait(sleep)
        logger.debug("stop %s", self)

    def _start(self) -> None:
        self._thread = threading.Thread(
            name=f"Timer-{self.name}", target=self._run, daemon=True
        )
        self._thread.start()

    def shutdown(self, wait=True) -> None:
        """Shutdown the timer.

        This method will mark the timer as shutdown.

        Args:
            wait: wait for the timer thread to exit
            if wait is True. Defaults to True.
        """
        logger.info("shut down %s", self)
        self._shutdown = True
        with self._condition:
            self._condition.notify()
        if wait:
            self._thread.join()

    def is_shutdown(self) -> bool:
        return self._shutdown

    def __str__(self) -> str:
        return f"{self.__class__.__name__}[{self.name}]"
