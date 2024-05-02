# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The chrome browser based test definition."""

# The following line disable the abstract function implementation warning.
# pylint: disable=W0223

import time
from typing import Any, Dict, Union

from cros_ca_lib.testing.lib import base_test
from selenium import webdriver
from selenium.common import exceptions
from selenium.webdriver.common.by import By
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.wait import WebDriverWait


DEFAULT_OPT = webdriver.ChromeOptions()
# TODO: Define the common options for all test.
DEFAULT_OPT.add_argument("--disable-infobars")
DEFAULT_OPT.add_argument("start-maximized")
DEFAULT_OPT.add_argument("--disable-extensions")
DEFAULT_OPT.add_argument("--disable-infobars")

# TODO: allow test to define its own metrics.
HISTOGRAM_MERICS = [
    "Graphics.Smoothness.PercentDroppedFrames3.AllSequences",
    "EventLatency.KeyPressed.TotalLatency",
    "EventLatency.MousePressed.TotalLatency",
    "PageLoad.PaintTiming.NavigationToLargestContentfulPaint2",
    "PageLoad.PaintTiming.NavigationToFirstContentfulPaint",
    "PageLoad.InteractiveTiming.InputDelay3",
    "PageLoad.InteractiveTiming.TimeToNextPaint",
    # pylint: disable=line-too-long
    "PageLoad.Experimental.NavigationTiming.NavigationStartToFirstResponseStart",
    # pylint: enable=line-too-long
    # "Graphics.Smoothness.Jank.AllSequences",
    "Graphics.Smoothness.Jank3.AllSequences",
    # "PageLoad.LayoutInstability.CumulativeShiftScore",
    # "Browser.MainThreadsCongestion",
    # "Browser.MainThreadsCongestion.RunningOnly",
    # "Media.DroppedFrameCount",
    # "Media.DroppedFrameCount2",
    # "Graphics.Smoothness.MaxStale.Video",
    # "Graphics.Smoothness.Stale.Video",
]


