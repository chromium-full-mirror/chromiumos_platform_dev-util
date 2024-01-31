# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Login to an account before using it."""

from selenium.webdriver.common.by import By
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.wait import WebDriverWait

from automated_use_cases.common_utility.get_website_util import (
    check_complete_page_loading_status,
)
from common.constants import (
    TestingAccountLicense,
    PageLoadConfig,
    LoggingConfig,
)

LOGGER = LoggingConfig.get_logger()


def login_to_google_account(driver):
    """Login to Google testing account.

    Args:
        driver: The WebDriver instance used to interact with the browser.
    """
    LOGGER.info("start login_to_google_account method.")
    WebDriverWait(driver, PageLoadConfig.MAX_PAGE_LOAD_TIME).until(
        check_complete_page_loading_status()
    )
    wait = WebDriverWait(driver, PageLoadConfig.WAIT_TIMEOUT)
    email_box = wait.until(
        EC.visibility_of_element_located((By.ID, "identifierId"))
    )
    email_box.send_keys(TestingAccountLicense.LOGIN_ACCOUNT_EMAIL)
    wait.until(
        EC.visibility_of_element_located((By.ID, "identifierNext"))
    ).click()
    password_box_xpath = (
        "//input[contains(@autocomplete,'current-password') "
        "and contains(@aria-label, 'Enter your password') "
        "and contains(@name,'Passwd') "
        "and contains(@type,'password')]"
    )
    password_box = wait.until(
        EC.visibility_of_element_located((By.XPATH, password_box_xpath))
    )
    password_box.send_keys(TestingAccountLicense.LOGIN_ACCOUNT_PASSWORD)
    url_before_login = driver.current_url
    wait.until(
        EC.visibility_of_element_located((By.ID, "passwordNext"))
    ).click()
    wait.until(EC.url_changes(url_before_login))
    if (
        "passkeyenrollment" in driver.current_url
        or "web/chip" in driver.current_url
    ):
        WebDriverWait(driver, PageLoadConfig.MAX_PAGE_LOAD_TIME).until(
            check_complete_page_loading_status()
        )
        wait.until(
            EC.element_to_be_clickable((By.XPATH, "//span[text()='Not now']"))
        ).click()
    LOGGER.info("end of login_to_google_account method.")


def login_to_microsoft_account(driver):
    """Login to Microsoft testing account.

    Args:
        driver: The WebDriver instance used to interact with the browser.
    """
    LOGGER.info("start login_to_microsoft_account method.")
    WebDriverWait(driver, PageLoadConfig.MAX_PAGE_LOAD_TIME).until(
        check_complete_page_loading_status()
    )
    wait = WebDriverWait(driver, PageLoadConfig.WAIT_TIMEOUT)
    email_box = wait.until(
        EC.visibility_of_element_located((By.NAME, "loginfmt"))
    )
    email_box.send_keys(TestingAccountLicense.LOGIN_ACCOUNT_EMAIL)
    wait.until(EC.visibility_of_element_located((By.ID, "idSIButton9"))).click()
    password_box = wait.until(
        EC.visibility_of_element_located((By.NAME, "passwd"))
    )
    password_box.send_keys(TestingAccountLicense.LOGIN_ACCOUNT_PASSWORD)
    wait.until(EC.visibility_of_element_located((By.ID, "idSIButton9"))).click()
    wait.until(
        EC.visibility_of_element_located((By.ID, "acceptButton"))
    ).click()
    LOGGER.info("end of login_to_microsoft_account method.")
