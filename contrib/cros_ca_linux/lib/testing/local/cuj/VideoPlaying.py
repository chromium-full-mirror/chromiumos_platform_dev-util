# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""VideoPlaying test case."""

# The following line disable the abstract function implementation warning.
# pylint: disable=W0223

import json
import os
import subprocess
import time

from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.wait import WebDriverWait

from lib.testing.lib import chrome_browser_test as cbt


test_duration = 20  # 20 minutes of test time
xperf = False  # If run xperf during the test


class VideoPlaying(cbt.ChromeBrowserTest):
    """VideoPlaying does the video playing test."""

    required_vars = {}
    optional_vars = dict(
        xperf=f"If run xperf during the test. Default is {xperf}",
        duration=f"Scheduled test duration. Default is {test_duration} minute",
    )

    # Common timeout values used by the test.
    button_click_wait_time = 0.25

    def extra_chrome_options(
        self, opt: webdriver.ChromeOptions
    ) -> webdriver.ChromeOptions:
        # TODO: excludeSwitches might not work in CrOS.
        if os.name == "nt":
            opt.add_experimental_option(
                "excludeSwitches", ["enable-automation"]
            )
        return opt

    def run(self):
        # Get the webdriver at the beginning of the test.
        driver = self.get_webdriver()
        wait = WebDriverWait(driver, 15)

        # Open histogram detail page for each of the metric being tracked
        # (same window, index = [1:5]
        self.open_histogram_tabs()

        driver.switch_to.new_window("window")
        driver.get("https://crosvideo.appspot.com/?resolution=1080&loop=true")
        wait.until(EC.title_contains("CrosVideo Test"))
        cros_video_window_handle = driver.current_window_handle

        # Switch back to histogram pages and enable monitoring mode.
        self.enable_histogram_monitoring()

        # Start recording.
        self.enable_performance()

        driver.switch_to.window(cros_video_window_handle)

        # capture metric snapshot after meet call effect is in place and stable
        self.get_histogram_snapshots("switching back to crosvideo")

        cycle_video = wait.until(
            EC.element_to_be_clickable((By.ID, "cycleVideo"))
        )
        cycle_video.click()

        mpd_list = wait.until(EC.element_to_be_clickable((By.ID, "mpdList")))
        mpd_list.click()

        manifest_option = wait.until(
            EC.presence_of_element_located(
                (
                    By.XPATH,
                    "//option[contains(text(), 'H264 DASH 60 FPS')]",
                )
            )
        )
        _ = wait.until(EC.element_to_be_clickable(manifest_option))
        manifest_option.click()
        time.sleep(self.button_click_wait_time)

        self.get_histogram_snapshots("before crosvideo playback")

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
                    "InstructionRetired,TotalCycles,"
                    + "UnhaltedCoreCycles,LLCReference,LLCMisses",
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

        end_time = time.time() + duration * 60
        # Keep playing the video for a period.
        counter = 0
        # take snapshot of battery
        # TODO: Didn't see the output in the test dir. Need debugging.
        # TODO: Use DUT APIs to get the power
        # report_path = self.test_dir / "battery-report-start.html"
        # subprocess.call(["powercfg", "/batteryreport",
        #                  f"/output \"{report_path}\""])

        while time.time() < end_time:
            start_video_playback_time = time.time()

            # start video playback
            load_stream = wait.until(
                EC.element_to_be_clickable((By.ID, "loadButton"))
            )
            load_stream.click()
            # time.sleep(button_click_wait_time)
            time.sleep(2.2)

            # start mouse click action
            for i in range(23):
                # Button mpd_list defined above
                if i % 2 == 0:
                    mpd_list.click()
                    time.sleep(1.1)
                    EC.presence_of_element_located(
                        (
                            By.XPATH,
                            "//option[contains(text(), 'H264 DASH 60 FPS')]",
                        )
                    )
                    _ = wait.until(EC.element_to_be_clickable(manifest_option))
                    manifest_option.click()
                    time.sleep(1.1)

                cycle_video = wait.until(
                    EC.element_to_be_clickable((By.ID, "cycleVideo"))
                )
                cycle_video.click()
                time.sleep(2.2)

            # stop video playback and capture decoded and dropped frames info
            video_playback_control = wait.until(
                EC.element_to_be_clickable((By.ID, "video"))
            )
            video_playback_control.click()
            time.sleep(1.1)
            stop_video_playback_time = time.time()
            elapsed_time = stop_video_playback_time - start_video_playback_time

            # capture dropped frame info from crosvideo
            decoded_frames_element = driver.find_element(
                By.ID, "decodedFramesDebug"
            )
            dropped_frames_element = driver.find_element(
                By.ID, "droppedFramesDebug"
            )
            self.logger.debug(
                "Duration: %.2f, counter: %d, DecodedFrames: %s, "
                + "DroppedFrames: %s",
                elapsed_time,
                counter,
                decoded_frames_element.text,
                dropped_frames_element.text,
            )
            counter = counter + 1

        # take snapshot of battery
        # report_path = self.test_dir / "battery-report-end.html"
        # TODO: Use DUT APIs to get the power
        # subprocess.call(["powercfg", "/batteryreport",
        #                 f"/output \"{report_path}\""])

        self.get_histogram_snapshots("after crosvideo playback")

        # report_path = (
        #     self.test_dir / "battery-report-after-metric-snapshot.html"
        # )
        # TODO: Use DUT APIs to get the power
        # subprocess.call(["powercfg", "/batteryreport",
        #                 f"/output \"{report_path}\""])

        # Close the CrosVideo page to generate FCP/LCP2.
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
            self.logger.info("tasklist output:\n", task_output.stdout)

            etl_path = self.test_dir / "test.etl"
            self.logger.info(f"xperf -d {etl_path}")
            # subprocess.call(["xperf", "-d", "test.etl"])
            xperf_output = subprocess.run(
                ["xperf", "-d", etl_path],
                shell=True,
                text=True,
                capture_output=True,
                check=True,
            )
            self.logger.info("xperf utput:\n", xperf_output.stdout)

        # TODO: decide the output format
        histogram_dict = self.get_historam_data()

        driver.quit()
        self.logger.info(f"Metrics: {histogram_dict}")
        # Save data to result.json file.
        # TODO: write common code for all tests to save results-chart.
        with open(
            self.test_dir / "results-chart.json",
            "w",
            encoding="utf-8",
        ) as f:
            f.write(json.dumps(histogram_dict))