class ChromeBrowserTest(base_test.BaseRun):
    """Base class for Chrome browser tests using Selenium WebDriver."""

    # The test cases can override these values in their class definition.
    getHistogram_wait_time = 0.1
    tabSwitch_wait_time = 0.1  # Time to wait after switching to new tab.

    chromeOps: webdriver.ChromeOptions = DEFAULT_OPT
    # TODO: change this to a getter function and check if it is None
    #       before using.
    web_driver: webdriver.Chrome = None
    # Histogram metrics to track
    histogram_metrics = HISTOGRAM_MERICS
    histogram_tabs = {}

    def extra_chrome_options(
        self, opt: webdriver.ChromeOptions
    ) -> webdriver.ChromeOptions:
        return opt

    def get_webdriver(self) -> Union[webdriver.Chrome, webdriver.Remote]:
        if not self.web_driver:
            opts = self.extra_chrome_options(self.chromeOps)
            self.web_driver = self.dut.chrome_webdriver(opts)

        return self.web_driver

    def _get_histogram_snapshot(
        self,
        driver: Union[webdriver.Chrome, webdriver.Remote],
        histogram: str,
        use_delta: bool,
    ):
        result = None
        try:
            if use_delta:
                result = driver.execute_cdp_cmd(
                    "Browser.getHistogram", {"delta": True, "name": histogram}
                )
            else:
                result = driver.execute_cdp_cmd(
                    "Browser.getHistogram", {"name": histogram}
                )
        except exceptions.InvalidArgumentException:
            # "Cannot find histogram" error
            self.logger.info(
                "Browser.getHistogram %s: cannot find histogrqm", histogram
            )
        except Exception as e:
            # Other exceptions.
            self.logger.warning(
                "Failed to execute Browser.getHistogram %s: %s", histogram, e
            )

        return result

    def open_histogram_tabs(self):
        driver = self.web_driver
        wait = WebDriverWait(driver, 15)
        self.logger.info("Opening histogram tabs...")
        for metric in self.histogram_metrics:
            driver.switch_to.new_window("tab")
            driver.get("chrome://histograms/#" + metric)
            wait.until(EC.url_contains(metric))
            self.histogram_tabs[metric] = driver.current_window_handle
            # capture histogram
            histogramSnapshot = self._get_histogram_snapshot(
                driver, metric, False
            )
            time.sleep(self.getHistogram_wait_time)
            self.logger.info(
                "histogram snapshot (open - full): %s -- %s",
                metric,
                histogramSnapshot,
            )

    def enable_histogram_monitoring(self):
        driver = self.web_driver
        self.logger.info("Enabling histogram monitoring...")
        for metric in self.histogram_metrics:
            if metric not in self.histogram_tabs:
                continue
            window_handle = self.histogram_tabs[metric]
            driver.switch_to.window(window_handle)
            time.sleep(self.tabSwitch_wait_time)
            # Switch to monitoring mode
            driver.find_element(By.ID, "enable_monitoring").click()
            time.sleep(self.getHistogram_wait_time)
            self.logger.info("Monitoring mode enabled %s", metric)
            histogramSnapshot = self._get_histogram_snapshot(
                driver, metric, True
            )
            time.sleep(self.getHistogram_wait_time)
            self.logger.info(
                "histogram snapshot (monitor - delta): %s -- %s",
                metric,
                histogramSnapshot,
            )
            histogramSnapshot = self._get_histogram_snapshot(
                driver, metric, False
            )
            time.sleep(self.getHistogram_wait_time)
            self.logger.info(
                "histogram snapshot (monitor - full): %s -- %s",
                metric,
                histogramSnapshot,
            )

    def get_histogram_snapshots(self, annotation: str):
        driver = self.web_driver
        for metric in self.histogram_metrics:
            histogramSnapshot = self._get_histogram_snapshot(
                driver, metric, True
            )
            time.sleep(self.getHistogram_wait_time)
            self.logger.info(
                "histogram snapshot (%s - delta): %s -- %s",
                annotation,
                metric,
                histogramSnapshot,
            )
            histogramSnapshot = self._get_histogram_snapshot(
                driver, metric, False
            )
            time.sleep(self.getHistogram_wait_time)
            self.logger.info(
                "histogram snapshot ({annotation} - full): %s -- %s",
                metric,
                histogramSnapshot,
            )

    def get_historam_data(self) -> Dict[str, Any]:
        # TODO: decide the output format
        histogram_dict = {}
        driver = self.web_driver
        self.logger.info("Get histogram data.")
        if not self.histogram_dir.exists():
            self.histogram_dir.mkdir(parents=True)

        for metric in self.histogram_metrics:
            if metric not in self.histogram_tabs:
                continue
            window_handle = self.histogram_tabs[metric]
            driver.switch_to.window(window_handle)
            time.sleep(self.tabSwitch_wait_time)

            # capture screenshot before monitoring mode disabled
            if not self.save_screenshot(f"stopping_{metric}"):
                self.logger.info(
                    "Failed to save a screenshot on DUT when stopping %s",
                    metric,
                )
            time.sleep(0.5)

            # capture histogram before monitoring mode disabled
            histogramSnapshot = self._get_histogram_snapshot(
                driver, metric, True
            )
            time.sleep(self.getHistogram_wait_time)
            self.logger.info(
                "histogram snapshot (before monitoring stopped - delta): "
                + "%s -- %s",
                metric,
                histogramSnapshot,
            )
            histogramSnapshot = self._get_histogram_snapshot(
                driver, metric, False
            )
            time.sleep(self.getHistogram_wait_time)
            self.logger.info(
                "histogram snapshot (before monitoring stopped - full): "
                + "%s -- %s",
                metric,
                histogramSnapshot,
            )

            # Stop monitoring mode
            driver.find_element(By.ID, "stop").click()
            time.sleep(0.25)
            self.logger.info("monitoring mode stopped: %s", metric)
            if not self.save_screenshot(f"stopped_{metric}"):
                self.logger.info(
                    "Failed to save a screenshot on DUT after stopping %s",
                    metric,
                )
            time.sleep(0.5)

            # capture histogram after monitoring mode disabled
            histogramSnapshot = self._get_histogram_snapshot(
                driver, metric, True
            )
            time.sleep(self.getHistogram_wait_time)
            self.logger.info(
                "histogram snapshot (monitoring stopped - delta): %s -- %s",
                metric,
                histogramSnapshot,
            )
            histogramSnapshot = self._get_histogram_snapshot(
                driver, metric, False
            )
            time.sleep(self.getHistogram_wait_time)
            self.logger.info(
                "histogram snapshot (monitoring stopped - full): %s -- %s",
                metric,
                histogramSnapshot,
            )

            # Exand the histograms and capture the samples, mean and the body
            histograms = driver.find_element(By.ID, "histograms")
            headers = histograms.find_elements(
                By.CLASS_NAME, "histogram-header"
            )
            samples, mean_value = "-1", "-1"
            try:
                for i, header in enumerate(headers):
                    histogram_name = header.get_attribute("histogram-name")

                    # Click the header text to expand the histogram details.
                    header.find_element(By.CLASS_NAME, "expand").click()
                    time.sleep(1)
                    # calculate mean and capture samples size
                    header_text = header.find_element(
                        By.CLASS_NAME, "histogram-header-text"
                    ).text
                    word = header_text.split(" ")
                    samples, mean_value = word[3], word[7]
                    self.logger.info(
                        "%s mean: %s samples: %s",
                        histogram_name,
                        mean_value,
                        samples,
                    )
                    self.save_screenshot(f"expend_{metric}")
                    time.sleep(0.1)
                    # print histogram body
                    histogram_body = histograms.find_elements(
                        By.XPATH, "//div[@class='histogram-body']//p"
                    )[i]
                    self.logger.info(
                        "histogram body: \n%s", histogram_body.text
                    )
            except Exception:
                self.logger.warning("Failed to get data from %s", metric)

            histogram_dict[metric] = {"sample": samples, "mean": mean_value}
            histogramSnapshot = self._get_histogram_snapshot(
                driver, metric, False
            )
            time.sleep(self.getHistogram_wait_time)
            self.logger.info(
                "histogram snapshot (get data - full): %s -- %s",
                metric,
                histogramSnapshot,
            )
        return histogram_dict

    def enable_performance(self):
        self.web_driver.execute_cdp_cmd("Performance.enable", {})

    def save_screenshot(self, suffix: str) -> bool:
        if not self.screenshots_dir.exists():
            self.screenshots_dir.mkdir(parents=True)
        file = self.screenshots_dir / f"screenshot-{suffix}.png"
        return self.web_driver.save_screenshot(file)
