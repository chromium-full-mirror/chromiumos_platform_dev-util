# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable= W1203
"""Script use to visit the websites based on driver type."""

from selenium.webdriver.support.wait import WebDriverWait

from benchmarking.benchmarking_exception import BenchmarkingException
from benchmarking.time_benchmark import time_measure
from common.constants import PageLoadConfig, LoggingConfig

LOGGER = LoggingConfig.get_logger()


@time_measure()
def open_new_blank_tab(driver):
    """Function to open a new tab and switch to it.

    Args:
        driver: The WebDriver instance used to interact with the browser.
    """
    LOGGER.info("start open_new_blank_tab method.")
    driver.execute_script("window.open('');")
    driver.switch_to.window(driver.window_handles[-1])
    LOGGER.info("end of open_new_blank_tab method.")


@time_measure(1)
def load_website_by_url(driver, url):
    """Function used to load any website by its url
    and wait until the website loading finish.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        url: The URL for website needs to be loaded.
    """
    LOGGER.info(f"start load_website_by_url method with url = {url}.")
    if url and driver is not None:
        driver.get(url)
        WebDriverWait(driver, PageLoadConfig.MAX_PAGE_LOAD_TIME).until(
            check_complete_page_loading_status()
        )
        LOGGER.info("end of load_website_by_url method.")
    else:
        LOGGER.info("load_website_by_url failed with BenchmarkingException.")
        raise BenchmarkingException("passed parameters should not be None.")


def check_complete_page_loading_status():
    """Helper function to check if the page load is complete."""
    LOGGER.info("executing check_complete_page_loading_status method.")
    return (
        lambda driver: driver.execute_script("return document.readyState")
        == "complete"
    )
