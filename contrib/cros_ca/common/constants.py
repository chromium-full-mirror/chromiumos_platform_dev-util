# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable = too-few-public-methods
"""Global Constants values."""
import getpass
import logging
import os

from dotenv import load_dotenv

from common.common_utility import check_out_dir_creation_required

CURRENT_OS_USER_NAME = getpass.getuser()
CURRENT_DIR_PATH = os.path.dirname(os.path.abspath(__file__))
load_dotenv(os.path.join(CURRENT_DIR_PATH, "../.env"))


class DriversAndAppsPaths:
    """This class contains the drivers paths in the local machine to use them
    in the automation code.
    """

    LINUX_CHROME_WEB_DRIVER_PATH = r"/usr/local/chromedriver/chromedriver"
    WIN_APP_DRIVER_PATH = os.getenv("WIN_APP_DRIVER_PATH")
    EXCEL_APP_PATH = (
        r"C:\Program Files\Microsoft Office\root\Office16\EXCEL.exe"
    )
    EXCEL_APP_ARGUMENTS = os.path.join(
        CURRENT_DIR_PATH, "../resources/native_app_files/CarsExample.xlsx"
    )

    WORD_APP_PATH = (
        r"C:\Program Files\Microsoft Office\root\Office16\WINWORD.EXE"
    )
    WORD_APP_ARGUMENTS = os.path.join(
        CURRENT_DIR_PATH, "../resources/native_app_files/HelloWorld.docx"
    )

    PPT_APP_PATH = (
        r"C:\Program Files\Microsoft Office\root\Office16\POWERPNT.EXE"
    )
    PPT_APP_ARGUMENTS = os.path.join(
        CURRENT_DIR_PATH, "../resources/native_app_files/Benchmarking.pptx"
    )

    XML_WEBSITES_URL_PATH = os.path.join(
        CURRENT_DIR_PATH, "../resources/browse_websites_by_url.xml"
    )

    WINDOWS_PHOTOS_APP_ID = "Microsoft.Windows.Photos_8wekyb3d8bbwe!App"

    TEST_PHOTO_URL_DIR_PATH = os.path.join(
        CURRENT_DIR_PATH, "../resources/native_app_files"
    )

    TEST_PHOTO_NAME = "TestImage.png"

    WINDOWS_VIDEOS_CAPTURES_DIR_PATH = (
        rf"C:\Users\{CURRENT_OS_USER_NAME}\Videos\Captures"
    )


class PageLoadConfig:
    """This class stores configuration constants related to page load times
    and timeouts."""

    MAX_PAGE_LOAD_TIME = 100
    WAIT_TIMEOUT = 50


class ProcessNames:
    """This class contains the process names and types for the OSs and the
    applications used in the automation code.
    """

    PROCESS_NAME_TO_BE_EXCLUDE = (
        "chrome_crashpad_handler",
        "chromedriver",
        "msedgedriver",
        "msedge_crashpad",
        "msedge_crashpad_handler",
    )
    EXCEL = "EXCEL.EXE"
    PPT = "POWERPNT.EXE"
    WORD = "WINWORD.EXE"
    AI_PPT = "ai.exe"
    PHOTOS_SERVICE = "PhotosService.exe"
    PHOTOS_APP = "PhotosApp.exe"
    WIN_GAME_BAR_APP = "GameBar.exe"
    WIN_GAME_BAR_SERVER_APP = "GameBarFTServer.exe"
    WINDOWS_CHROME = "chrome.exe"
    LINUX_CHROME = "chrome"
    WINDOWS_EDGE = "msedge.exe"
    WINDOWS_PLATFORM_TYPE = "Windows"
    CHROME_BROWSER_TYPE = "chrome"
    EDGE_BROWSER_TYPE = "edge"


class TestingAccountLicense:
    """This class contains the license for the account that will be used for
    testing.
    """

    LOGIN_ACCOUNT_EMAIL = os.getenv("LOGIN_ACCOUNT_EMAIL")
    LOGIN_ACCOUNT_PASSWORD = os.getenv("LOGIN_ACCOUNT_PASSWORD")


