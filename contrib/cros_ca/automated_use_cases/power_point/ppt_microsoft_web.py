#!/usr/bin/env python3
# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable= E1120, W1203
"""Microsoft PowerPoint web use case."""
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
from automated_use_cases.power_point.utils.ppt_microsoft_utils import (
    navigate_to_last_slide,
    add_comment_to_last_slide,
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


class PowerPointMSWebTest(BenchmarkingTest):
    """Test class for Microsoft PowerPoint web use case."""

    def __init__(self):
        self.new_doc_url = None
        self.doc_extension_type = ".pptx"
        self.new_doc_name = str(uuid.uuid4()) + self.doc_extension_type
        self.frame_name = "WacFrame_PowerPoint_0"
        self.doc_account_type = DocumentAccountType.MICROSOFT
        self.account_drive_url = UseCasesURLS.MS_DRIVE_URL_PATH
        self.ppt_file_path = get_file_path_to_upload(
            self.new_doc_name, self.doc_extension_type
        )

    def set_up(self):
        LOGGER.info(f"start set_up method in {self.__class__.__name__}")

        @setup(is_browsing=True, process_name=BROWSER_PROCESS_NAME)
        def run(driver):
            """Run the set_up of use case for Microsoft web Slides.

            This function will perform the following:
            1. Open the testing One Drive using the browser.
            2. Wait until the login page finish reloading then login
            to the Microsoft account.
            3. Create new PowerPoint document.
            4. Add dummy data to PowerPoint document.
            5. Change PowerPoint document name to be equal uuid.

            This function is decorated with the @setup decorator,
            configuring the driver for the browser processes.

            Args:
                driver: The WebDriver instance used to interact with
                    the browser.
            """
            ppt_wait = WebDriverWait(driver, PageLoadConfig.WAIT_TIMEOUT)
            open_and_load_account_document_or_drive(
                driver, self.account_drive_url, self.doc_account_type
            )
            self.new_doc_url = upload_document_to_one_drive(
                driver,
                ppt_wait,
                self.new_doc_name,
                self.frame_name,
                self.ppt_file_path,
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
            """Run Microsoft web Slides use case.

            This function will perform the following:
            1. Open the testing Microsoft Slides using the browser.
            2. Wait until the login page finish reloading then login to
            the Microsoft account.
            3. Wait until testing Microsoft Slides page finish reloading.
            4. Navigate to the last slide.
            5. Add comment on the last slide and wait the saving status.
            6. Delete existing comments on the last slide and wait the
            saving status.
            7. Close the browser.

            This function is decorated with the @setup decorator,
            configuring the driver for the browser processes.

            Args:
                driver: The WebDriver instance used to interact with
                    the browser.

            Returns:
                dict: Dictionary containing variables to be used in
                    measurements process.
            """
            ppt_wait = WebDriverWait(driver, PageLoadConfig.WAIT_TIMEOUT)
            login_time = open_and_load_account_document_or_drive(
                driver, self.new_doc_url, self.doc_account_type
            )
            switch_to_document_frame(driver, ppt_wait, self.frame_name)
            navigate_to_last_slide(driver, ppt_wait)
            check_ribbon_container_if_open_needed(driver)
            add_comment_to_last_slide(driver, ppt_wait)
            return {
                "period_of_login_time": login_time,
                "human_delay": HumanDelay.MICROSOFT_PPT_WEB,
            }

        run()
        LOGGER.info(f"end of execute method in {self.__class__.__name__}")

    def teardown(self):
        LOGGER.info(f"start teardown method in {self.__class__.__name__}")

        @setup(is_browsing=True, process_name=BROWSER_PROCESS_NAME)
        def run(driver):
            """Run the teardown of use case for Microsoft web Slides.

            This function will perform the following:
            1. Open the testing Microsoft One Drive using the browser.
            2. Wait until the login page finish reloading then login to
            the Microsoft account.
            3. delete the copy of original slides document from One Drive.

            This function is decorated with the @setup decorator,
            configuring the driver for the browser processes.

            Args:
                driver: The WebDriver instance used to interact with
                    the browser.
            """
            ppt_wait = WebDriverWait(driver, PageLoadConfig.WAIT_TIMEOUT)
            open_and_load_account_document_or_drive(
                driver, self.account_drive_url, self.doc_account_type
            )
            delete_created_document_from_one_drive(
                driver, ppt_wait, self.new_doc_name
            )

        run()
        LOGGER.info(f"end of teardown method in {self.__class__.__name__}")


if __name__ == "__main__":
    PowerPointMSWebTest().run_test()
