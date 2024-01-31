#!/usr/bin/env python3
# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable= E1120, W1203
"""Load PDF document from local storge use case."""

from msedge.selenium_tools import Edge
from selenium import webdriver as chrome_webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.wait import WebDriverWait

from automated_use_cases.common_utility.get_website_util import (
    load_website_by_url,
)
from benchmarking.benchmarking_test import (
    BenchmarkingTest,
)
from benchmarking.performance_decorator import performance
from benchmarking.platform_benchmarking import setup, get_browser_process_name
from benchmarking.time_benchmark import time_measure
from common.constants import UseCasesURLS, LoggingConfig

LOGGER = LoggingConfig.get_logger()

BROWSER_PROCESS_NAME = get_browser_process_name()


@time_measure()
def print_and_wait_until_loading_preview_finish(driver):
    """Function to press on print and wait until loading preview finish.

    Args:
        driver: The WebDriver instance used to interact with the browser.
    """
    LOGGER.info("start print_and_wait_until_loading_preview_finish method.")
    wait = WebDriverWait(driver, 20)
    window_before = driver.window_handles
    driver.execute_script("window.print();")
    wait.until(EC.new_window_is_opened(window_before))
    driver.switch_to.window(driver.window_handles[-1])
    # wait until pages loaded in print mode
    if isinstance(driver, chrome_webdriver.Chrome):
        wait.until_not(
            EC.text_to_be_present_in_element(
                (By.XPATH, "/html/body"), "Loading preview..."
            )
        )
    elif isinstance(driver, Edge):
        wait.until_not(
            EC.visibility_of_element_located(
                (By.XPATH, "//div[contains(@role,'progressbar')]")
            )
        )
    LOGGER.info("end of print_and_wait_until_loading_preview_finish method.")


class PDFWebTest(BenchmarkingTest):
    """Test class for Load PDF document from local storge use case."""

    def __init__(self):
        self.doc_url = UseCasesURLS.PDF_WEB_URL_PATH

    def execute(self):
        LOGGER.info(f"start execute method in {self.__class__.__name__}")

        @performance(process_name=BROWSER_PROCESS_NAME)
        @setup(
            is_browsing=True,
            process_name=BROWSER_PROCESS_NAME,
            is_execution_part=True,
        )
        def run(driver):
            """Run the Web PDF use case.

            This function will perform the following:
            1. Open browser and load PDF file from local storge.
            2. Click on the printer button.
            3. Wait until the loading preview step finish.
            4. Close the browser.

            This function is decorated with the @setup decorator,
            configuring the driver for the browser processes.

            Args:
                driver: The WebDriver instance used to interact with
                    the browser.
            """
            driver.maximize_window()
            load_website_by_url(driver, self.doc_url)
            print_and_wait_until_loading_preview_finish(driver)

        run()
        LOGGER.info(f"end of execute method in {self.__class__.__name__}")


if __name__ == "__main__":
    PDFWebTest().run_test()
