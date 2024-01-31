# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable= W1203
"""Microsoft Excel web use case utils."""

import time

from selenium.webdriver import ActionChains
from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.support import expected_conditions as EC

from automated_use_cases.common_utility import doc_editing_utils
from automated_use_cases.common_utility.microsoft_documents_utils import (
    wait_saving_status,
)
from benchmarking.time_benchmark import time_measure
from common.constants import LoggingConfig

LOGGER = LoggingConfig.get_logger()


def select_and_start_from_first_page(driver, wait):
    """Helper function to select an element and start from the first working
    sheet.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: The WebDriverWait instance used for waiting on conditions.
    """
    LOGGER.info("start select_and_start_from_first_page method.")
    wait.until(
        EC.visibility_of_element_located((By.CLASS_NAME, "ewa-stb-tabs"))
    )
    working_sheet_xpath = (
        "//a[contains(@class,'tab-anchor-text') and"
        " contains(@role, 'tab') "
        "and (@aria-label='Sheet1')]"
    )
    wait.until(
        EC.visibility_of_element_located((By.XPATH, working_sheet_xpath))
    )
    first_page = driver.find_element_by_xpath(working_sheet_xpath)
    first_page.click()
    LOGGER.info("end of select_and_start_from_first_page method.")


def select_sheet_content(driver, wait):
    """Helper function to select all the elements in the Excel sheet (Ctrl+a).

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: The WebDriverWait instance used for waiting on conditions.
    """
    LOGGER.info("start select_sheet_content method.")
    sheet_content = "m_excelWebRenderer_ewaCtl_sheetContentDiv_Flow_0"
    wait.until(EC.visibility_of_element_located((By.ID, sheet_content)))
    sheet_content = driver.find_element_by_id(sheet_content)
    doc_editing_utils.select_sheet_content(driver, sheet_content)
    LOGGER.info("end of select_sheet_content method.")


@time_measure()
def create_stacked_bar_chart(driver, wait):
    """Helper function to Create a stacked bar from the existing values in
     the Excel sheet.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: The WebDriverWait instance used for waiting on conditions.
    """
    LOGGER.info("start create_stacked_bar_chart method.")
    wait.until(EC.visibility_of_element_located((By.ID, "Insert")))
    driver.find_element_by_id("Insert").click()
    # select pivot table from bar
    wait.until(
        EC.visibility_of_element_located(
            (By.ID, "InsertChart-previewExpandButton")
        )
    )
    driver.find_element_by_id("InsertChart-previewExpandButton").click()
    wait.until(
        EC.visibility_of_element_located(
            (By.ID, "Ribbon-InsertChart-previewExpandButtonDropdown")
        )
    )
    wait.until(
        EC.visibility_of_element_located(
            (
                By.ID,
                "title-InsertChartGallery-Bar",
            )
        )
    )
    wait.until(
        EC.presence_of_element_located((By.ID, "Insert2DBarChartStacked"))
    )
    wait.until(
        EC.visibility_of_element_located((By.ID, "Insert2DBarChartStacked"))
    )
    wait.until(EC.element_to_be_clickable((By.ID, "Insert2DBarChartStacked")))
    driver.find_element_by_id("Insert2DBarChartStacked").click()
    LOGGER.info("end of create_stacked_bar_chart method.")


def delete_stacked_bar_chart(driver, wait):
    """Helper function to delete the stacked bar chart that we create it before.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: The WebDriverWait instance used for waiting on conditions.
    """
    LOGGER.info("start delete_stacked_bar_chart method.")
    wait.until(
        EC.visibility_of_element_located((By.CLASS_NAME, "ewa-fo-img-div"))
    )
    actions = ActionChains(driver)
    actions.send_keys(Keys.DELETE)
    actions.perform()
    LOGGER.info("end of delete_stacked_bar_chart method.")


@time_measure()
def create_pivot_table_with_new_sheet(driver, wait):
    """Helper function to create a pivot table for the Excel.

     This function will create the pivot table and select the required rows
     and values for this pivot table.

    Args:
        driver: The WebDriver instance used to interact with the browser.
        wait: The WebDriverWait instance used for waiting on conditions.
    """
    LOGGER.info("start create_pivot_table_with_new_sheet method.")
    driver.find_element_by_id("Insert").click()
    # select pivot table from bar
    wait.until(
        EC.visibility_of_element_located(
            (By.XPATH, '//*[@id="InsertPivotTableMenu"]/button[1]')
        )
    )
    driver.find_element_by_xpath(
        '//*[@id="InsertPivotTableMenu"]/button[1]'
    ).click()

    pivot_frame_xpath = "//iframe[contains(@id,'.Insights.Https.Bundle')]"
    pivot_option_element = wait.until(
        lambda d: d.find_elements_by_id("newWorksheet")
        or d.find_elements_by_xpath(pivot_frame_xpath)
    )[0]
    if pivot_option_element.tag_name == "iframe":
        driver.switch_to.frame(pivot_option_element)
        wait.until(EC.visibility_of_element_located((By.ID, "recommendation0")))
        driver.find_element_by_xpath(
            '//*[@id="recommendation0_InsertOnNew"]/span/span[1]'
        ).click()
        driver.switch_to.parent_frame()
    else:
        wait.until(EC.visibility_of_element_located((By.ID, "newWorksheet")))
        driver.find_element_by_id("newWorksheet").click()
        driver.find_element_by_id("WACDialogActionButton").click()

    # wait Pane of selection to appear
    wait.until(EC.visibility_of_element_located((By.ID, "FarPane")))
    wait.until(
        EC.visibility_of_element_located((By.CLASS_NAME, "ms-List-cell"))
    )
    pivot_elements = driver.find_elements_by_xpath(
        "//div[starts-with(@class,'checkbox-checkbox checkbox')]"
    )
    for i in range(3):
        time.sleep(1)
        pivot_elements[i].click()
    time.sleep(1)
    wait_saving_status(wait)
    LOGGER.info("end of create_pivot_table_with_new_sheet method.")
