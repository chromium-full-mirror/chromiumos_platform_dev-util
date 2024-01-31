#!/usr/bin/env python3
# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable= E1120, W1203
"""CNN use case."""

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
from benchmarking.time_benchmark import time_measure
from common.constants import LoggingConfig, PageLoadConfig

LOGGER = LoggingConfig.get_logger()

BROWSER_PROCESS_NAME = get_browser_process_name()


def handle_cookies_popup(driver):
    """Function to handle cookies popup by rejection.

    Args:
        driver: The WebDriver instance used to interact with the browser.
    """
    LOGGER.info("start handle_cookies_popup method.")
    try:
        show_purposes = driver.find_element(By.ID, "onetrust-pc-btn-handler")
        show_purposes.click()
        reject_cookies = driver.find_element(
            By.XPATH,
            '//*[@id="onetrust-pc-sdk"]/div/div[3]/div[1]/button[1]',
        )
        reject_cookies.click()
        LOGGER.info("end of handle_cookies_popup method.")
    except NoSuchElementException:
        LOGGER.info("end of handle_cookies_popup method.")


@time_measure()
def watch_cnn_video(driver):
    """Function to select and start watching a video.

    Args:
        driver: The WebDriver instance used to interact with the browser.
    """
    LOGGER.info("start watch_cnn_video method.")
    video = driver.find_element(
        By.XPATH, '//*[@id="pageHeader"]/div/div/div[1]/div[2]/a[1]'
    )
    video.click()
    WebDriverWait(driver, PageLoadConfig.WAIT_TIMEOUT).until(
        EC.presence_of_element_located((By.ID, "pauseIconTitle"))
    )
    LOGGER.info("end of watch_cnn_video method.")


class LoadCNNTest(BenchmarkingTest):
    """Test class for CNN use case."""

    def execute(self):
        LOGGER.info(f"start execute method in {self.__class__.__name__}")

        @performance(process_name=BROWSER_PROCESS_NAME)
        @setup(
            is_browsing=True,
            process_name=BROWSER_PROCESS_NAME,
            is_execution_part=True,
        )
        def run(driver):
            """Run the use case for CNN page.

            This function will perform the following:
            1. Open the CNN edition page and wait until the reload sign
            disappears in the tab.
            2. Open the video page from the CNN page.
            3. Wait until the video start playing.
            4. Close the browser.

            This function is decorated with the @setup decorator,
            configuring the driver for the browser processes.

            Args:
                driver: The WebDriver instance used to interact with
                    the browser.
            """
            driver.maximize_window()
            load_website_by_url(driver, "https://edition.cnn.com")
            handle_cookies_popup(driver)
            watch_cnn_video(driver)

        run()
        LOGGER.info(f"end of execute method in {self.__class__.__name__}")


if __name__ == "__main__":
    LoadCNNTest().run_test()
