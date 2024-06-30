# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""GoogleMeet test case."""

import json
import os
import shutil
import subprocess
import time

import cros_ca_lib.definition as df
from cros_ca_lib.host import base_host
from cros_ca_lib.metric import recorder as rd
from cros_ca_lib.testing.lib import chrome_browser_test as cbt
from cros_ca_lib.testing.lib.cros import browser_args
from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.wait import WebDriverWait


meet_ttl = 45  # Meet length in minutes
test_duration = 20  # 20 minutes of test time
bot_video_file_path = "jamboard_three_close_video_hd.1280_720.yuv"
xperf = False  # If run xperf during the test


class GoogleMeet(cbt.ChromeBrowserTest):
    """Test case for Google Meet."""

    required_vars = {
        "typing_delay": "Key typing delay in seconds",
        f"{df.VAR_GAIA_ACCOUNT}": "The test account",
        f"{df.VAR_GAIA_PASSWORD}": "The password of the test account",
        "meet_code": "Google Meet code to join.",
    }
    optional_vars = dict(
        xperf=f"If run xperf during the test. Default is {xperf}",
        duration=f"Scheduled test duration. Default is {test_duration} minute",
    )
    parameters = {
        # Total participant number in the meet
        "": 2,  # Default is 2
        "2p": 2,
        "16p": 16,
    }

    # Common timeout values used by the test.
    send_keys_wait_time = 5
    button_click_wait_time = 0.25
    debug_wait_time = 0.25

    def extra_chrome_options(
        self, opt: webdriver.ChromeOptions
    ) -> webdriver.ChromeOptions:
        if os.name != "nt":
            return opt
        # TODO: excludeSwitches and prefs might not work in CrOS.
        opt.add_experimental_option("excludeSwitches", ["enable-automation"])
        # Enable mic and camera for Google Meet.
        # TODO: Find how to do it in CrOS
        opt.add_experimental_option(
            "prefs",
            {
                "profile.default_content_setting_values.media_stream_mic": 1,
                "profile.default_content_setting_values.media_stream_camera": 1,
                "profile.default_content_setting_values.geolocation": 1,
                "profile.default_content_setting_values.notifications": 1,
            },
        )
        opt.add_argument("--enable-benchmarking")

        return opt

    def _login(self) -> None:
        if os.name == "nt":
            return
        # Only CrOS needs login
        option = base_host.LoginOption()
        option.username = self.var(df.VAR_GAIA_ACCOUNT)
        option.password = self.var(df.VAR_GAIA_PASSWORD)
        option.arc = True
        browser_args.default_args(option)
        browser_args.add_args(
            option,
            [
                "--enable-benchmarking",
                # Disable media permission request
                "--use-fake-ui-for-media-stream",
                # Disable site notification
                "--disable-notifications",
            ],
        )
        self.dut.auto_login(option)

    def _run(self) -> None:
        self._login()
        self.dut.set_volume(0)

        recorder = rd.SimpleRecorder(self.test_name)
        try:
            recorder.start()
            self._run_test(recorder)
        finally:
            recorder.stop()
        recorder.save(self.test_dir)

    def _run_test(self, recorder: rd.Recorder) -> None:
        # Get the webdriver at the beginning of the test.
        driver = self.get_webdriver()
        wait = WebDriverWait(driver, 15)

        if os.name == "nt":
            # Only do this for Windows. For CrOS, the DUT has been signed in
            # with GAIA login.
            # Sign in with the Google account (index = 0)
            driver.get("https://accounts.google.com/ServiceLogin")
            driver.find_element(By.TAG_NAME, "input").send_keys(
                self.var(df.VAR_GAIA_ACCOUNT)
            )
            driver.find_element(By.XPATH, '//span[text()="Next"]').find_element(
                By.XPATH, "./.."
            ).click()
            _ = wait.until(
                EC.presence_of_element_located(
                    (By.XPATH, '//span[text()="Welcome"]')
                )
            )
            submit = driver.find_element(By.ID, "passwordNext")
            _ = wait.until(EC.element_to_be_clickable(submit))
            driver.find_element(By.NAME, "Passwd").send_keys(
                self.var(df.VAR_GAIA_PASSWORD)
            )
            submit.click()
            # time.sleep(send_keys_wait_time)
            time.sleep(0.1)

        # Open WebRTC Internals page (index = 0)
        driver.get("chrome://webrtc-internals")
        wait.until(EC.title_contains("WebRTC Internals"))
        webrtc_window_handle = driver.current_window_handle

        # Open histogram detail page for each of the metric being tracked (same
        # window, index = [1:5]
        self.open_histogram_tabs()
        # Switch back to webrtc tab
        driver.switch_to.window(webrtc_window_handle)
        time.sleep(self.tabSwitch_wait_time)

        # Switch back to histogram pages and enable monitoring mode.
        self.enable_histogram_monitoring()

        # Start recording.
        self.enable_performance()

        # Join the meeting (window_1, index = 6)
        driver.switch_to.new_window("window")
        driver.get(f"https://meet.google.com/{self.var('meet_code')}")
        wait.until(EC.title_contains("Meet"))
        try:
            time.sleep(10)
            driver.find_element(
                By.CSS_SELECTOR, 'button[jsname="EszDEe"][aria-label="Got it"]'
            ).click()
        except Exception:
            self.logger.info("No pop up window in Google Meet page, skip it")

        # Set full screen to match chromeos
        self.logger.info("Enter full screen")
        more_options = wait.until(
            EC.element_to_be_clickable(
                (
                    By.CSS_SELECTOR,
                    "button[jsname='NakZHc'][aria-label='More options']",
                )
            )
        )
        more_options.click()

        full_screen = driver.find_element(
            By.CSS_SELECTOR,
            "li[delegate-controller='pc452']",
        )
        full_screen.click()

        self.logger.info("Turn off camera")
        driver.implicitly_wait(2)
        # Turn off camera
        camera_button = driver.find_element(
            By.CSS_SELECTOR, 'button[aria-label="Turn off camera"]'
        )
        camera_button.click()

        chat_button = wait.until(
            EC.element_to_be_clickable(
                (
                    By.CSS_SELECTOR,
                    'button[aria-label="Chat with everyone"]',
                )
            )
        )
        chat_button.click()

        text_area = wait.until(
            EC.element_to_be_clickable(
                (
                    By.CSS_SELECTOR,
                    'textarea[aria-label="Send a message"]'
                    + '[placeholder="Send a message"]',
                )
            )
        )
        text_area.click()

        # capture metric snapshot after meet call effect is in place and stable
        self.get_histogram_snapshots("meet effect stablized and before typing")

