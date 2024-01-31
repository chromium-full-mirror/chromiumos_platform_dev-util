#!/usr/bin/env python3
# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable= E1120, W1203
"""Microsoft Excel web use case."""
import uuid

from selenium.webdriver.support.wait import WebDriverWait

from automated_use_cases.common_utility.doc_editing_utils import (
    open_and_load_account_document_or_drive,
)
from automated_use_cases.common_utility.microsoft_documents_utils import (
    switch_to_document_frame,
    check_ribbon_container_if_open_needed,
    delete_created_document_from_one_drive,
    upload_document_to_one_drive,
)
from automated_use_cases.excel.utils.excel_microsoft_utils import (
    select_and_start_from_first_page,
    select_sheet_content,
    create_stacked_bar_chart,
    delete_stacked_bar_chart,
    create_pivot_table_with_new_sheet,
)
from benchmarking.benchmarking_test import (
    BenchmarkingTest,
)
from benchmarking.performance_decorator import performance
from benchmarking.platform_benchmarking import setup, get_browser_process_name
from common.common_utility import get_file_path_to_upload
from common.constants import (
    UseCasesURLS,
    PageLoadConfig,
    DocumentAccountType,
    HumanDelay,
    LoggingConfig,
)

LOGGER = LoggingConfig.get_logger()

BROWSER_PROCESS_NAME = get_browser_process_name()


class ExcelMSWebTest(BenchmarkingTest):
    """Test class for Microsoft Excel web use case."""

    def __init__(self):
        self.new_doc_url = None
        self.doc_extension_type = ".xlsx"
        self.new_doc_name = str(uuid.uuid4()) + self.doc_extension_type
        self.frame_name = "WacFrame_Excel_0"
        self.doc_account_type = DocumentAccountType.MICROSOFT
        self.account_drive_url = UseCasesURLS.MS_DRIVE_URL_PATH
        self.excel_file_path = get_file_path_to_upload(
            self.new_doc_name, self.doc_extension_type
        )

    def set_up(self):
        LOGGER.info(f"start set_up method in {self.__class__.__name__}")

        @setup(is_browsing=True, process_name=BROWSER_PROCESS_NAME)
        def run(driver):
            """Run the set_up of use case for Excel web Microsoft Sheet.

            This function will perform the following:
            1. Open the testing One Drive using the browser.
            2. Wait until the login page finish reloading then login
            to the Microsoft account.
            3. Create new Excel document.
            4. Add dummy data to Excel document.
            5. Change Excel document name to be equal uuid.

            This function is decorated with the @setup decorator,
            configuring the driver for the browser processes.

            Args:
                driver: The WebDriver instance used to interact with
                    the browser.
            """
            excel_wait = WebDriverWait(driver, PageLoadConfig.WAIT_TIMEOUT)
            open_and_load_account_document_or_drive(
                driver, self.account_drive_url, self.doc_account_type
            )
            self.new_doc_url = upload_document_to_one_drive(
                driver,
                excel_wait,
                self.new_doc_name,
                self.frame_name,
                self.excel_file_path,
            )

        run()
        LOGGER.info(f"end of set_up method in {self.__class__.__name__}")

    def execute(self):
        LOGGER.info(f"start execute method in {self.__class__.__name__}")

        @performance(process_name=BROWSER_PROCESS_NAME)
        @setup(
            is_browsing=True,
            process_name=BROWSER_PROCESS_NAME,
            is_execution_part=True,
        )
        def run(driver):
            """Run the use case for Excel web Microsoft Sheet.

            This function will perform the following:
            1. Open the testing Microsoft Sheet using the browser.
            2. Wait until the login page finish reloading then login to
            the Microsoft account.
            3. Wait until testing Microsoft sheet page finish reloading.
            4. Select and start from first working sheet
            5. Select the Microsoft Sheet elements and create a Stacked
            Bar Graph.
            6. Delete the created graph.
            7. Create a Pivot Table in a new sheet with the data
            and wait for the
            save status to be saved.
            8. Delete existing new sheets except first working sheet and wait
            for the save status to be saved.
            9. Close the browser.

            This function is decorated with the @setup decorator,
            configuring the driver for the browser processes.

            Args:
                driver: The WebDriver instance used to interact with
                    the browser.

            Returns:
                dict: Dictionary containing variables to be used in
                    measurements process.
            """
            excel_wait = WebDriverWait(driver, PageLoadConfig.WAIT_TIMEOUT)
            login_time = open_and_load_account_document_or_drive(
                driver, self.new_doc_url, self.doc_account_type
            )

            switch_to_document_frame(driver, excel_wait, self.frame_name)
            select_and_start_from_first_page(driver, excel_wait)
            select_sheet_content(driver, excel_wait)
            check_ribbon_container_if_open_needed(driver)
            create_stacked_bar_chart(driver, excel_wait)
            delete_stacked_bar_chart(driver, excel_wait)
            create_pivot_table_with_new_sheet(driver, excel_wait)
            return {
                "period_of_login_time": login_time,
                "human_delay": HumanDelay.MICROSOFT_EXCEL_WEB,
            }

        run()
        LOGGER.info(f"end of execute method in {self.__class__.__name__}")

    def teardown(self):
        LOGGER.info(f"start teardown method in {self.__class__.__name__}")

        @setup(is_browsing=True, process_name=BROWSER_PROCESS_NAME)
        def run(driver):
            """Run the teardown of use case for Excel web Microsoft Sheet.

            This function will perform the following:
            1. Open the testing Microsoft One Drive using the browser.
            2. Wait until the login page finish reloading then login to
            the Microsoft account.
            3. delete the copy of original sheet document from One Drive.

            This function is decorated with the @setup decorator,
            configuring the driver for the browser processes.

            Args:
                driver: The WebDriver instance used to interact with
                    the browser.
            """
            excel_wait = WebDriverWait(driver, PageLoadConfig.WAIT_TIMEOUT)
            open_and_load_account_document_or_drive(
                driver, self.account_drive_url, self.doc_account_type
            )
            delete_created_document_from_one_drive(
                driver, excel_wait, self.new_doc_name
            )

        run()
        LOGGER.info(f"end of teardown method in {self.__class__.__name__}")


if __name__ == "__main__":
    ExcelMSWebTest().run_test()
