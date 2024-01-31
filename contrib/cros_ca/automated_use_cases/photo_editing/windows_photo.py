#!/usr/bin/env python3
# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable= E1120, W1203
"""Windows Photos Editing/Crop native desktop use case."""
import os

from selenium.common.exceptions import NoSuchElementException, TimeoutException
from selenium.webdriver import ActionChains
from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.wait import WebDriverWait

from automated_use_cases.common_utility.doc_editing_utils import (
    open_win_app_driver,
    close_win_app_driver,
)
from automated_use_cases.common_utility.microsoft_documents_utils import (
    switch_to_native_app_window,
)
from benchmarking.benchmarking_test import (
    BenchmarkingTest,
)
from benchmarking.performance_decorator import performance
from benchmarking.platform_benchmarking import setup
from benchmarking.time_benchmark import time_measure
from common.constants import (
    ProcessNames,
    PageLoadConfig,
    DriversAndAppsPaths,
    LoggingConfig,
)

LOGGER = LoggingConfig.get_logger()

PROCESS_NAME = (ProcessNames.PHOTOS_SERVICE, ProcessNames.PHOTOS_APP)


def close_syncing_with_one_drive_tips(driver):
    """Helper Function to close syncing with one drive tips appear
    when opens Photos app for first usage.

    Args:
        driver: The Windows Desktop App Driver instance used to
            interact with the desktop app.
    """
    LOGGER.info("start close_syncing_with_one_drive_tips method.")
    try:
        driver.find_element_by_name("Next").click()
        driver.find_element_by_name("Go to Photos").click()
        WebDriverWait(driver, 5).until(
            lambda d: d.find_elements_by_accessibility_id(
                "AlternateCloseButton"
            )
        )[0].click()
    except (NoSuchElementException, TimeoutException):
        pass
    LOGGER.info("end of close_syncing_with_one_drive_tips method.")


@time_measure()
def select_test_photo(driver, wait, photo_name):
    """Function to select the test photo and open it.

    Args:
        driver: The Windows Desktop App Driver instance used to
            interact with the desktop app.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
        photo_name: photo name to be selected and opened.
    """
    LOGGER.info(
        "start select_test_photo method with " f"photo_name = {photo_name}."
    )
    photo = wait.until(EC.visibility_of_element_located((By.NAME, photo_name)))
    actions = ActionChains(driver)
    actions.move_to_element(photo).click(photo).send_keys(Keys.ENTER).perform()
    LOGGER.info("end of select_test_photo method.")


@time_measure()
def edit_crop_save_test_photo(driver, wait, new_photo_name):
    """Function used to edit, crop and save the test photo.

    This Function will perform the following:
    1. Edit the photo.
    2. Cut the photo by square aspect ratio.
    3. Rotate clockwise the test photo four times,
    4. Flip horizontally the test photo.
    5. Save the test photo on same photo.
    6. Wait until new edited photo to be saved.

    Args:
        driver: The Windows Desktop App Driver instance used to
            interact with the desktop app.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
        new_photo_name: The new name of edited photo that will be saved.
    """
    LOGGER.info(
        "start edit_crop_save_test_photo method with "
        f"new_photo_name = {new_photo_name}."
    )
    # edit the image and make a cut for it by Square aspect ratio
    wait.until(
        EC.visibility_of_element_located((By.NAME, "Edit image"))
    ).click()
    switch_to_native_app_window(driver, wait, 2)
    wait.until(
        EC.visibility_of_element_located((By.NAME, "Free aspect ratio"))
    ).click()
    wait.until(
        EC.visibility_of_element_located((By.NAME, "Square aspect ratio"))
    ).click()
    wait.until(EC.visibility_of_element_located((By.NAME, "Done"))).click()
    # rotate the image in clockwise direction 4 times
    counter = 0
    while counter < 4:
        wait.until(
            EC.element_to_be_clickable((By.NAME, "Rotate clockwise"))
        ).click()
        counter += 1
    # flip the image horizontally
    wait.until(
        EC.visibility_of_element_located((By.NAME, "Flip image horizontally"))
    ).click()
    # save the image by save as button
    wait.until(
        EC.visibility_of_element_located(
            (By.NAME, "Press Enter for more save options")
        )
    ).click()
    wait.until(
        EC.visibility_of_element_located((By.NAME, "Save as copy"))
    ).click()
    file_name_input = wait.until(
        EC.visibility_of_element_located((By.NAME, "File name:"))
    )
    actions = ActionChains(driver)
    actions.move_to_element(file_name_input).send_keys(new_photo_name).perform()
    wait.until(EC.element_to_be_clickable((By.NAME, "Save"))).click()
    # wait image to be saved before close the windows
    wait.until(EC.visibility_of_element_located((By.NAME, "Edit image")))
    LOGGER.info("end of edit_crop_save_test_photo method.")


