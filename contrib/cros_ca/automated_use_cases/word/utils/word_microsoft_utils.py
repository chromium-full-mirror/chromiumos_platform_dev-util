# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable= W1203
"""Microsoft Word web use case utils."""
from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.support import expected_conditions as EC

from automated_use_cases.common_utility.microsoft_documents_utils import (
    wait_saving_status,
)
from benchmarking.time_benchmark import time_measure
from common.constants import LoggingConfig

LOGGER = LoggingConfig.get_logger()


@time_measure()
def navigate_to_last_page(driver, wait):
    """Function to navigate to last page.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
    """
    LOGGER.info("start navigate_to_last_page method.")
    wait.until(
        EC.visibility_of_element_located(
            (By.ID, "PaginatedContentContainerDisplayWrapper")
        )
    )
    pages_container = driver.find_element_by_id(
        "PaginatedContentContainerDisplayWrapper"
    )
    normal_text = pages_container.find_elements_by_class_name("NormalTextRun")[
        0
    ]
    normal_text.click()
    normal_text.send_keys(Keys.CONTROL, Keys.END)
    LOGGER.info("end of navigate_to_last_page method.")


@time_measure()
def add_comment_to_last_page(driver, wait):
    """Function to add comments in document.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
    """
    LOGGER.info("start add_comment_to_last_page method.")
    wait.until(EC.visibility_of_element_located((By.ID, "Insert")))
    driver.find_element_by_id("Insert").click()
    insert_button_disabled_xpath = (
        "//button[contains(@id,"
        "'NewComment') and contains"
        "(@aria-disabled, 'true')]"
    )
    wait.until_not(
        EC.visibility_of_element_located(
            (By.XPATH, insert_button_disabled_xpath)
        )
    )
    wait.until(EC.element_to_be_clickable((By.ID, "NewComment")))
    driver.find_element_by_id("NewComment").click()

    last_card_editor_xpath = (
        "//div[starts-with(@id,'cardEditor_') "
        "and contains(@class, "
        "'ql-editor ql-blank') and "
        "(contains(@data-placeholder, "
        "'@mention or comment') or contains"
        "(@data-placeholder, "
        "'Start a conversation'))]"
    )
    wait.until(
        EC.visibility_of_element_located((By.XPATH, last_card_editor_xpath))
    )
    last_card_editor = driver.find_element_by_xpath(last_card_editor_xpath)
    last_card_editor.click()
    last_card_editor.send_keys("Hi.")
    send_button_xpath = (
        "//button[starts-with(@id,'sendReplyButton')"
        " and contains(@aria-label, 'Post comment "
        "(Ctrl + Enter)')]"
    )
    wait.until(EC.visibility_of_element_located((By.XPATH, send_button_xpath)))
    driver.find_element_by_xpath(send_button_xpath).click()
    wait_saving_status(wait)
    LOGGER.info("end of add_comment_to_last_page method.")
