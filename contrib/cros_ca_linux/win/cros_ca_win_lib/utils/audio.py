# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Utilities to control audio"""

from typing import List

from pycaw import constants as pycaw_const
from pycaw import pycaw


def mute_device(dev: pycaw.AudioDevice, mute: bool) -> None:
    """Mute the given device.

    Args:
        dev: The device to mute.
        mute: Mute the device if True, otherwise unmute.
    """

    dev.EndpointVolume.SetMute(mute, None)


def set_device_volume(dev: pycaw.AudioDevice, percent: int) -> None:
    """Set the volume of the given device.

    Args:
        dev: The device to set the volume for.
        percent: The volume percentage, ranging from 0 to 100.
    """
    dev.EndpointVolume.SetMasterVolumeLevelScalar(percent / 100, None)


def get_all_devices(
    flow_type: pycaw_const.EDataFlow, state: pycaw_const.DEVICE_STATE
) -> List[pycaw.AudioDevice]:
    """List all the devices in the system by the flow type and state.

    Args:
        flow_type: The flow type of the device to filter.
        state: the state of the device to filter.

    Returns:
        A list of all devices that match the given flow type and state.
    """
    devices = []
    enumerator = pycaw.AudioUtilities.GetDeviceEnumerator()

    collection = enumerator.EnumAudioEndpoints(
        flow_type.value,
        state.value,
    )
    if collection is None:
        return devices

    for i in range(collection.GetCount()):
        dev = collection.Item(i)
        if dev is not None:
            devices.append(pycaw.AudioUtilities.CreateDevice(dev))
    return devices


def get_all_active_render_devices() -> None:
    """List all active render devices in the system.

    Returns:
        A list of all active render devices in the system.
    """
    return get_all_devices(
        pycaw_const.EDataFlow.eRender, pycaw_const.DEVICE_STATE.ACTIVE
    )
