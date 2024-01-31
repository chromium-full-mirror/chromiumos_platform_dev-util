#!/usr/bin/env python3
# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable= E1120, W1203
"""SpeedTest use case."""
from selenium.common.exceptions import NoSuchElementException
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
from common.constants import LoggingConfig, PageLoadConfig

LOGGER = LoggingConfig.get_logger()
RESULTS_LOGGER = LoggingConfig.get_logger(False)
BROWSER_PROCESS_NAME = get_browser_process_name()


def handle_cookies_popup(driver):
    """Function to handle cookies popup by rejection.

    Args:
        driver: The WebDriver instance used to interact with the browser.
    """
    LOGGER.info("start handle_cookies_popup method.")
    try:
        driver.find_element_by_xpath("//button[text()='Reject All']").click()
    except NoSuchElementException:
        pass
    LOGGER.info("end of handle_cookies_popup method.")


def start_speed_test(driver, wait):
    """Function to start speed test and wait the result.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
    """
    LOGGER.info("start executing start_speed_test method.")
    wait.until(
        EC.element_to_be_clickable((By.XPATH, "//span[text()='Go']"))
    ).click()
    wait.until(
        EC.visibility_of_element_located(
            (
                By.XPATH,
                "//div[contains(@class, 'result-container-speed-active')]",
            )
        )
    )
    download_speed = driver.find_element_by_xpath(
        "//span[contains(@class, 'download-speed')]"
    ).get_attribute("innerHTML")
    upload_speed = driver.find_element_by_xpath(
        "//span[contains(@class, 'upload-speed')]"
    ).get_attribute("innerHTML")
    ping_speed = driver.find_element_by_xpath(
        "//span[contains(@class, 'ping-speed')]"
    ).get_attribute("innerHTML")
    RESULTS_LOGGER.info(f"download_speed = {download_speed}")
    RESULTS_LOGGER.info(f"upload_speed = {upload_speed}")
    RESULTS_LOGGER.info(f"ping_speed = {ping_speed}")
    LOGGER.info("end of executing start_speed_test method.")


class LoadSpeedTestTest(BenchmarkingTest):
    """Test class for SpeedTest use case."""

    def __init__(self):
        self.speed_test_url = "https://www.speedtest.net/"

    def execute(self):
        LOGGER.info(f"start execute method in {self.__class__.__name__}")

        @performance(process_name=BROWSER_PROCESS_NAME)
        @setup(
            is_browsing=True,
            process_name=BROWSER_PROCESS_NAME,
            is_execution_part=True,
        )
        def run(driver):
            """Run the use case for SpeedTest.

            This function will perform the following:
            1. Open and load speed test website.
            2. Wait until the login page finish reloading.
            3. Handle Cookies Popup.
            4. Start speed test and wait the result.
            5. Close the browser.

            This function is decorated with the @setup decorator,
            configuring the driver for the browser processes.

            Args:
                driver: The WebDriver instance used to interact with
                    the browser.
            """
            wait = WebDriverWait(driver, PageLoadConfig.WAIT_TIMEOUT)
            driver.maximize_window()
            load_website_by_url(driver, self.speed_test_url)
            handle_cookies_popup(driver)
            start_speed_test(driver, wait)

        run()
        LOGGER.info(f"end of execute method in {self.__class__.__name__}")


if __name__ == "__main__":
    LoadSpeedTestTest().run_test()
