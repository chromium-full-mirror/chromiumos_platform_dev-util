# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The Windows local host implementation."""

# The following line disable the abstract function implementation warning.
# pylint: disable=W0223

from cros_ca_lib.host import base_host
from selenium import webdriver


class WinLocalHost(base_host.BaseHost):
    """A class representing a local Windows host"""

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

    def mute(self, flag=True):
        """Set the device speaker to mute.

        Args:
            flag: True if set to mute, Fale to unmute.
        """
        # TODO: To be implemented.
