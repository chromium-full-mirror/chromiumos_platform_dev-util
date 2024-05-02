# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""PythonWeb test case."""

# The following line disable the abstract function implementation warning.
# pylint: disable=W0223

import time

from cros_ca_lib.host import base_host
from cros_ca_lib.testing.lib import chrome_browser_test as cbt
from selenium.webdriver import Remote
from selenium.webdriver.common.keys import Keys


class PythonWeb(cbt.ChromeBrowserTest):
    """PythonWeb test case for exploring the python.org web"""

    parameters = {"": 2, "3_tab": 3}  # Tab loop number

    def _run(self) -> None:
        self.dut.auto_login(base_host.LoginOption())  # Fake login option.
        driver: Remote = self.get_webdriver()
        self.logger.debug(f'Test parameter: "{self.param}"')
        loop = self.parameter
        for i in range(loop):
            self.logger.info(f"Start the testing with tab {i+1}.")

            old_tab = driver.current_window_handle
            self.run_python_web(driver)
            driver.switch_to.new_window("tab")
            new_tab = driver.current_window_handle
            driver.switch_to.window(old_tab)
            driver.close()  # Close the old tab
            driver.switch_to.window(new_tab)
        driver.close()

    def run_python_web(self, driver: Remote):
        driver.get("https://www.python.org")
        time.sleep(2)
        self.logger.info(f"Current web page title:  {driver.title}")
        search_bar = driver.find_element("name", "q")
        search_bar.clear()
        search_bar.send_keys("pip")
        time.sleep(2)
        search_bar.send_keys(Keys.RETURN)
        self.logger.info(f"Current web page url: {driver.current_url}")
        time.sleep(3)
