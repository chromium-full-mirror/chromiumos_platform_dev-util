#!/usr/bin/env python3
# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable= E1120, W1203
"""Browsing four websites use case."""
import xml.etree.ElementTree as ET

from automated_use_cases.common_utility.get_website_util import (
    load_website_by_url,
    open_new_blank_tab,
)
from benchmarking.benchmarking_test import (
    BenchmarkingTest,
)
from benchmarking.performance_decorator import performance
from benchmarking.platform_benchmarking import setup, get_browser_process_name
from common.constants import DriversAndAppsPaths, LoggingConfig

LOGGER = LoggingConfig.get_logger()

BROWSER_PROCESS_NAME = get_browser_process_name()


class BrowseFourWebsiteTest(BenchmarkingTest):
    """Browsing four websites use case."""

    def __init__(self):
        self.urls = None
        self.xml_file_path = DriversAndAppsPaths.XML_WEBSITES_URL_PATH

    def set_up(self):
        """This method will perform preparing of website to be loaded
        by reading the XML file and get the websites url."""
        LOGGER.info(f"start set_up method in {self.__class__.__name__}")
        tree = ET.parse(self.xml_file_path)
        self.urls = tree.getroot()
        LOGGER.info(f"end of set_up method in {self.__class__.__name__}")

    def execute(self):
        LOGGER.info(f"start execute method in {self.__class__.__name__}")

        @performance(process_name=BROWSER_PROCESS_NAME)
        @setup(
            is_browsing=True,
            process_name=BROWSER_PROCESS_NAME,
            is_execution_part=True,
        )
        def run(driver):
            """Run the use case for four browsers (Amazon, Google, Bing,
            and YouTube).

            This function will perform the following:
            1. Open the Amazon website and wait until the reload sign
            in the tab disappears.
            2. Open Google in a new tab and wait until the reload sign
            in the tab disappears.
            3. Open Bing in a new tab and wait until the reload sign
            in the tab disappears.
            4. Open YouTube in a new tab and wait until the reload sign
            in the tab disappears.

            This function is decorated with the @setup decorator,
            configuring the driver for the browser processes.

            Args:
                driver: The WebDriver instance used to interact with
                    the browser.
            """
            driver.maximize_window()
            last_url_index = len(self.urls) - 1
            for index, url in enumerate(self.urls):
                load_website_by_url(driver, url.text)
                if index != last_url_index:
                    # open a new tab if not last website to visit
                    open_new_blank_tab(driver)

        run()
        LOGGER.info(f"end of execute method in {self.__class__.__name__}")


if __name__ == "__main__":
    BrowseFourWebsiteTest().run_test()
