# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Google Slides Web use case."""
from selenium.webdriver import ActionChains
from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.support import expected_conditions as EC

from benchmarking.time_benchmark import time_measure
from common.constants import LoggingConfig

LOGGER = LoggingConfig.get_logger()


@time_measure()
def navigate_to_last_slide(driver, wait):
    """Function to navigate to last slide.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
    """
    LOGGER.info("start navigate_to_last_slide method.")
    wait.until(
        EC.visibility_of_element_located(
            (By.CLASS_NAME, "punch-filmstrip-thumbnail-background")
        )
    )
    first_slide = driver.find_elements_by_class_name(
        "punch-filmstrip-thumbnail-background"
    )[0]
    actions = ActionChains(driver)
    actions.move_to_element(first_slide).click(first_slide).send_keys(
        Keys.END
    ).perform()
    LOGGER.info("end of navigate_to_last_slide method.")
