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

    def chrome_webdriver(
        self, opts: webdriver.ChromeOptions
    ) -> webdriver.Chrome:
        """Create a selenium Chrome webdriver.

        The webdriver connects to the browser on the DUT.

        Returns:
            selenium webdriver.
        """

        subprocess.run(
            ["/usr/local/autotest/bin/autologin.py"],
            stdout=subprocess.DEVNULL,  # Supress the stdout and stderr
            stderr=subprocess.DEVNULL,
            check=True,
        )
        time.sleep(2)

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
