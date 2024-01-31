#!/usr/bin/env python3
# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable= E1120, W1203
"""Microsoft Excel native desktop use case."""

from selenium.webdriver.support.wait import WebDriverWait

from automated_use_cases.common_utility.doc_editing_utils import (
    open_win_app_driver,
    close_win_app_driver,
)
from automated_use_cases.common_utility.microsoft_documents_utils import (
    close_native_app_document_without_save,
    switch_to_native_app_window,
)
from automated_use_cases.excel.utils.excel_native_desktop_utils import (
    select_sheet_content,
    create_stacked_bar_chart,
    delete_stacked_bar_chart,
    create_pivot_table,
)
from benchmarking.benchmarking_test import (
    BenchmarkingTest,
)
from benchmarking.performance_decorator import performance
from benchmarking.platform_benchmarking import setup
from common.constants import ProcessNames, PageLoadConfig, LoggingConfig

LOGGER = LoggingConfig.get_logger()

PROCESS_NAME = (ProcessNames.EXCEL,)


class ExcelNativeDesktopTest(BenchmarkingTest):
    """Test class for Microsoft Excel native desktop use case."""

    def set_up(self):
        LOGGER.info(f"start set_up method in {self.__class__.__name__}")
        open_win_app_driver()
        LOGGER.info(f"end of set_up method in {self.__class__.__name__}")

    def execute(self):
        LOGGER.info(f"start execute method in {self.__class__.__name__}")

        @performance(process_name=PROCESS_NAME)
        @setup(process_name=PROCESS_NAME, is_execution_part=True)
        def run(driver):
            """Run the use case for Excel native desktop Microsoft Sheet.

            This function will perform the following:
            1. Open the testing Microsoft Sheet using the native Excel app.
            2. Wait until testing Microsoft Excel be visible.
            3. Select the Microsoft Sheet elements and create a Stacked Bar
            Graph.
            4. Delete the created graph.
            5. Create a Pivot Table in a new sheet with the data.
            6. Close the app and select don't save option.

            This function is decorated with the @setup decorator,
            configuring the driver for the native desktop app processes.

            Args:
                driver: The Windows Desktop App Driver instance used to
                    interact with the desktop app.
            """
            wait = WebDriverWait(driver, PageLoadConfig.WAIT_TIMEOUT)
            switch_to_native_app_window(driver, wait)
            select_sheet_content(driver, wait)
            create_stacked_bar_chart(driver, wait)
            delete_stacked_bar_chart(driver, wait)
            create_pivot_table(driver, wait)
            close_native_app_document_without_save(driver)

        run()
        LOGGER.info(f"end of execute method in {self.__class__.__name__}")

    def teardown(self):
        LOGGER.info(f"start teardown method in {self.__class__.__name__}")
        close_win_app_driver()
        LOGGER.info(f"end of teardown method in {self.__class__.__name__}")


if __name__ == "__main__":
    ExcelNativeDesktopTest().run_test()
