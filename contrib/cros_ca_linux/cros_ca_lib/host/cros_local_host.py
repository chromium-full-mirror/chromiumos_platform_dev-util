# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The CrOS local host implementation."""

# The following line disable the abstract function implementation warning.
# pylint: disable=W0223

import subprocess
import time

from cros_ca_lib.host import base_host
from cros_ca_lib.utils import logging as lg
from cros_ca_lib.utils.autotest import autologin
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.chrome.service import Service


logger = lg.logger


class CrosLocalHost(base_host.BaseHost):
    """A class representing a local Windows host"""

    def name(self) -> str:
        """The name of the host"""
        # TODO: get version, platform, spec, etc.
        return "CrOS Local"

    def _connect(self):
        """Connects to the local host."""

    def close(self):
        """Close the local host."""

    def update_src_version(self):
        """Update the src code"""

    def auto_login(self, option: base_host.LoginOption):
        """Automatically log into the DUT.

        Args:
            option: Login option.

        Raises:
            RuntimeError: if autologin fails.
        """

        autologin.autologin(
            # This is the only way it can work together with gaia login. Without
            # this option, if another account has already exists on the DUT,
            # the GAIA login would fail on the screen of "Choose your setup"
            # for personal use, for a child or for work.
            dont_override_profile=True,
            arc=option.arc,
            no_arc_syncs=option.arc,
            username=option.username,
            password=option.password,
            extra_args=option.chrome_args,
            enable_features=option.enable_features,
        )
        time.sleep(2)

    def chrome_webdriver(
        self, opts: webdriver.ChromeOptions
    ) -> webdriver.Chrome:
        """Create a selenium Chrome webdriver.

        The webdriver connects to the browser on the DUT.

        Args:
            opts: The chrome options. Test case can pass the options
                  it desires to use.

        Returns:
            selenium webdriver.
        """

        # TODO: For the webdriver to work on CrOS, the device must have
        #       an open Chrome window. This is achieved by doing auto_login().
        #       However, if auto_login() not called, we should find a way
        #       to bring up the Chrome Window automatically.

        debugging_port = 0
        with open(
            "/home/chronos/DevToolsActivePort", "r+", encoding="utf-8"
        ) as f:
            file_data = f.read()
            lines = file_data.splitlines()  # Split the string into lines
            first_line = lines[
                0
            ].strip()  # Get the first line and remove whitespaces
            debugging_port = int(
                first_line.split()[0]
            )  # Convert the first word to integer
        if debugging_port == 0:
            raise Exception("no debugging port found")

        ip_address = "127.0.0.1"
        options = opts if opts is not None else Options()
        options.add_experimental_option(
            "debuggerAddress", f"{ip_address}:{debugging_port}"
        )
        # Change chrome driver path accordingly
        s = Service("/usr/local/chromedriver/chromedriver")
        return webdriver.Chrome(service=s, options=options)

    def mute(self, flag=True) -> None:
        """Mute or unmute all active speakers of the device.

        Args:
            flag: Mute the device if True, otherwise unmute.
        """
        logger.debug("%s the device sound", "Mute" if flag else "Unmute")
        mute = "true" if flag else "false"
        command = f"cras_tests control set_mute {mute}"
        subprocess.run(
            command,
            shell=True,
            stdout=subprocess.DEVNULL,  # Supress the stdout and stderr
            stderr=subprocess.DEVNULL,
            check=True,
        )

    def set_volume(self, percent: int) -> None:
        """Set the sound volume of all the active device speakers.

        Args:
            percent: The volume percentage, ranging from 0 to 100.
        """
        logger.debug("Set device sound volume to %s", percent)
        command = f"cras_tests control set_volume {percent}"
        subprocess.run(
            command,
            shell=True,
            stdout=subprocess.DEVNULL,  # Supress the stdout and stderr
            stderr=subprocess.DEVNULL,
            check=True,
        )
