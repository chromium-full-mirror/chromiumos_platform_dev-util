# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Microsoft Word native desktop app use case utils."""

from selenium.webdriver import ActionChains
from selenium.webdriver.common.keys import Keys

from automated_use_cases.common_utility.microsoft_documents_utils import (
    write_comment_on_native_app_document,
    insert_comment_on_native_app_document,
)
from benchmarking.time_benchmark import time_measure
from common.constants import LoggingConfig

LOGGER = LoggingConfig.get_logger()


@time_measure()
def navigate_to_last_page(driver):
    """Function to navigate to last page.

    Args:
        driver: The Windows Desktop App Driver instance used to
            interact with the desktop app.
    """
    LOGGER.info("start navigate_to_last_page method.")
    doc_body = driver.find_elements_by_accessibility_id("Body")
    first_doc_body = doc_body[0]
    actions = ActionChains(driver)
    (
        actions.move_to_element(first_doc_body)
        .click(first_doc_body)
        .key_down(Keys.CONTROL)
        .send_keys(Keys.END)
        .key_up(Keys.CONTROL)
        .perform()
    )
    LOGGER.info("ending of navigate_to_last_page method.")


@time_measure()
def add_comment_to_last_page(driver, wait):
    """Function to add comment to last page.

    Args:
        driver: The Windows Desktop App Driver instance used to
            interact with the desktop app.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
    """
    LOGGER.info("start add_comment_to_last_page method.")
    insert_comment_on_native_app_document(driver, wait)
    write_comment_on_native_app_document(driver, wait)
    LOGGER.info("end of add_comment_to_last_page method.")
