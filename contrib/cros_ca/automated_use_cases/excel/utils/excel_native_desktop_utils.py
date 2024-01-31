# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Microsoft Excel native desktop use case utils."""

import time

from selenium.webdriver import ActionChains
from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.support import expected_conditions as EC

from automated_use_cases.common_utility import doc_editing_utils
from benchmarking.time_benchmark import time_measure
from common.constants import LoggingConfig

LOGGER = LoggingConfig.get_logger()


def select_sheet_content(driver, wait):
    """Function to select sheet content.

    Args:
        driver: The Windows Desktop App Driver instance used to
            interact with the desktop app.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
    """
    LOGGER.info("start select_sheet_content method.")
    sheet_grid = "XLSpreadsheetGrid"
    wait.until(EC.visibility_of_element_located((By.CLASS_NAME, sheet_grid)))
    sheet_content = driver.find_element_by_class_name(sheet_grid)
    doc_editing_utils.select_sheet_content(driver, sheet_content)
    LOGGER.info("end of select_sheet_content method.")


@time_measure()
def create_stacked_bar_chart(driver, wait):
    """Function to create stacked bar chart.

    Args:
        driver: The Windows Desktop App Driver instance used to
            interact with the desktop app.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
    """
    LOGGER.info("start create_stacked_bar_chart method.")
    # insert
    driver.find_element_by_accessibility_id("TabInsert").click()
    wait.until(EC.visibility_of_element_located((By.CLASS_NAME, "NetUIAnchor")))
    driver.find_element_by_accessibility_id(
        "ChartTypeColumnInsertGallery"
    ).click()
    wait.until(
        EC.visibility_of_all_elements_located(
            (By.CLASS_NAME, "NetUIGalleryButton")
        )
    )
    driver.find_element_by_name("Stacked Bar").click()
    LOGGER.info("end of create_stacked_bar_chart method.")


def delete_stacked_bar_chart(driver, wait):
    """Function to delete stacked bar chart.

    Args:
        driver: The Windows Desktop App Driver instance used to
            interact with the desktop app.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
    """
    LOGGER.info("start delete_stacked_bar_chart method.")
    wait.until(EC.visibility_of_element_located((By.NAME, "Chart Area")))
    actions = ActionChains(driver)
    actions.send_keys(Keys.DELETE)
    actions.perform()
    LOGGER.info("end of delete_stacked_bar_chart method.")


@time_measure()
def create_pivot_table(driver, wait):
    """Function to create pivot table.

    Args:
        driver: The Windows Desktop App Driver instance used to
            interact with the desktop app.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
    """
    LOGGER.info("start create_pivot_table method.")
    # insert
    driver.find_element_by_accessibility_id("TabInsert").click()
    # select pivot table from bar
    wait.until(
        EC.visibility_of_element_located((By.CLASS_NAME, "NetUIRibbonButton"))
    )
    driver.find_element_by_name("PivotTable").click()
    # wait new page
    current_window_handle = driver.current_window_handle
    wait.until(EC.number_of_windows_to_be(2))
    all_win_handle = driver.window_handles
    driver.switch_to.window(all_win_handle[0])
    wait.until(EC.visibility_of_element_located((By.NAME, "New Worksheet")))
    driver.find_element_by_name("New Worksheet").click()
    # press ok
    driver.find_element_by_name("OK").click()
    ###
    driver.switch_to.window(current_window_handle)
    wait.until(
        EC.visibility_of_element_located((By.CLASS_NAME, "NetUInetpane"))
    )
    pivot_elements = driver.find_elements_by_tag_name("CheckBox")
    for i in range(3):
        time.sleep(1)
        pivot_elements[i].click()
    time.sleep(1)
    LOGGER.info("end of create_pivot_table method.")
