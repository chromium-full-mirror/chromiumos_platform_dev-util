# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The CrOS local host implementation."""

# The following line disable the abstract function implementation warning.
# pylint: disable=W0223

import subprocess
import time

from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.chrome.service import Service

from lib.host import base_host


class CrosLocalHost(base_host.BaseHost):
    """A class representing a local Windows host"""

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

        command = "/usr/local/autotest/bin/autologin.py "
        # Don't keep state. Delete existing profile.
        # This is the only way it can work together with gaia login. Without
        # this option, if another account has already exists on the DUT,
        # the GAIA login would fail on the screen of "Choose your setup"
        # for personal use, for a child or for work.
        command += "-d "
        if option.arc:
            command += "--arc --no-arc-syncs "
        if option.username and option.password:
            command += f"-u {option.username} -p {option.password} "
        # TODO: Other Chrome options can be added here. These options
        #       can be set in the "option" class and applied to the
        #       autologin.py argument here.

        subprocess.run(
            command,
            shell=True,
            stdout=subprocess.DEVNULL,  # Supress the stdout and stderr
            stderr=subprocess.DEVNULL,
            check=True,
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

    def mute(self, flag=True):
        """Set the device speaker to mute.

        Args:
            flag: True if set to mute, Fale to unmute.
        """
        command = "cras_test_client "
        command += f"--mute {1 if flag else 0}"
        subprocess.run(
            command,
            shell=True,
            stdout=subprocess.DEVNULL,  # Supress the stdout and stderr
            stderr=subprocess.DEVNULL,
            check=True,
        )
