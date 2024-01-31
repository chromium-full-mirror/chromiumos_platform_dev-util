# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable= W1203
"""Script contains some functions to handle common cases in
Microsoft documents.
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
from common.constants import PageLoadConfig, LoggingConfig

LOGGER = LoggingConfig.get_logger()


def switch_to_native_app_window(driver, wait, no_of_windows_to_be=1):
    """Function used to switch to native application window after open it.

    Args:
        driver: The Windows Desktop App Driver instance used to
            interact with the desktop app.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
        no_of_windows_to_be: Number of window that should be available
            when switch between windows required
    """
    LOGGER.info(
        "start switch_to_native_app_window method"
        f" with no_of_windows_to_be = {no_of_windows_to_be}."
    )
    wait.until(EC.number_of_windows_to_be(no_of_windows_to_be))
    driver.switch_to.window(driver.window_handles[0])
    driver.maximize_window()
    LOGGER.info("end switch_to_native_app_window method.")


def close_native_app_document_without_save(driver):
    """Function used to close native application without save the document
    by select don't save option.

    Args:
        driver: The Windows Desktop App Driver instance used to
            interact with the desktop app.
    """
    LOGGER.info("start close_native_app_document_without_save method.")
    frame_helper_buttons = driver.find_elements_by_class_name(
        "NetUIAppFrameHelper"
    )
    for i in frame_helper_buttons:
        if i.text == "Close":
            i.click()
    all_win_handle = driver.window_handles
    driver.switch_to.window(all_win_handle[0])
    driver.find_element_by_name("Don't Save").click()
    LOGGER.info("end of close_native_app_document_without_save method.")


def wait_saving_status(wait):
    """Function to wait saving status of documents to be saved.

    Args:
        wait: Explicit wait used while checking the expected conditions
            of the elements.
    """
    LOGGER.info("start wait_saving_status method.")
    saving_status_path = (
        "//div[contains(@aria-label,'Saving... Last saved: Just now')]"
    )
    wait.until_not(
        EC.visibility_of_element_located((By.XPATH, saving_status_path))
    )
    LOGGER.info("end of wait_saving_status method.")


def check_ribbon_container_if_open_needed(driver):
    """Function to check if ribbon container needs to be open.

    Args:
        driver: The WebDriver instance used to interact with the browser.
    """
    LOGGER.info("start check_ribbon_container_if_open_needed method.")
    ribbon_container = driver.find_element_by_id("RibbonContainer")
    if not ribbon_container.is_displayed():
        header_region = driver.find_element_by_id("HeaderRegion")
        action = ActionChains(driver)
        action.move_to_element(header_region).perform()
        LOGGER.info("ribbon container open done.")
    LOGGER.info("end of check_ribbon_container_if_open_needed method.")


def switch_to_document_frame(driver, wait, frame_name):
    """Function to switch to document frame.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: Explicit wait used while checking the expected conditions
           of the elements.
        frame_name: Name of frame needs to switch to it.
    """
    LOGGER.info(
        f"start switch_to_document_frame method with frame_name = {frame_name}."
    )
    frame = wait.until(lambda d: d.find_elements_by_id(frame_name))[0]
    driver.switch_to.frame(frame)
    wait.until(EC.visibility_of_element_located((By.ID, "AppBrand")))
    LOGGER.info("end of switch_to_document_frame method.")


def insert_comment_on_native_app_document(driver, wait):
    """Helper function to insert a comment for native application document.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
    """
    LOGGER.info("start insert_comment_on_native_app_document method.")
    driver.find_element_by_accessibility_id("TabInsert").click()
    wait.until(
        EC.visibility_of_element_located((By.CLASS_NAME, "NetUIRibbonButton"))
    )
    driver.find_element_by_accessibility_id("InsertNewComment").click()
    all_win_handle = driver.window_handles
    driver.switch_to.window(all_win_handle[0])
    LOGGER.info("end of insert_comment_on_native_app_document method.")


def write_comment_on_native_app_document(driver, wait):
    """Helper function to write the comment for native application document.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
    """
    LOGGER.info("start write_comment_on_native_app_document method.")
    wait.until(
        EC.visibility_of_element_located((By.NAME, "Start a conversation"))
    )
    comments_input_section = driver.find_element_by_name("Start a conversation")
    comments_input_section.click()
    comments_input_section.send_keys("Hi.")
    wait.until(
        EC.visibility_of_element_located(
            (By.NAME, "Post comment (Ctrl + Enter)")
        )
    ).click()
    LOGGER.info("end of write_comment_on_native_app_document method.")


def upload_document_to_one_drive(
    driver, wait, new_doc_name, frame_name, file_path
):
    """Function used for uploading office document to One Drive.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
        new_doc_name: The new name of document that be represented by uuid4
            with file type extension.
        frame_name: Name of office document frame used to switch to it.
        file_path: The absolute path of file that needs to be uploaded
            from local storge.
    Returns:
        The url of uploaded document.

    Raises:
        BenchmarkingException: in case uploaded file not found in files list.
    """
    LOGGER.info(
        "start upload_document_to_one_drive method with "
        f"new_doc_name = {new_doc_name} and frame_name = {frame_name} "
        f"and file_path = {file_path}"
    )
    close_first_usage_dialogs(driver)
    wait.until(EC.visibility_of_element_located((By.NAME, "My files"))).click()
    add_new_button = wait.until(
        EC.visibility_of_element_located(
            (By.XPATH, "//button[(@title='Add new')]")
        )
    )
    close_file_picker_dialog(driver)
    while True:
        upload_button = driver.find_elements_by_xpath(
            "//button[@title='Files upload']"
        )
        if len(upload_button) > 0:
            driver.execute_script("arguments[0].click();", upload_button[0])
            break
        driver.execute_script("arguments[0].click();", add_new_button)
        time.sleep(2)

    upload_file_input = wait.until(
        EC.presence_of_element_located((By.XPATH, "//input[@type='file']"))
    )
    upload_file_input.send_keys(file_path)
    wait.until(
        EC.visibility_of_element_located(
            (By.XPATH, "//div[starts-with(text(),'Uploaded ')]")
        )
    )
    LOGGER.info(f"File with name = {new_doc_name}, uploaded successfully.")
    no_window_before = driver.window_handles
    move_to_main_app_content(driver, wait)
    last_height = 0
    document_xpath = f"//button[text() = '{new_doc_name}']"
    while True:
        document_list = driver.find_elements_by_xpath(document_xpath)
        if len(document_list) > 0:
            document = document_list[0]
            driver.execute_script(
                "return arguments[0].scrollIntoView(true);", document
            )
            driver.execute_script("arguments[0].click();", document)
            break
        action = ActionChains(driver)
        action.send_keys(Keys.PAGE_DOWN).perform()
        time.sleep(2)
        new_height = driver.find_element_by_class_name(
            "ms-SelectionZone"
        ).location["y"]
        if new_height == last_height:
            LOGGER.info(
                "end of upload_document_to_one_drive method with "
                "no document found."
            )
            raise BenchmarkingException(
                f"Cannot find uploaded file = {new_doc_name} in files list"
            )
        last_height = new_height
    wait.until(EC.new_window_is_opened(no_window_before))
    driver.switch_to.window(driver.window_handles[-1])
    WebDriverWait(driver, PageLoadConfig.MAX_PAGE_LOAD_TIME).until(
        check_complete_page_loading_status()
    )
    time.sleep(10)
    switch_to_document_frame(driver, wait, frame_name)
    LOGGER.info("end of upload_document_to_one_drive method.")
    return driver.current_url


def close_first_usage_dialogs(driver):
    """Function used to close some dialogs window that's appear for first
    usage of Google account with Google Drive.

    Args:
        driver: The WebDriver instance used to interact with the browser.
    """
    LOGGER.info("start close_first_usage_dialogs method.")
    try:
        dialog1_xpath = "//button[@aria-label='Close dialog']"
        WebDriverWait(driver, 10).until(
            EC.element_to_be_clickable((By.XPATH, dialog1_xpath))
        ).click()
    except TimeoutException:
        pass
    try:
        dialog2_xpath = "//button[@aria-label='Close']"
        WebDriverWait(driver, 4).until(
            EC.element_to_be_clickable((By.XPATH, dialog2_xpath))
        ).click()
    except TimeoutException:
        pass
    LOGGER.info("end of close_first_usage_dialogs method.")


def move_to_main_app_content(driver, wait):
    """Helper function for moving to main app content.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
    """
    main_app_content_xpath = (
        "//main[contains(@data-automationid, 'main-app-content')]"
    )
    main_app_content = wait.until(
        EC.visibility_of_element_located((By.XPATH, main_app_content_xpath))
    )
    action = ActionChains(driver)
    action.move_to_element(main_app_content).click(main_app_content).perform()


def delete_created_document_from_one_drive(driver, wait, new_doc_name):
    """Function used to delete the document that was created in setup
    operation from One Drive.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
        new_doc_name: Name of new document was copied from original
            document, and it's equal random uuid.
    """
    LOGGER.info(
        "start delete_created_document_from_one_drive method with "
        f"new_doc_name = {new_doc_name}."
    )
    close_first_usage_dialogs(driver)
    wait.until(EC.visibility_of_element_located((By.NAME, "My files"))).click()
    move_to_main_app_content(driver, wait)
    doc_to_be_deleted_xpath = f"//button[starts-with(text(), '{new_doc_name}')]"
    last_height = 0
    while True:
        doc_to_be_deleted_list = driver.find_elements_by_xpath(
            doc_to_be_deleted_xpath
        )
        if len(doc_to_be_deleted_list) > 0:
            doc_to_be_deleted = doc_to_be_deleted_list[0]
            action = ActionChains(driver)
            action.move_to_element(doc_to_be_deleted).context_click(
                doc_to_be_deleted
            ).perform()
            break
        action = ActionChains(driver)
        action.send_keys(Keys.PAGE_DOWN).perform()
        time.sleep(2)
        new_height = driver.find_element_by_class_name(
            "ms-SelectionZone"
        ).location["y"]
        if new_height == last_height:
            LOGGER.info(
                "end of delete_created_document_from_one_drive method with "
                "no document to delete found."
            )
            return
        last_height = new_height
    wait.until(EC.visibility_of_element_located((By.NAME, "Move to"))).click()
    wait.until(
        EC.frame_to_be_available_and_switch_to_it(
            (
                By.XPATH,
                "//iframe[starts-with(@data-automationid, 'filePickerFrame')]",
            )
        )
    )
    wait.until(EC.element_to_be_clickable((By.NAME, "Move here"))).click()
    driver.switch_to.parent_frame()
    try:
        time.sleep(5)
        driver.find_element_by_name("Move anyway").click()
    except NoSuchElementException:
        pass
    time.sleep(5)
    wait.until(EC.visibility_of_element_located((By.NAME, "Delete"))).click()
    wait.until(
        EC.visibility_of_element_located(
            (By.XPATH, "//button[(@data-automationid= 'confirmbutton')]")
        )
    ).click()
    wait.until(
        EC.invisibility_of_element_located((By.XPATH, doc_to_be_deleted_xpath))
    )
    wait.until(
        EC.visibility_of_element_located((By.NAME, "Recycle bin"))
    ).click()
    wait.until(
        EC.visibility_of_element_located((By.NAME, "Empty recycle bin"))
    ).click()
    confirm_empty_trash = wait.until(
        EC.element_to_be_clickable((By.XPATH, "//span[(text() = 'Yes')]"))
    )
    driver.execute_script("arguments[0].click();", confirm_empty_trash)
    wait.until(
        EC.invisibility_of_element_located((By.NAME, "Empty recycle bin"))
    )
    time.sleep(10)
    LOGGER.info("end of delete_created_document_from_one_drive method.")
