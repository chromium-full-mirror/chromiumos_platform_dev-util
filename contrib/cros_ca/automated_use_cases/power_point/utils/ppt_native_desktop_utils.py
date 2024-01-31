# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Microsoft PowerPoint native desktop use case utils."""

from selenium.webdriver import ActionChains
from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.support import expected_conditions as EC

from automated_use_cases.common_utility.microsoft_documents_utils import (
    write_comment_on_native_app_document,
    insert_comment_on_native_app_document,
)
from benchmarking.time_benchmark import time_measure
from common.constants import LoggingConfig

LOGGER = LoggingConfig.get_logger()


@time_measure()
def navigate_to_last_slide(driver):
    """function to navigate to last slide.

    Args:
        driver: The Windows Desktop App Driver instance used to
            interact with the desktop app.
    """
    LOGGER.info("start navigate_to_last_slide method.")
    actions = ActionChains(driver)
    actions.send_keys(Keys.END).perform()
    LOGGER.info("end of navigate_to_last_slide method.")


@time_measure()
def add_comment_to_last_slide(driver, wait):
    """function to add comment.

    Args:
        driver: The Windows Desktop App Driver instance used to
            interact with the desktop app.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
    """
    LOGGER.info("start add_comment_to_last_slide method.")
    insert_comment_on_native_app_document(driver, wait)
    wait.until(EC.visibility_of_element_located((By.NAME, "Comments Pane")))
    wait.until(
        EC.visibility_of_element_located((By.NAME, "New comment"))
    ).click()
    write_comment_on_native_app_document(driver, wait)
    LOGGER.info("end add_comment_to_last_slide method.")
