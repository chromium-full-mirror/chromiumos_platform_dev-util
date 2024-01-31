# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Microsoft PowerPoint web use case utils."""
from selenium.webdriver import ActionChains
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
def navigate_to_last_slide(driver, wait):
    """function to navigate to last slide.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
    """
    LOGGER.info("start navigate_to_last_slide method.")
    wait.until(EC.visibility_of_element_located((By.ID, "WACViewPanel")))
    first_slide_path = (
        "//div[contains(@id,'grid-content-view-') and "
        "starts-with(@aria-posinset,'1')]"
    )
    wait.until(EC.visibility_of_element_located((By.XPATH, first_slide_path)))
    first_slide = driver.find_element_by_xpath(first_slide_path)
    actions = ActionChains(driver)
    actions.move_to_element(first_slide).click(first_slide).send_keys(
        Keys.END
    ).perform()
    LOGGER.info("ending of navigate_to_last_slide method.")


@time_measure()
def add_comment_to_last_slide(driver, wait):
    """function to add comments in document.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
    """
    LOGGER.info("start add_comment_to_last_slide method.")
    wait.until(EC.visibility_of_element_located((By.ID, "Insert")))
    driver.find_element_by_id("Insert").click()
    insert_button_disabled_xpath = (
        "//button[contains(@id,"
        "'InsertComment') and contains"
        "(@aria-disabled, 'true')]"
    )
    wait.until_not(
        EC.visibility_of_element_located(
            (By.XPATH, insert_button_disabled_xpath)
        )
    )
    wait.until(EC.element_to_be_clickable((By.ID, "InsertComment")))
    driver.find_element_by_id("InsertComment").click()
    wait.until(
        EC.visibility_of_element_located(
            (By.ID, "EditingCommentsPanelFocus-panel")
        )
    )
    wait.until(EC.visibility_of_element_located((By.ID, "NewCommentButton")))
    driver.find_element_by_id("NewCommentButton").click()
    wait.until(
        EC.visibility_of_element_located((By.ID, "CommentsListScrollView"))
    )
    comment_input_section_xpath = (
        "//div[contains(@class,'ql-editor "
        "ql-blank') and (contains"
        "(@aria-label, '@mention or "
        "comment') or contains(@aria-label,"
        " 'Start a conversation'))]"
    )
    wait.until(
        EC.visibility_of_element_located(
            (By.XPATH, comment_input_section_xpath)
        )
    )
    comment_input_section = driver.find_element_by_xpath(
        comment_input_section_xpath
    )
    comment_input_section.click()
    comment_input_section.send_keys("Hi.")
    send_replay_button_xpath = (
        "//button[starts-with(@id,'sendReplyButton') "
        "and contains(@aria-label, "
        "'Post comment (Ctrl + Enter)')]"
    )
    wait.until(
        EC.visibility_of_element_located((By.XPATH, send_replay_button_xpath))
    )
    driver.find_element_by_xpath(send_replay_button_xpath).click()
    wait_saving_status(wait)
    LOGGER.info("end of add_comment_to_last_slide method.")
