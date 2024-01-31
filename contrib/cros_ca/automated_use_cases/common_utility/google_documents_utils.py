# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable= W1203
"""Script contains some functions to handle common cases in
Google documents.
"""
import time

from selenium.common.exceptions import (
    NoSuchElementException,
    TimeoutException,
)
from selenium.webdriver import ActionChains
from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.wait import WebDriverWait

from automated_use_cases.common_utility.doc_editing_utils import (
    close_file_picker_dialog,
)
from automated_use_cases.common_utility.get_website_util import (
    check_complete_page_loading_status,
)
from benchmarking.benchmarking_exception import BenchmarkingException
from benchmarking.time_benchmark import time_measure
from common.constants import PageLoadConfig, LoggingConfig

LOGGER = LoggingConfig.get_logger()


def wait_visibility_of_document_docs_bars(driver, wait):
    """Function to wait visibility of document and close doc material
    dialog when it is available.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
    """
    LOGGER.info("start wait_visibility_of_document_docs_bars method.")
    wait.until(EC.visibility_of_element_located((By.ID, "docs-bars")))
    try:
        driver.find_element_by_xpath(
            "//span[@class='docs-material-gm-dialog-title-close']"
        ).click()
        LOGGER.info("docs material dialog closed.")
    except NoSuchElementException:
        pass
    try:
        driver.find_element_by_xpath("//span[text()='Got it']").click()
        LOGGER.info("docs material dialog closed.")
    except NoSuchElementException:
        pass
    LOGGER.info("end of wait_visibility_of_document_docs_bars method.")


def check_view_mode_if_open_needed(driver):
    """Function to check if view mode needs to be open.

    Args:
        driver: The WebDriver instance used to interact with the browser.
    """
    LOGGER.info("start check_view_mode_if_open_needed method.")
    view_mode = driver.find_element_by_id("viewModeButton")
    if "Show the menus" in view_mode.get_attribute("aria-label"):
        view_mode.click()
    LOGGER.info("end of check_view_mode_if_open_needed method.")


def wait_saving_status(wait):
    """Function to wait saving status of documents to be saved.

    Args:
        wait: Explicit wait used while checking the expected conditions
            of the elements.
    """
    LOGGER.info("start wait_saving_status method.")
    saving_status_path = (
        "//div[contains(@id,"
        "'docs-save-indicator-badge') and contains"
        "(@aria-label,'Document status: Saving….')]"
    )
    wait.until_not(
        EC.visibility_of_element_located((By.XPATH, saving_status_path))
    )
    LOGGER.info("end of wait_saving_status method.")


@time_measure()
def add_comment(driver, wait, index_of_comment_button: int):
    """Function to add comments in document.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
        index_of_comment_button: Index of comment button from
            select menu appears when its appear.
    """
    LOGGER.info(
        "start add_comment method with "
        f"index_of_comment_button = {index_of_comment_button}."
    )
    wait.until(EC.visibility_of_element_located((By.ID, "docs-insert-menu")))
    driver.find_element_by_id("docs-insert-menu").click()
    # select add comment from insert menu
    insert_menu_xpath = (
        "//div[contains(@class,'goog-menu "
        "goog-menu-vertical docs-material ia-menu "
        "ia-primary-menu ia-has-icon "
        "apps-menu-hide-mnemonics') and "
        "contains(@style, 'visibility: visible;')]"
    )
    wait.until(EC.visibility_of_element_located((By.XPATH, insert_menu_xpath)))
    insert_menu = driver.find_element_by_xpath(insert_menu_xpath)
    add_comments_button = insert_menu.find_elements_by_class_name(
        "goog-menuitem-content"
    )[index_of_comment_button]
    add_comments_button.click()
    comment_input_xpath = (
        "//div[contains(@class,'editable "
        "awr-inlineAssistMonitorContainer')"
        " and contains(@placeholder, "
        "'Comment or add others with @')]"
    )
    wait.until(
        EC.visibility_of_element_located((By.XPATH, comment_input_xpath))
    )
    comment_input = driver.find_element_by_xpath(comment_input_xpath)
    comment_input.click()
    comment_input.send_keys("Hi.")
    post_comment_button_xpath = (
        "//div[contains(@class,"
        "'goog-inline-block jfk-button "
        "jfk-button-action docos-input-post "
        "docos-input-buttons-post') and "
        "contains(@aria-disabled, 'false')]"
    )
    wait.until(
        EC.visibility_of_element_located((By.XPATH, post_comment_button_xpath))
    )
    post_comment_button = driver.find_element_by_xpath(
        post_comment_button_xpath
    )
    post_comment_button.click()
    wait_saving_status(wait)
    LOGGER.info("end of add_comment method.")


def move_to_main_container(driver, wait):
    """Helper function for moving to main container.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
    """
    main_container_xpath = "//div[@role='main']"
    main_container = wait.until(
        EC.visibility_of_element_located((By.XPATH, main_container_xpath))
    )
    action = ActionChains(driver)
    action.move_to_element(main_container).click(main_container).perform()