#        sentences = [
#            "Wear two pairs of thick socks. Wear two pairs of thick socks."
#            + " Wear a pair of thick socks",
#           "Running on a cold and cloudy day Running on a cold rainy day"
#            + " Running on a cold rainy day",
#        ]

        sentences = [
            "Wear two pairs of thick socks "
            "to run on a freezing cold day"
        ]

        counter = 0
        self.logger.info("Typing started.")

        run_xperf = xperf
        try:
            value = self.var("xperf")
            if value.lower in ["y", "yes", "t", "true"]:
                run_xperf = True
        except Exception:
            # Ignore if var is not given.
            pass

        if run_xperf:
            self.logger.info(
                "Run: xperf -on DiagEasy+CSWITCH  "
                + "-pmc InstructionRetired,TotalCycles CSWITCH strict"
            )
            subprocess.call(
                [
                    "xperf",
                    "-on",
                    "PROC_THREAD+LOADER+CSWITCH",
                    "-pmc",
                    "InstructionRetired,TotalCycles,UnhaltedCoreCycles,"
                    + "LLCReference,LLCMisses",
                    "CSWITCH",
                    "strict",
                ]
            )

        # start typing
        duration = test_duration
        try:
            value = self.var("duration")
            duration = float(value)
        except Exception:
            # Ignore if var is not given.
            pass

        typing_delay = float(self.var("typing_delay"))

        end_time = time.time() + duration * 60
        while time.time() < end_time:
            strCounter = str(counter)

            if counter <= 9:
                msgLabel = "00" + strCounter
            elif counter <= 99:
                msgLabel = "0" + strCounter
            else:
                msgLabel = strCounter
            # attach counter to the message
            msgLabel = " " + msgLabel + "_w_count"
            for sentence in sentences:
                for c in sentence:
                    text_area.send_keys(c)
                    time.sleep(typing_delay)
                for c in msgLabel:
                    text_area.send_keys(c)
                    time.sleep(typing_delay)
                delaybeforeEnter = 1 - typing_delay
                time.sleep(delaybeforeEnter)  # 1s gap between each message.
                text_area.send_keys(Keys.ENTER)
            counter += 1
        # capture histogram immeditely after typing
        self.get_histogram_snapshots("after typing")

        self.logger.info(
            "Typing stopped. Final counter value after typing: %s", counter
        )

        if not self.save_screenshot("after_typing"):
            self.logger.warning(
                "Failed to save a screenshot on DUT: after_typing"
            )

        driver.close()

        if run_xperf:
            # task_output = subprocess.check_output(["tasklist"])
            # cmd = "tasklist"
            task_output = subprocess.run(
                ["tasklist"],
                shell=True,
                text=True,
                capture_output=True,
                check=True,
            )
            self.logger.info("tasklist output:\n%s", task_output.stdout)

            etl_path = self.test_dir / "test.etl"
            self.logger.info("xperf -d %s", etl_path)
            # subprocess.call(["xperf", "-d", "test.etl"])
            xperf_output = subprocess.run(
                ["xperf", "-d", etl_path],
                shell=True,
                text=True,
                capture_output=True,
                check=True,
            )
            self.logger.info("xperf utput:\n%s", xperf_output.stdout)

        # If the dump file exists, remove it.
        file_path = os.path.join(
            self.dut.get_download_path(), "webrtc_internals_dump.txt"
        )
        if os.path.exists(file_path):
            os.remove(file_path)
        driver.switch_to.window(webrtc_window_handle)
        time.sleep(self.debug_wait_time)
        wait.until(EC.title_contains("WebRTC Internals"))
        time.sleep(self.debug_wait_time)
        driver.find_element(
            By.CSS_SELECTOR, "details.peer-connection-dump-root"
        ).click()
        driver.find_element(By.XPATH, '//div[@id="dump"]//button').click()
        time.sleep(self.button_click_wait_time)

        recorder.add_histograms(self.get_historam_data())
        driver.quit()

        # Add the data of dump file to result.
        with open(file_path, "r", encoding="utf-8") as f:
            webrtc_dump = json.load(f)
            recorder.add_result({"webrtc": webrtc_dump})

        dst = os.path.join(self.histogram_dir, "webrtc_internals_dump.txt")
        shutil.move(file_path, dst)
