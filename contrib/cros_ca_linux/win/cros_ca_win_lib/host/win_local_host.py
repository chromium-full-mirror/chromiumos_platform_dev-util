# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The Windows local host implementation."""

# The following line disable the abstract function implementation warning.
# pylint: disable=W0223

from cros_ca_lib.host import base_host
from cros_ca_lib.utils import logging as lg
from cros_ca_win_lib.utils import audio
from selenium import webdriver


logger = lg.logger


class WinLocalHost(base_host.BaseHost):
    """A class representing a local Windows host"""

    def name(self) -> str:
        """The name of the host"""
        # TODO: get version, platform, spec, etc.
        return "Windows Local"

    def _connect(self):
        """Connects to the local host."""

    def close(self):
        """Close the local host."""

    def update_src_version(self):
        """Update the srce code"""

    def chrome_webdriver(
        self, opts: webdriver.ChromeOptions
    ) -> webdriver.Chrome:
        """Create a selenium Chrome webdriver

        It connects to the browser on the DUT.

        Args:
            opts: The chrome options. Test case can pass the options
                  it desires to use.

        Returns:
            selenium webdriver.
        """

        return webdriver.Chrome(options=opts)

    def auto_login(self, option: base_host.LoginOption):
        """Automatically log into the DUT.

        Args:
            option: Login option.
        """

    def mute(self, flag: bool = True) -> None:
        """Mute or unmute all active speakers of the device.

        Args:
            flag: Mute the device if True, otherwise unmute.
        """
        logger.debug("%s the device sound", "Mute" if flag else "Unmute")
        for dev in audio.get_all_active_render_devices():
            audio.mute_device(dev, flag)

    def set_volume(self, percent: int) -> None:
        """Set the sound volume of all the active device speakers.

        Args:
            percent: The volume percentage, ranging from 0 to 100.
        """
        logger.debug("Set device sound volume to %s", percent)
        for dev in audio.get_all_active_render_devices():
            audio.set_device_volume(dev, percent)
