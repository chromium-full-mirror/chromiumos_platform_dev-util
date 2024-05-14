# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""System monitor file"""

import contextlib
import ctypes
import threading
from typing import Optional

import psutil
import pythoncom
import wmi


@contextlib.contextmanager
def connect_wmi(namespace: Optional[str] = None) -> wmi.WMI:
    is_main_thread = threading.current_thread() is threading.main_thread()
    if not is_main_thread:
        pythoncom.CoInitialize()
    try:
        if namespace is not None:
            yield wmi.WMI(namespace=namespace)
        else:
            yield wmi.WMI()
    finally:
        if not is_main_thread:
            pythoncom.CoUninitialize()


def get_cpu_usage() -> int:
    """Get cpu usage by wmi."""
    with connect_wmi() as c:
        percent = c.Win32_Processor()[0].LoadPercentage
        return 0 if percent is None else percent


def get_memory() -> float:
    """Get memory usage."""
    return psutil.virtual_memory()


def get_battery_discharge_rate() -> int:
    """Get battery discharge rate in mW."""
    with connect_wmi(namespace="wmi") as c:
        return c.batterystatus()[0].dischargeRate


def get_battery_sensor():
    """Get battery info."""
    battery = psutil.sensors_battery()
    return battery


class SYSTEM_BATTERY_STATE(ctypes.Structure):
    """Battery state to show."""

    _fields_ = [
        ("AcOnLine", ctypes.c_ubyte),
        ("BatteryPresent", ctypes.c_ubyte),
        ("Charging", ctypes.c_ubyte),
        ("Discharging", ctypes.c_ubyte),
        ("Spare1", ctypes.c_ubyte * 4),
        ("MaxCapacity", ctypes.c_ulong),
        ("RemainingCapacity", ctypes.c_ulong),
        ("Rate", ctypes.c_ulong),
        ("EstimatedTime", ctypes.c_ulong),
        ("DefaultAlert1", ctypes.c_ulong),
        ("DefaultAlert2", ctypes.c_ulong),
    ]


def get_battery_state() -> Optional[SYSTEM_BATTERY_STATE]:
    """Retrieve the system battery state obtained through an API.

    Returns:
        SYSTEM_BATTERY_STATE, None if fail to retrieve.
    """
    # Load required DLLs
    powrprof = ctypes.windll.LoadLibrary("powrprof.dll")

    # Access the API
    result = SYSTEM_BATTERY_STATE()
    retval = powrprof.CallNtPowerInformation(
        # SystemBatteryState
        5,
        None,
        0,
        ctypes.byref(result),
        ctypes.sizeof(result),
    )

    return result if retval == 0 else None


def get_battery_max_capacity() -> Optional[int]:
    """Retrieve the system battery max capacity in mWh.

    Returns:
        System battery max capacity in mWh, None if fail to retrieve.
    """
    battery_state = get_battery_state()
    if battery_state is None:
        return None
    return battery_state.MaxCapacity


def get_battery_remaining_capacity() -> Optional[int]:
    """Retrieve the system battery remaining capacity in mWh.

    Returns:
        System battery remaining capacity in mWh, None if fail to retrieve.
    """
    battery_state = get_battery_state()
    if battery_state is None:
        return None
    return battery_state.RemainingCapacity
