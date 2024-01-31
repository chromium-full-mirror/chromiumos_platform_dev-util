# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable= W1203
"""Google Sheet web use case utils."""

import time

from selenium.webdriver import ActionChains
from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.support import expected_conditions as EC

from automated_use_cases.common_utility import doc_editing_utils
from automated_use_cases.common_utility.google_documents_utils import (
    wait_saving_status,
)
from benchmarking.benchmarking_exception import BenchmarkingException
from benchmarking.time_benchmark import time_measure
from common.constants import LoggingConfig

LOGGER = LoggingConfig.get_logger()


def get_columns_menu(driver):
    """Get the columns menu elements from the Google Sheet page.

    Args:
       driver: The WebDriver instance used to interact with the browser.

    Returns:
       list: A list of column menu elements.

    Raises:
        BenchmarkingException: in case no columns found.
    """
    LOGGER.info("starts get_columns_menu method.")
    columns_menu = driver.find_elements_by_xpath(
        "//div[contains(@class,'waffle-dbsource-column-filtered-menu') and"
        " contains(@style, 'visibility: visible;')]"
    )
    for i in columns_menu:
        if "display: none;" not in i.get_attribute("style"):
            LOGGER.info("end of get_columns_menu method.")
            return i.find_elements_by_class_name(
                "waffle-filterable-by-text-contains-menu-item-label"
            )
    LOGGER.info("get_columns_menu failed with BenchmarkingException.")
    raise BenchmarkingException(
        "Error in get_columns_menu for the excel, no columns found"
    )


def select_and_start_from_first_page(driver, wait):
    """Helper function to select an element and start from the first working
    sheet.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: The WebDriverWait instance used for waiting on conditions.
    """
    LOGGER.info("start select_and_start_from_first_page method.")
    wait.until_not(
        EC.visibility_of_element_located(
            (By.ID, "waffle-loading-progress-bar-overlay")
        )
    )
    wait.until(
        EC.visibility_of_element_located(
            (
                By.XPATH,
                "//div[contains(@class,'docs-sheet-container-bar "
                "goog-toolbar goog-inline-block')]",
            )
        )
    )
    working_sheet_xpath = (
        "//span[contains"
        "(@class,'docs-sheet-tab-name')"
        " and (text()='Sheet1')]"
    )
    wait.until(
        EC.visibility_of_element_located((By.XPATH, working_sheet_xpath))
    )
    first_page = driver.find_element_by_xpath(working_sheet_xpath)
    first_page.click()
    LOGGER.info("ending of select_and_start_from_first_page method.")


def select_sheet_content(driver, wait):
    """Helper function to select all the elements in the Excel sheet (Ctrl+a).

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: The WebDriverWait instance used for waiting on conditions.
    """
    LOGGER.info("start select_sheet_content method.")
    grid_wrapper = "grid-scrollable-wrapper"
    wait.until(EC.visibility_of_element_located((By.CLASS_NAME, grid_wrapper)))
    sheet_content = driver.find_element_by_class_name(grid_wrapper)
    doc_editing_utils.select_sheet_content(driver, sheet_content)
    LOGGER.info("end of select_sheet_content method.")


def get_insert_menu_xpath():
    """Get the XPath of the insert choice in the Excel taskbar.

    Returns:
        String represents insert menu xpath.
    """
    LOGGER.info("executing get_insert_menu_xpath method.")
    return (
        "//div[contains(@class,'docs-material ia-menu"
        " ia-primary-menu ia-has-icon "
        "apps-menu-hide-mnemonics') and "
        "contains(@style, 'visibility: visible;')]"
    )


def delete_stacked_bar_chart(driver, wait):
    """Helper function to delete the stacked bar chart that we create it before.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: The WebDriverWait instance used for waiting on conditions.
    """
    LOGGER.info("start delete_stacked_bar_chart method.")
    stacked_chart_path = (
        "//div[contains(@class,"
        "'docs-charts-editor-charttype-option') and "
        "contains(@data-tooltip,"
        " 'Stacked bar chart')]"
    )
    wait.until(EC.visibility_of_element_located((By.XPATH, stacked_chart_path)))
    driver.find_element_by_xpath(stacked_chart_path).click()
    actions = ActionChains(driver)
    actions.send_keys(Keys.DELETE)
    actions.perform()
    LOGGER.info("end of delete_stacked_bar_chart method.")


@time_measure()
def create_stacked_bar_chart(driver, wait):
    """Helper function to Create a stacked bar from the existing values in
     the Excel sheet.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: The WebDriverWait instance used for waiting on conditions.
    """
    LOGGER.info("start create_stacked_bar_chart method.")
    wait.until(EC.visibility_of_element_located((By.ID, "docs-insert-menu")))
    driver.find_element_by_id("docs-insert-menu").click()

    # Get insert choice xpath
    insert_menu_xpath = get_insert_menu_xpath()

    wait.until(EC.visibility_of_element_located((By.XPATH, insert_menu_xpath)))
    insert_menu = driver.find_element_by_xpath(insert_menu_xpath)
    chart_button = insert_menu.find_elements_by_class_name(
        "goog-menuitem-content"
    )[4]
    chart_button.click()
    wait.until(
        EC.visibility_of_element_located(
            (
                By.CLASS_NAME,
                "waffle-borderless-embedded-object-container",
            )
        )
    )
    wait.until(
        EC.visibility_of_element_located(
            (By.CLASS_NAME, "docs-charts-editor-charttype-select-name")
        )
    )
    driver.find_element_by_class_name(
        "docs-charts-editor-charttype-select-name"
    ).click()
    LOGGER.info("end of create_stacked_bar_chart method.")


@time_measure()
def create_pivot_table_in_new_sheet(driver, wait):
    """Helper function to create a pivot table for the Excel.

     This function will create the pivot table and select the required rows
     and values for this pivot table.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: The WebDriverWait instance used for waiting on conditions.
    """
    LOGGER.info("start create_pivot_table_in_new_sheet method.")
    insert_menu_xpath = get_insert_menu_xpath()
    driver.find_element_by_id("docs-insert-menu").click()
    wait.until(EC.visibility_of_element_located((By.XPATH, insert_menu_xpath)))
    insert_menu = driver.find_element_by_xpath(insert_menu_xpath)
    pivot_button = insert_menu.find_elements_by_class_name(
        "goog-menuitem-content"
    )[5]
    pivot_button.click()
    wait.until(
        EC.visibility_of_element_located(
            (By.CLASS_NAME, "jfk-radiobutton-radio")
        )
    )
    driver.find_element_by_class_name("jfk-radiobutton-radio").click()
    driver.find_element_by_xpath(
        "//div[contains(@class,"
        "'docs-material-gm-dialog-call-to-action-button')]"
    ).click()
    wait.until(
        EC.presence_of_element_located(
            (By.CLASS_NAME, "waffle-sidebar-content")
        )
    )
    wait.until(
        EC.visibility_of_element_located((By.ID, "waffle-pivot-sidebar"))
    )

    # start selecting the pivot item
    add_row = driver.find_element_by_id("waffle-pivot-add-field-row")
    add_value = driver.find_element_by_id("waffle-pivot-add-field-value")
    counter = 0
    while counter < 2:
        time.sleep(1)
        add_row.click()
        menus = get_columns_menu(driver)
        menus[0].click()
        counter += 1
    time.sleep(1)
    add_value.click()
    menus = get_columns_menu(driver)
    menus[2].click()
    time.sleep(1)
    wait_saving_status(wait)
    LOGGER.info("end of create_pivot_table_in_new_sheet method.")
