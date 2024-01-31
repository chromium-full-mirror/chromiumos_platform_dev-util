# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable= W1203
"""Script contains some utils function responsible to handle
some documents use cases.
"""
import os
import subprocess
import time

from selenium.common.exceptions import InvalidArgumentException
from selenium.webdriver import ActionChains
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.support.wait import WebDriverWait

from automated_use_cases.common_utility.account_login import (
    login_to_google_account,
    login_to_microsoft_account,
)
from automated_use_cases.common_utility.get_website_util import (
    check_complete_page_loading_status,
)
from benchmarking.benchmarking_exception import BenchmarkingException
from common.constants import (
    PageLoadConfig,
    DocumentAccountType,
    TestingAccountLicense,
    LoggingConfig,
    DriversAndAppsPaths,
)

LOGGER = LoggingConfig.get_logger()


def open_and_load_account_document_or_drive(
    driver, url_path: str, account_type: DocumentAccountType
):
    """Helper function to open web Excel, Word and PowerPoint documents
    or open account drive using Google or Microsoft account.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        url_path: The URL of web document.
        account_type: The account type to use for login, either
            'google' or 'microsoft'.

    Returns:
        tuple: A tuple contains login time.

    Raises:
        InvalidArgumentException: If the provided app_account is not
            'google' or 'microsoft'.
    """
    LOGGER.info(
        "start open_and_load_account_document_or_drive method with "
        f"url_path = {url_path} and "
        f"account_type = {account_type}."
    )
    if driver is not None and account_type is not None and url_path:
        driver.maximize_window()
        validate_email_and_password_existence()
        driver.get(url_path)
        if account_type == DocumentAccountType.GOOGLE:
            time_before_login = time.time()
            login_to_google_account(driver)
        elif account_type == DocumentAccountType.MICROSOFT:
            time_before_login = time.time()
            login_to_microsoft_account(driver)
        else:
            LOGGER.info(
                "open_and_load_account_document_or_drive failed "
                "with InvalidArgumentException."
            )
            raise InvalidArgumentException(
                "Error! Invalid doc_account_type parameter, "
                "you can only use 'GOOGLE' or 'MICROSOFT'"
            )
        login_time = time.time() - time_before_login
        WebDriverWait(driver, PageLoadConfig.MAX_PAGE_LOAD_TIME).until(
            check_complete_page_loading_status()
        )
        LOGGER.info("end of open_and_load_account_document_or_drive method.")
        return login_time
    LOGGER.info(
        "open_and_load_account_document_or_drive failed "
        "with BenchmarkingException."
    )
    raise BenchmarkingException(
        "passed parameters should not be None or empty."
    )


def validate_email_and_password_existence():
    """Function to validate if email and password used in login process
    exist or not.

    Raises:
        EnvironmentError: in case email or password not exist in
            environment variables.
    """
    LOGGER.info("start validate_email_and_password_existence method.")
    if (
        not TestingAccountLicense.LOGIN_ACCOUNT_EMAIL
        or not TestingAccountLicense.LOGIN_ACCOUNT_PASSWORD
    ):
        LOGGER.info(
            "validate_email_and_password_existence failed "
            "with EnvironmentError."
        )
        raise EnvironmentError(
            "Error! Cannot find the account email or password,"
            " please check your environment variables."
        )
    LOGGER.info("end of validate_email_and_password_existence method.")


def select_sheet_content(driver, sheet_content):
    """Helper function to select sheet content.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        sheet_content: Indicate the sheet content web element
            to interact with it.
    """
    LOGGER.info("start select_sheet_content method.")
    actions = ActionChains(driver)
    (
        actions.move_to_element(sheet_content)
        .click(sheet_content)
        .key_down(Keys.CONTROL)
        .send_keys(Keys.HOME)
        .send_keys("a")
        .key_up(Keys.CONTROL)
        .perform()
    )
    LOGGER.info("end of select_sheet_content method.")


def close_file_picker_dialog(driver):
    """Function used to close file picker dialog when upload files
    functionality required.

    Args:
        driver: The WebDriver instance used to interact with the browser.
    """
    driver.execute_script(
        "HTMLInputElement.prototype.click = function () {"
        + "    if (this.type !== 'file') {"
        + "        HTMLElement.prototype.click.call(this);"
        + "    }"
        + "    else if (!this.parentNode) {"
        + "        this.style.display = 'none';"
        + "        this.ownerDocument.documentElement.appendChild(this);"
        + "        this.addEventListener('change', () => this.remove());"
        + "    }"
        + "}"
    )


def open_win_app_driver():
    """Function to open WinAppDriver application.

    Raises:
        EnvironmentError: in case WinAppDriver path in environment file
            is empty or not exist in the system.
    """
    if not DriversAndAppsPaths.WIN_APP_DRIVER_PATH:
        raise EnvironmentError(
            "Error! WinAppDriver path is empty,"
            " please check your environment variables."
        )
    if not os.path.exists(DriversAndAppsPaths.WIN_APP_DRIVER_PATH):
        raise EnvironmentError(
            "Error! WinAppDriver path not exist in the system,"
            " please check your environment variables."
        )
    os.startfile(DriversAndAppsPaths.WIN_APP_DRIVER_PATH)
    time.sleep(5)


def close_win_app_driver():
    """Function to close WinAppDriver application."""
    time.sleep(5)
    subprocess.call("TASKKILL /F /IM WinAppDriver.exe")
