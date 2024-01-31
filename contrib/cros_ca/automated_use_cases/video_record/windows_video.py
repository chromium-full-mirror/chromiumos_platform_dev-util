#!/usr/bin/env python3
# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable= E1120, W1203
"""Windows Photos Editing/Crop native desktop use case."""
import os

import pyautogui
from selenium.webdriver.common.by import By
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.wait import WebDriverWait

from automated_use_cases.common_utility.get_website_util import (
    load_website_by_url,
)
from benchmarking.benchmarking_test import (
    BenchmarkingTest,
)
from benchmarking.performance_decorator import performance
from benchmarking.platform_benchmarking import (
    setup,
    get_windows_chrome_web_driver,
)
from benchmarking.time_benchmark import time_measure
from common.constants import (
    ProcessNames,
    PageLoadConfig,
    DriversAndAppsPaths,
    LoggingConfig,
)

LOGGER = LoggingConfig.get_logger()

PROCESS_NAME = (
    ProcessNames.WIN_GAME_BAR_APP,
    ProcessNames.WIN_GAME_BAR_SERVER_APP,
)


@time_measure()
def record_video(recorded_video_time):
    """Function to start recording a video for specific period of time
    using Game Bar application for windows.

    Args:
        recorded_video_time: Specific period of time indicates
            total video time
    """
    LOGGER.info(
        "start record_video method with "
        f"recorded_video_time = {recorded_video_time}."
    )
    # start video recording
    pyautogui.hotkey("win", "alt", "r")
    # sleep for total video time
    pyautogui.sleep(recorded_video_time)
    # stop video recording
    pyautogui.hotkey("win", "alt", "r")
    LOGGER.info("end of record_video method.")


class WindowsVideoRecordingNativeDesktopTest(BenchmarkingTest):
    """Test class for Windows Photos Editing/Crop native desktop use case."""

    def __init__(self):
        self.driver = None
        self.video_url = "https://www.youtube.com/watch?v=YXMo5w9aMNs"
        self.video_name_prefix = "Wolves 101 _ Nat Geo Wild"
        self.recorded_video_time = 10
        self.videos_dir = DriversAndAppsPaths.WINDOWS_VIDEOS_CAPTURES_DIR_PATH

    def set_up(self):
        """This method will perform the following:
        1. Get Windows Chrome Web Driver
        2. Open and load specific video url like YouTube in this case.
        3. Wait cookies popup and reject it.
        """
        LOGGER.info(f"start set_up method in {self.__class__.__name__}")
        self.driver, _ = get_windows_chrome_web_driver()
        self.driver.maximize_window()
        wait = WebDriverWait(self.driver, PageLoadConfig.WAIT_TIMEOUT)
        load_website_by_url(self.driver, self.video_url)
        wait.until(
            EC.element_to_be_clickable(
                (By.XPATH, "//button[starts-with(@aria-label,'Reject')]")
            )
        ).click()
        LOGGER.info(f"end of set_up method in {self.__class__.__name__}")

    def execute(self):
        LOGGER.info(f"start execute method in {self.__class__.__name__}")

        @performance(process_name=PROCESS_NAME)
        @setup(process_name=PROCESS_NAME, is_execution_part=True)
        def run():
            """Run Windows Video Recording native desktop use case
            using Game Bar application for windows.

            This function will perform the following:

            1. Start recording the YouTube tab in Chrome browser for
            specific time like 10s in this case.
            2. Stop recording the video.
            """
            record_video(self.recorded_video_time)

        run()
        LOGGER.info(f"end of execute method in {self.__class__.__name__}")

    def teardown(self):
        """This method will perform delete for recorded video from
        Videos/Captures directory path in Windows platform
        and quit opened webdriver."""
        LOGGER.info(f"start teardown method in {self.__class__.__name__}")
        self.driver.quit()
        files_name = os.listdir(self.videos_dir)
        for file_name in files_name:
            if file_name.startswith(self.video_name_prefix):
                os.remove(self.videos_dir + rf"\{file_name}")
        LOGGER.info(f"end of teardown method in {self.__class__.__name__}")


if __name__ == "__main__":
    WindowsVideoRecordingNativeDesktopTest().run_test()
