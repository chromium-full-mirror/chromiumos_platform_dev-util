# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable= W0603, W1203, C0415
"""
Script used to determine the platform, and it's requirement
needed before running any use case.
"""

import platform
import sys
import time
import uuid

from selenium import webdriver as selenium_chrome_webdriver
from selenium.common.exceptions import NoSuchWindowException

from benchmarking import time_benchmark
from benchmarking.benchmarking_exception import BenchmarkingException
from common.constants import (
    DriversAndAppsPaths,
    UseCasesURLS,
    ProcessNames,
    UserAgent,
    LoggingConfig,
)

LOGGER = LoggingConfig.get_logger()
RESULTS_LOGGER = LoggingConfig.get_logger(False)


def get_browser_process_name():
    """Function used to determined process_name for any use case that
    use web task, and to decide which browser will be used based on
    first arguments variable passed by command line while running
    any use case.

    Returns:
        process_name based on OS type and first arguments variable,
            so Chrome browser will be the default unless "edge" keyword passed
            as first arguments variable to use Edge browser.
    """
    LOGGER.info("executing get_browser_process_name method.")
    arguments_variable = sys.argv
    browser_type = (
        arguments_variable[1] if len(arguments_variable) > 1 else None
    )
    if browser_type is None or browser_type == ProcessNames.CHROME_BROWSER_TYPE:
        if platform.system() == ProcessNames.WINDOWS_PLATFORM_TYPE:
            return (ProcessNames.WINDOWS_CHROME,)
        return (ProcessNames.LINUX_CHROME,)
    if browser_type == ProcessNames.EDGE_BROWSER_TYPE:
        if platform.system() == ProcessNames.WINDOWS_PLATFORM_TYPE:
            return (ProcessNames.WINDOWS_EDGE,)
        raise BenchmarkingException(
            "edge browser for linux platform not supported."
        )
    raise BenchmarkingException(
        "first arguments passed for web browser is not supported."
    )


def get_browser_web_driver(process_name):
    """Function used to determined which web driver should be used.

    Args:
        process_name: Process_name of use case.

    Returns:
        Web driver.
    """
    LOGGER.info("executing get_browser_web_driver method.")
    if ProcessNames.WINDOWS_CHROME in process_name:
        return get_windows_chrome_web_driver()
    if ProcessNames.LINUX_CHROME in process_name:
        return get_linux_chrome_web_driver()
    if ProcessNames.WINDOWS_EDGE in process_name:
        return get_windows_edge_web_driver()
    raise BenchmarkingException(
        "for browsing test, either (Windows, Linux) chrome "
        "or (Windows) edge should selected."
    )


def get_windows_chrome_web_driver():
    """Function to get Chrome web driver for Windows platform.

    Returns:
        Windows Chrome web driver.
    """
    LOGGER.info("executing get_windows_chrome_web_driver method.")
    chrome_options = selenium_chrome_webdriver.ChromeOptions()
    chrome_options.add_argument("--incognito")
    chrome_options.add_argument("--disable-application-cache")
    chrome_options.add_argument("--disable-extensions")
    chrome_options.add_argument(f"--user-agent={get_random_user_agent()}")
    from webdriver_manager.chrome import ChromeDriverManager

    chrome_driver_path = ChromeDriverManager().install()
    time_before_run_use_case = time.time()
    driver = selenium_chrome_webdriver.Chrome(
        executable_path=chrome_driver_path,
        options=chrome_options,
    )
    return driver, time_before_run_use_case


old_windows = []


def get_linux_chrome_web_driver():
    """Function to get Chrome web driver for Linux platform.

    Returns:
        Linux Chrome web driver.
    """
    LOGGER.info("executing get_linux_chrome_web_driver method.")
    chrome_options = selenium_chrome_webdriver.ChromeOptions()
    chrome_options.add_experimental_option("debuggerAddress", "127.0.0.1:9222")
    chrome_options.add_argument("--incognito")
    chrome_options.add_argument("--disable-application-cache")
    chrome_options.add_argument("--disable-extensions")
    chrome_options.add_argument(f"--user-agent={get_random_user_agent()}")
    time_before_run_use_case = time.time()
    driver = selenium_chrome_webdriver.Chrome(
        executable_path=DriversAndAppsPaths.LINUX_CHROME_WEB_DRIVER_PATH,
        options=chrome_options,
    )
    global old_windows
    old_windows = driver.window_handles
    # Open a new window for running tests
    driver.execute_script("chrome.windows.create({incognito:true})")
    test_window = [w for w in driver.window_handles if w not in old_windows][0]
    driver.switch_to.window(test_window)
    return driver, time_before_run_use_case


def get_windows_edge_web_driver():
    """Function to get Edge web driver for Windows platform.

    Returns:
        Windows Edge web driver.
    """
    LOGGER.info("executing get_windows_edge_web_driver method.")
    from msedge.selenium_tools import Edge, EdgeOptions

    edge_options = EdgeOptions()
    edge_options.use_chromium = True
    edge_options.add_argument("inprivate")
    edge_options.add_argument("--disable-application-cache")
    edge_options.add_argument("--disable-extensions")
    edge_options.add_argument(f"--user-agent={get_random_user_agent()}")
    from webdriver_manager.microsoft import EdgeChromiumDriverManager

    edge_driver_path = EdgeChromiumDriverManager().install()
    time_before_run_use_case = time.time()
    driver = Edge(
        executable_path=edge_driver_path,
        options=edge_options,
    )
    return driver, time_before_run_use_case