def close_first_usage_dialogs(driver):
    """Function used to close some dialogs window that's appear for first
    usage of Google account with Google Drive.

    Args:
        driver: The WebDriver instance used to interact with the browser.
    """
    LOGGER.info("start close_first_usage_dialogs method.")
    try:
        WebDriverWait(driver, 12).until(
            EC.element_to_be_clickable(
                (By.XPATH, "//button[@aria-label='Close']")
            )
        ).click()
    except TimeoutException:
        pass
    try:
        hide_details_button_xpath = (
            "//div[(@role='button') and (@aria-label='Hide Details')]"
        )
        WebDriverWait(driver, 2).until(
            EC.element_to_be_clickable((By.XPATH, hide_details_button_xpath))
        ).click()
    except TimeoutException:
        pass
    LOGGER.info("end of close_first_usage_dialogs method.")


def upload_document_to_google_drive(driver, wait, file_path, new_doc_name):
    """Function used for uploading office document to Google Drive.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
        file_path: The absolute path of file that needs to be uploaded
            from local storge.
        new_doc_name: The new name of document that be represented by uuid4
            with file type extension.

    Returns:
        the url of uploaded document.
    """
    LOGGER.info(
        "start upload_document_to_google_drive method with "
        f"file_path = {file_path} and new_doc_name = {new_doc_name}"
    )
    close_first_usage_dialogs(driver)
    wait.until(
        EC.visibility_of_element_located((By.XPATH, "//span[text()='New']"))
    ).click()
    close_file_picker_dialog(driver)
    wait.until(
        EC.visibility_of_element_located(
            (By.XPATH, "//div[(@role='menuitem') and (@aria-posinset='5')]")
        )
    )
    wait.until(
        EC.element_to_be_clickable((By.XPATH, "//div[text()='File upload']"))
    ).click()
    upload_options = wait.until(
        EC.presence_of_element_located((By.XPATH, "//input[@type='file']"))
    )
    upload_options.send_keys(file_path)
    wait.until(
        EC.visibility_of_element_located(
            (By.XPATH, "//span[text()='1 upload complete']")
        )
    )
    LOGGER.info(f"File with name = {new_doc_name}, uploaded successfully.")
    window_before = driver.window_handles
    move_to_main_container(driver, wait)
    last_height = 0
    document_xpath = f"//div[(text() = '{new_doc_name}')]"
    while True:
        document_list = driver.find_elements_by_xpath(document_xpath)
        if len(document_list) > 0:
            document = document_list[0]
            actions = ActionChains(driver)
            actions.move_to_element(document).double_click(document).perform()
            break
        action = ActionChains(driver)
        action.send_keys(Keys.PAGE_DOWN).perform()
        time.sleep(2)
        new_height = driver.find_element_by_xpath(
            "//c-wiz[@role='grid']"
        ).location["y"]
        if new_height == last_height:
            LOGGER.info(
                "end of upload_document_to_google_drive method with "
                "no document found."
            )
            raise BenchmarkingException(
                f"Cannot find uploaded file = {new_doc_name} in files list"
            )
        last_height = new_height
    wait.until(EC.new_window_is_opened(window_before))
    driver.switch_to.window(driver.window_handles[-1])
    WebDriverWait(driver, PageLoadConfig.MAX_PAGE_LOAD_TIME).until(
        check_complete_page_loading_status()
    )
    wait_visibility_of_document_docs_bars(driver, wait)
    time.sleep(10)
    LOGGER.info("end of upload_document_to_google_drive method.")
    return driver.current_url


def delete_created_document_from_google_drive(driver, wait, new_doc_name):
    """Function used to delete the document that was created in setup
    operation from Google Drive.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
        new_doc_name: Name of new document was copied from original
            document, and it's equal random uuid.
    """
    LOGGER.info(
        "start delete_created_document_from_google_drive method with "
        f"new_doc_name = {new_doc_name}."
    )
    close_first_usage_dialogs(driver)
    move_to_main_container(driver, wait)
    last_height = 0
    doc_to_delete_xpath = f"//div[(text() = '{new_doc_name}')]"
    move_to_trash_button_xpath = (
        "//div[(@role='button') and (@aria-label='Move to trash')]"
    )
    while True:
        doc_to_delete_list = driver.find_elements_by_xpath(doc_to_delete_xpath)
        if len(doc_to_delete_list) > 0:
            doc_to_delete = doc_to_delete_list[0]
            action = ActionChains(driver)
            action.move_to_element(doc_to_delete).click(doc_to_delete).perform()
            wait.until(
                EC.element_to_be_clickable(
                    (By.XPATH, move_to_trash_button_xpath)
                )
            ).click()
            break
        action = ActionChains(driver)
        action.send_keys(Keys.PAGE_DOWN).perform()
        time.sleep(2)
        new_height = driver.find_element_by_xpath(
            "//c-wiz[@role='grid']"
        ).location["y"]
        if new_height == last_height:
            LOGGER.info(
                "end of delete_created_document_from_google_drive method with "
                "no document to delete found."
            )
            return
        last_height = new_height
    wait.until(
        EC.invisibility_of_element_located((By.XPATH, doc_to_delete_xpath))
    )
    driver.find_element_by_xpath(
        "//span[contains(@aria-label, 'Trashed items')]"
    ).click()
    empty_trash_button = wait.until(
        EC.visibility_of_element_located(
            (By.XPATH, "//button[contains(@aria-label, 'Empty trash')]")
        )
    )
    driver.execute_script("arguments[0].click();", empty_trash_button)
    delete_forever_button = wait.until(
        EC.visibility_of_element_located(
            (By.XPATH, "//span[(text() = 'Delete forever')]")
        )
    )
    driver.execute_script("arguments[0].click();", delete_forever_button)
    time.sleep(10)
    LOGGER.info("end of delete_created_document_from_google_drive method.")