def close_current_window(driver, original_window_handle):
    """Function to close current window before close original window.

    Args:
        driver: The Windows Desktop App Driver instance used to
            interact with the desktop app.
        original_window_handle: Original or first window opened in Photos
            app to be closed after switching to it.
    """
    LOGGER.info(
        "start close_current_window method with "
        f"original_window_handle = {original_window_handle}."
    )
    driver.close()
    driver.switch_to.window(original_window_handle)
    LOGGER.info("end of close_current_window method.")


def check_resource_folder_existence(driver, wait, photo_dir_path):
    """Function to check if native_app_files folder exist in Photos app list.

    Args:
        driver: The Windows Desktop App Driver instance used to
            interact with the desktop app.
        wait: Explicit wait used while checking the expected conditions
            of the elements.
        photo_dir_path: Path of photo directory.
    """
    LOGGER.info(
        "start check_resource_folder_existence method with "
        f"photo_dir_path = {photo_dir_path}."
    )
    native_resource_folder = driver.find_elements_by_name("native_app_files")
    if len(native_resource_folder) > 0:
        native_resource_folder[0].click()
    else:
        wait.until(
            EC.visibility_of_element_located((By.NAME, "Folders"))
        ).click()
        driver.find_element_by_accessibility_id(
            "AddFolderTemplateButton"
        ).click()
        folder_input = wait.until(
            EC.visibility_of_element_located((By.NAME, "Folder:"))
        )
        actions = ActionChains(driver)
        actions.move_to_element(folder_input).send_keys(
            photo_dir_path
        ).perform()
        driver.find_element_by_accessibility_id("1").click()
        wait.until(
            (EC.visibility_of_element_located((By.NAME, "native_app_files")))
        ).click()
    LOGGER.info("end of check_resource_folder_existence method.")


class WindowsPhotosNativeDesktopTest(BenchmarkingTest):
    """Test class for Windows Photos Editing/Crop native desktop use case."""

    def __init__(self):
        self.photo_dir_path = os.path.abspath(
            DriversAndAppsPaths.TEST_PHOTO_URL_DIR_PATH
        )
        self.photo_name = DriversAndAppsPaths.TEST_PHOTO_NAME
        self.new_photo_name = "New" + self.photo_name
        self.new_photo_path = os.path.abspath(
            self.photo_dir_path + rf"\{self.new_photo_name}"
        )

    def set_up(self):
        """This method will perform coping the test photo to Pictures
        directory path in Windows platform."""
        LOGGER.info(f"starting set_up method in {self.__class__.__name__}")
        open_win_app_driver()
        LOGGER.info(f"ending of set_up method in {self.__class__.__name__}")

    def execute(self):
        LOGGER.info(f"start execute method in {self.__class__.__name__}")

        @performance(process_name=PROCESS_NAME)
        @setup(process_name=PROCESS_NAME, is_execution_part=True)
        def run(driver):
            """Run Windows Photos Editing/Crop native desktop use case.

            This function will perform the following:
            1. Open Windows Photos using the native Photos app.
            2. Select and open the test photo.
            4. Switch to the new test photo window.
            5. Edit, crop and save the new edited test photo in new file.
            6. Close current window before close original window.

            This function is decorated with the @setup decorator,
            configuring the driver for the native desktop app processes.

            Args:
                driver: The Windows Desktop App Driver instance used to
                    interact with the desktop app.
            """
            wait = WebDriverWait(driver, PageLoadConfig.WAIT_TIMEOUT)
            switch_to_native_app_window(driver, wait)
            close_syncing_with_one_drive_tips(driver)
            check_resource_folder_existence(driver, wait, self.photo_dir_path)
            select_test_photo(driver, wait, self.photo_name)
            original_window_handle = driver.current_window_handle
            switch_to_native_app_window(driver, wait, 2)
            edit_crop_save_test_photo(driver, wait, self.new_photo_path)
            close_current_window(driver, original_window_handle)

        run()
        LOGGER.info(f"end of execute method in {self.__class__.__name__}")

    def teardown(self):
        """This method will perform delete for test photos from Pictures
        directory path in Windows platform."""
        LOGGER.info(f"starting teardown method in {self.__class__.__name__}")
        close_win_app_driver()
        os.remove(self.new_photo_path)
        LOGGER.info(f"ending of teardown method in {self.__class__.__name__}")


if __name__ == "__main__":
    WindowsPhotosNativeDesktopTest().run_test()