class UseCasesURLS:
    """This class contains the required URLS to be used by the use cases in
    the automation code.
    """

    GOOGLE_DRIVE_URL_PATH = "https://drive.google.com/drive/my-drive"
    MS_DRIVE_URL_PATH = "https://onedrive.live.com/?id=root"
    WIN_APP_DRIVER_URL_PATH = "http://127.0.0.1:4723"
    PDF_WEB_URL_PATH = "file:///" + os.path.join(
        CURRENT_DIR_PATH, "../resources/native_app_files/HelloWorld.pdf"
    )


class DocumentAccountType:
    """This class contains constants used in account
    log in operation to differentiate if you should
    log in to google or microsoft account.
    """

    GOOGLE = "Google"
    MICROSOFT = "Microsoft"


class HumanDelay:
    """This class contains constants used to be added to total time for
    some use case as human delay when human try to simulate running
    the use case by hand.
    """

    EXPECTED_CLICK_OR_ACTION_TIME = 0.150
    GOOGLE_EXCEL_WEB = 22 * EXPECTED_CLICK_OR_ACTION_TIME
    GOOGLE_PPT_WEB = 7 * EXPECTED_CLICK_OR_ACTION_TIME
    GOOGLE_WORD_WEB = 9 * EXPECTED_CLICK_OR_ACTION_TIME
    MICROSOFT_EXCEL_WEB = 19 * EXPECTED_CLICK_OR_ACTION_TIME
    MICROSOFT_PPT_WEB = 8 * EXPECTED_CLICK_OR_ACTION_TIME
    MICROSOFT_WORD_WEB = 5 * EXPECTED_CLICK_OR_ACTION_TIME


class UserAgent:
    """This class contains constants value for user agents
    based on platform."""

    WINDOWS_USER_AGENT = (
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
        "AppleWebKit/537.36 (KHTML, like Gecko) "
        "Chrome/121.0.0.0 Safari/537.36"
    )
    LINUX_USER_AGENT = (
        "Mozilla/5.0 (X11; Linux x86_64) "
        "AppleWebKit/537.36 (KHTML, like Gecko) "
        "Chrome/121.0.0.0 Safari/537.36"
    )


class LoggingConfig:
    """Class contains Logging configuration."""

    TRACES_FORMAT = "%(asctime)s.%(msecs)03d %(levelname)s| %(filename)s:%(lineno)s| %(message)s"
    RESULTS_FORMAT = "%(asctime)s - %(message)s"
    DATE_FORMAT = "%m/%d %H:%M:%S"
    TRACES_FILE = "/logs.log"
    RESULTS_FILE = "/results.log"
    LOGS_FILE_DIR_PATH = check_out_dir_creation_required()

    @classmethod
    def get_logger(cls, is_traces_log: bool = True):
        """Method used to get the logger with specific configuration.
        The Logger have 2 configuration:
        1. traces logger configuration for printing traces logs of code.
        2. results logger configuration for printing the results of
        benchmarking.

        Args:
            is_traces_log: Indicate if returned logger should be
                traces or results logger.
        """
        name = "traces" if is_traces_log else "results"
        logger = logging.getLogger(name)
        if not logger.hasHandlers():
            if is_traces_log:
                log_file_path = (
                    LoggingConfig.LOGS_FILE_DIR_PATH + LoggingConfig.TRACES_FILE
                )
                formatter = logging.Formatter(
                    fmt=LoggingConfig.TRACES_FORMAT,
                    datefmt=LoggingConfig.DATE_FORMAT,
                )
            else:
                log_file_path = (
                    LoggingConfig.LOGS_FILE_DIR_PATH
                    + LoggingConfig.RESULTS_FILE
                )
                formatter = logging.Formatter(
                    fmt=LoggingConfig.RESULTS_FORMAT,
                    datefmt=LoggingConfig.DATE_FORMAT,
                )

            handler = logging.FileHandler(log_file_path)
            handler.setFormatter(formatter)
            logger.setLevel(logging.INFO)
            logger.addHandler(handler)

        return logger