def get_native_app_driver(process_name):
    """Function used to determined which native app driver should be used
    based on OS type.

    Args:
        process_name: process_name of use case.

    Returns:
        native app driver.
    """
    LOGGER.info("executing get_native_app_driver method.")
    if platform.system() == ProcessNames.WINDOWS_PLATFORM_TYPE:
        return get_windows_native_app_driver(process_name)
    return get_linux_native_app_driver(process_name)


def get_windows_native_app_driver(process_name):
    """Function used to determined which native app driver should be used
    for Windows OS type.

    Args:
        process_name: process_name of use case.

    Returns:
        Windows native app driver.
    """
    LOGGER.info("executing get_windows_native_app_driver method.")
    if ProcessNames.EXCEL in process_name:
        desired_caps = {
            "app": DriversAndAppsPaths.EXCEL_APP_PATH,
            "appArguments": DriversAndAppsPaths.EXCEL_APP_ARGUMENTS,
        }
    elif ProcessNames.PPT in process_name:
        desired_caps = {
            "app": DriversAndAppsPaths.PPT_APP_PATH,
            "appArguments": DriversAndAppsPaths.PPT_APP_ARGUMENTS,
        }
    elif ProcessNames.WORD in process_name:
        desired_caps = {
            "app": DriversAndAppsPaths.WORD_APP_PATH,
            "appArguments": DriversAndAppsPaths.WORD_APP_ARGUMENTS,
        }
    elif ProcessNames.PHOTOS_APP in process_name:
        desired_caps = {"app": DriversAndAppsPaths.WINDOWS_PHOTOS_APP_ID}
    elif ProcessNames.WIN_GAME_BAR_APP in process_name:
        return None, time.time()
    else:
        raise NotImplementedError("other windows native app not supported yet.")
    from appium import webdriver as appium_webdriver

    time_before_run_use_case = time.time()
    driver = appium_webdriver.Remote(
        command_executor=UseCasesURLS.WIN_APP_DRIVER_URL_PATH,
        desired_capabilities=desired_caps,
    )
    return driver, time_before_run_use_case


def get_linux_native_app_driver(process_name):
    """Function used to determined which native app driver should be used
    for Linux OS type.

    Args:
        process_name: process_name of use case.

    Returns:
        Linux native app driver.

    Raises:
        NotImplementedError because it is not supported yet.
    """
    raise NotImplementedError("get_linux_native_app_driver not supported yet.")


def get_random_user_agent():
    """Function used to get user agent with random uuid based on
    platform type for browser use cases.

    Returns:
        User agent with random uuid.
    """
    LOGGER.info("executing get_random_user_agent method.")
    if platform.system() == ProcessNames.WINDOWS_PLATFORM_TYPE:
        user_agent = UserAgent.WINDOWS_USER_AGENT
    else:
        user_agent = UserAgent.LINUX_USER_AGENT
    user_agent += " " + str(uuid.uuid4())
    return user_agent


def print_time_results(result, time_before_run_use_case):
    """Function to print time results collected during running the use case.

    Args:
        result: The results returned by executing the use case.
        time_before_run_use_case: the time before run the use case.
    """
    LOGGER.info("executing print_time_results method.")
    time_after_run_use_case = time.time()
    period_of_login_time = 0
    human_delay = 0
    if result is not None and isinstance(result, dict):
        if result.get("period_of_login_time") is not None:
            period_of_login_time = result.get("period_of_login_time")
        if result.get("human_delay") is not None:
            human_delay = result.get("human_delay")
    result_time = time_after_run_use_case - time_before_run_use_case
    result_time += human_delay
    result_time -= period_of_login_time
    execution_times = time_benchmark.execution_times
    if len(execution_times) > 0:
        for exec_time in execution_times:
            RESULTS_LOGGER.info(
                f"Time for execute function = {exec_time[0]} = {exec_time[1]}"
            )
    RESULTS_LOGGER.info(f"The time for running this use case = {result_time}")
    RESULTS_LOGGER.info(f"{15 * '-'}**********{15 * '-'}")


def close_used_linux_browser_windows(driver):
    """Function to close used linux browser windows.

    Args:
        driver: The WebDriver instance used to interact with
                    the browser.
    """
    for w in driver.window_handles:
        if w not in old_windows:
            try:
                driver.switch_to.window(w)
                driver.close()
            except NoSuchWindowException:
                pass


def setup(
    is_browsing: bool = False,
    process_name: tuple = None,
    is_execution_part: bool = False,
):
    """
    Helper function that's responsible for determining the driver
    to use corresponding if it is chrome or edge or native platform
    application on both environment windows and linux.

    Args:
        is_browsing: It should be True when the test will run on browser.
        process_name: Tuple of process name.
        is_execution_part: Indicates if the decorator ran for execute
            part to allow calculation of it's time
    """

    def decorator(func):
        def wrapper():
            LOGGER.info(
                "start executing setup decorator with "
                f"is_browsing = {is_browsing} and "
                f"process_name = {process_name} and "
                f"is_execution_part = {is_execution_part}."
            )
            if process_name is None or len(process_name) == 0:
                raise BenchmarkingException("process_name should not be empty.")
            if is_browsing:
                # browser app
                driver, time_before_run_use_case = get_browser_web_driver(
                    process_name
                )
            else:
                # native app
                driver, time_before_run_use_case = get_native_app_driver(
                    process_name
                )
            # invoke the use case
            try:
                if driver is not None:
                    result = func(driver)
                else:
                    result = func()
            finally:
                if driver is not None:
                    if platform.system() != ProcessNames.WINDOWS_PLATFORM_TYPE:
                        close_used_linux_browser_windows(driver)
                    else:
                        driver.quit()
            if is_execution_part:
                print_time_results(result, time_before_run_use_case)
            LOGGER.info("ending of execute setup decorator.")
            return result

        return wrapper

    return decorator
