# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The chrome browser based test definition."""

# The following line disable the abstract function implementation warning.
# pylint: disable=W0223

import copy
from pathlib import Path
import time
from typing import Any, Dict, List, Union

from cros_ca_lib.host import base_host
from cros_ca_lib.metric import summary
from cros_ca_lib.testing.lib import base_test
from cros_ca_lib.utils import logging as lg
from selenium import webdriver
from selenium.common import exceptions
from selenium.webdriver.common import by
from selenium.webdriver.remote import webelement
from selenium.webdriver.support import expected_conditions as ec
from selenium.webdriver.support import wait


DEFAULT_OPT = webdriver.ChromeOptions()
# TODO: Define the common options for all test.
DEFAULT_OPT.add_argument("--disable-infobars")
DEFAULT_OPT.add_argument("start-maximized")
DEFAULT_OPT.add_argument("--disable-extensions")
DEFAULT_OPT.add_argument("--disable-infobars")


class ChromeBrowserTest(base_test.BaseRun):
    """Base class for Chrome browser tests using Selenium WebDriver."""

    # The test cases can override these values in their class definition.
    getHistogram_wait_time = 0.1
    tabSwitch_wait_time = 0.1  # Time to wait after switching to new tab.

    chromeOps: webdriver.ChromeOptions = DEFAULT_OPT
    # TODO: change this to a getter function and check if it is None
    #       before using.
    web_driver: webdriver.Chrome = None
    histogram_tabs = {}

    def __init__(
        self,
        dut: base_host.BaseHost,
        param: str,
        variables: Dict[str, Any],
        test_dir: Path,
        logger: lg.logging.Logger,
    ) -> None:
        super().__init__(dut, param, variables, test_dir, logger)
        self._metrics: Dict[str, summary.Histogram] = {}
        self._init_metrics()

    def _init_metrics(self) -> None:
        """Override to add other metrics for specific test case requirements."""
        self._add_metric(
            "Graphics.Smoothness.PercentDroppedFrames3.AllSequences",
            units="percent",
        )
        self._add_metric("EventLatency.KeyPressed.TotalLatency")
        self._add_metric("EventLatency.MousePressed.TotalLatency")
        self._add_metric(
            "PageLoad.PaintTiming.NavigationToLargestContentfulPaint2"
        )
        self._add_metric(
            "PageLoad.PaintTiming.NavigationToFirstContentfulPaint"
        )
        self._add_metric("PageLoad.InteractiveTiming.InputDelay3")
        self._add_metric("PageLoad.InteractiveTiming.TimeToNextPaint")
        self._add_metric(
            "PageLoad.Experimental.NavigationTiming"
            ".NavigationStartToFirstResponseStart"
        )
        # self._add_metric("Graphics.Smoothness.Jank.AllSequences")
        self._add_metric("Graphics.Smoothness.Jank3.AllSequences")
        # self._add_metric("PageLoad.LayoutInstability.CumulativeShiftScore")
        # self._add_metric("Browser.MainThreadsCongestion")
        # self._add_metric("Browser.MainThreadsCongestion.RunningOnly")
        # self._add_metric("Media.DroppedFrameCount")
        # self._add_metric("Media.DroppedFrameCount2")
        # self._add_metric("Graphics.Smoothness.MaxStale.Video")
        # self._add_metric("Graphics.Smoothness.Stale.Video")

    def _add_metric(
        self,
        name: str,
        histogram_type: str = "average",
        units: str = "ms",
        improvement_direction: str = "down",
    ) -> None:
        data = summary.HistogramValue(units, improvement_direction)
        self._metrics[name] = summary.Histogram(name, histogram_type, data)

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
        wd_wait = wait.WebDriverWait(driver, 15)
        self.logger.info("Opening histogram tabs...")
        for metric in self._metrics:
            driver.switch_to.new_window("tab")
            driver.get("chrome://histograms/#" + metric)
            wd_wait.until(ec.url_contains(metric))
            self.histogram_tabs[metric] = driver.current_window_handle
            # capture histogram
            histogram_snapshot = self._get_histogram_snapshot(
                driver, metric, False
            )
            time.sleep(self.getHistogram_wait_time)
            self.logger.info(
                "histogram snapshot (open - full): %s -- %s",
                metric,
                histogram_snapshot,
            )

    def enable_histogram_monitoring(self):
        driver = self.web_driver
        self.logger.info("Enabling histogram monitoring...")
        for metric, window_handle in self.histogram_tabs.items():
            driver.switch_to.window(window_handle)
            time.sleep(self.tabSwitch_wait_time)
            # Switch to monitoring mode
            driver.find_element(by.By.ID, "enable_monitoring").click()
            time.sleep(self.getHistogram_wait_time)
            self.logger.info("Monitoring mode enabled %s", metric)
            histogram_snapshot = self._get_histogram_snapshot(
                driver, metric, True
            )
            time.sleep(self.getHistogram_wait_time)
            self.logger.info(
                "histogram snapshot (monitor - delta): %s -- %s",
                metric,
                histogram_snapshot,
            )
            histogram_snapshot = self._get_histogram_snapshot(
                driver, metric, False
            )
            time.sleep(self.getHistogram_wait_time)
            self.logger.info(
                "histogram snapshot (monitor - full): %s -- %s",
                metric,
                histogram_snapshot,
            )

    def get_histogram_snapshots(self, annotation: str):
        driver = self.web_driver
        for metric in self._metrics:
            histogram_snapshot = self._get_histogram_snapshot(
                driver, metric, True
            )
            time.sleep(self.getHistogram_wait_time)
            self.logger.info(
                "histogram snapshot (%s - delta): %s -- %s",
                annotation,
                metric,
                histogram_snapshot,
            )
            histogram_snapshot = self._get_histogram_snapshot(
                driver, metric, False
            )
            time.sleep(self.getHistogram_wait_time)
            self.logger.info(
                "histogram snapshot ({annotation} - full): %s -- %s",
                metric,
                histogram_snapshot,
            )

    def get_historam_data(self) -> List[summary.Histogram]:
        """Get the histograms of the test."""
        driver = self.web_driver
        self.logger.info("Get histogram data.")
        if not self.histogram_dir.exists():
            self.histogram_dir.mkdir(parents=True)

        results: List[summary.Histogram] = []
        for metric, window_handle in self.histogram_tabs.items():
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
            histogram_snapshot = self._get_histogram_snapshot(
                driver, metric, True
            )
            time.sleep(self.getHistogram_wait_time)
            self.logger.info(
                "histogram snapshot (before monitoring stopped - delta): "
                + "%s -- %s",
                metric,
                histogram_snapshot,
            )
            histogram_snapshot = self._get_histogram_snapshot(
                driver, metric, False
            )
            time.sleep(self.getHistogram_wait_time)
            self.logger.info(
                "histogram snapshot (before monitoring stopped - full): "
                + "%s -- %s",
                metric,
                histogram_snapshot,
            )

            # Stop monitoring mode
            driver.find_element(by.By.ID, "stop").click()
            time.sleep(0.25)
            self.logger.info("monitoring mode stopped: %s", metric)
            if not self.save_screenshot(f"stopped_{metric}"):
                self.logger.info(
                    "Failed to save a screenshot on DUT after stopping %s",
                    metric,
                )
            time.sleep(0.5)

            # capture histogram after monitoring mode disabled
            histogram_snapshot = self._get_histogram_snapshot(
                driver, metric, True
            )
            time.sleep(self.getHistogram_wait_time)
            self.logger.info(
                "histogram snapshot (monitoring stopped - delta): %s -- %s",
                metric,
                histogram_snapshot,
            )
            histogram_snapshot = self._get_histogram_snapshot(
                driver, metric, False
            )
            time.sleep(self.getHistogram_wait_time)
            self.logger.info(
                "histogram snapshot (monitoring stopped - full): %s -- %s",
                metric,
                histogram_snapshot,
            )

            results.extend(self._parse_histograms(metric))

            histogram_snapshot = self._get_histogram_snapshot(
                driver, metric, False
            )
            time.sleep(self.getHistogram_wait_time)
            self.logger.info(
                "histogram snapshot (get data - full): %s -- %s",
                metric,
                histogram_snapshot,
            )
        return results

    def _parse_histograms(self, metric: str) -> List[summary.Histogram]:
        """Expand the histograms and capture the samples, mean and the body."""
        histograms = self.web_driver.find_element(by.By.ID, "histograms")
        headers = histograms.find_elements(by.By.CLASS_NAME, "histogram-header")
        histogram_bodies = histograms.find_elements(
            by.By.XPATH, "//div[@class='histogram-body']//p"
        )
        results: List[summary.Histogram] = []
        try:
            for i, header in enumerate(headers):
                results.extend(self._parse_histogram(metric, header))
                time.sleep(0.1)
                # print histogram body
                self.logger.info(
                    "histogram body: \n%s", histogram_bodies[i].text
                )
        except Exception:
            self.logger.warning(
                "Failed to parse histograms of metric %s", metric
            )
        return results

    def _parse_histogram(
        self, metric: str, header: webelement.WebElement
    ) -> List[summary.Histogram]:
        name = header.get_attribute("histogram-name")

        # Click the header text to expand the histogram details.
        header.find_element(by.By.CLASS_NAME, "expand").click()
        time.sleep(1)
        # Capture mean and samples size
        header_text = header.find_element(
            by.By.CLASS_NAME, "histogram-header-text"
        ).text
        word = header_text.split(" ")
        samples, mean = word[3], word[7]
        self.logger.info("%s mean: %s samples: %s", name, mean, samples)
        self.save_screenshot(f"expand_{name}")
        return self._to_histograms(metric, name, float(mean), int(samples))

    def _to_histograms(
        self, metric: str, name: str, mean: float, samples: int
    ) -> List[summary.Histogram]:
        template = self._metrics[metric]
        mean_histogram = copy.deepcopy(template)
        mean_histogram.name = name
        mean_histogram.data.value = mean
        sample_histogram = copy.deepcopy(template)
        sample_histogram.name = name + ".samples"
        sample_histogram.data.units = "count"
        sample_histogram.data.value = samples
        return [mean_histogram, sample_histogram]

    def enable_performance(self):
        self.web_driver.execute_cdp_cmd("Performance.enable", {})

    def save_screenshot(self, suffix: str) -> bool:
        if not self.screenshots_dir.exists():
            self.screenshots_dir.mkdir(parents=True)
        file = self.screenshots_dir / f"screenshot-{suffix}.png"
        return self.web_driver.save_screenshot(file)
