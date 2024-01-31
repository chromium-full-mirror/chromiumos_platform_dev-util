# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable= W1203
"""Google Doc web use case utils."""
from selenium.webdriver import ActionChains
from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.support import expected_conditions as EC

from benchmarking.time_benchmark import time_measure
from common.constants import LoggingConfig

LOGGER = LoggingConfig.get_logger()


def pick_first_page_canvas(driver, wait):
    """Function to select first page canvas.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: Explicit wait used while checking the expected conditions
            of the elements.

    Returns:
        first page canvas
    """
    LOGGER.info("start pick_first_page_canvas method.")
    wait.until(EC.visibility_of_element_located((By.ID, "kix-appview")))
    first_page_path = (
        "//div[contains(@class,'kix-page-paginated canvas-first-page')]"
    )
    wait.until(EC.visibility_of_element_located((By.XPATH, first_page_path)))
    first_page_canvas = driver.find_element_by_xpath(
        first_page_path
    ).find_element_by_class_name("kix-canvas-tile-content")
    LOGGER.info("end of pick_first_page_canvas method.")
    return first_page_canvas


@time_measure()
def navigate_to_last_page(driver, wait):
    """Function to navigate to last page.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
    """
    LOGGER.info("start navigate_to_last_page method.")
    first_page_canvas = pick_first_page_canvas(driver, wait)
    actions = ActionChains(driver)
    (
        actions.move_to_element(first_page_canvas)
        .click(first_page_canvas)
        .key_down(Keys.CONTROL)
        .send_keys(Keys.END)
        .key_up(Keys.CONTROL)
        .perform()
    )
    LOGGER.info("end of navigate_to_last_page method.")
