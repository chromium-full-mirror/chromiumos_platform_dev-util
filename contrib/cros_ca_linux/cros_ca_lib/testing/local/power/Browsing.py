# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Browsing test case."""

import datetime
import os
import time
from typing import Any, Dict, List

import cros_ca_lib.definition as df
from cros_ca_lib.host import base_host
from cros_ca_lib.metric import recorder as rd
from cros_ca_lib.metric import summary
from cros_ca_lib.metric.system import power
from cros_ca_lib.runner.post_processing import power as process_power
from cros_ca_lib.testing.lib import chrome_browser_test as cbt
from cros_ca_lib.utils.autotest import power_dashboard as dashboard
import requests


def load_config(config_path="") -> Dict[str, Any]:
    """Gets the configuration file from the specified url."""
    if config_path == "":
        return None
    response = requests.get(config_path, timeout=15)
    return response.json()


class BrowsingConfig:
    """Stores the variables used in Browsing test."""

    def __init__(
        self,
        loop_count: int,
        secs_per_page: int,
        secs_per_scroll: int,
        pages: List[str],
    ):
        self.loop_count = loop_count
        self.secs_per_page = secs_per_page
        self.secs_per_scroll = secs_per_scroll
        self.pages = pages

    def __str__(self) -> str:
        return (
            f"[loop_count={self.loop_count}, "
            f"secs_per_page={self.secs_per_page}, "
            f"secs_per_scroll={self.secs_per_scroll}, "
            f"pages={self.pages}]"
        )


LOOP_COUNT = 1
SECS_PER_PAGE = 60
SECS_PER_SCROLL = 5
SCROLL_AMOUNT = 600
PAGES = ["apple", "bbc", "google"]


def to_browsing_config(json_data: Dict[str, Any]) -> BrowsingConfig:
    """Extracts variables from the configuration file."""
    loop_count = (
        json_data.get("timing_data", {}).get("loop_count", LOOP_COUNT)
        if json_data is not None
        else LOOP_COUNT
    )
    secs_per_page = (
        json_data.get("timing_data", {}).get("secs_per_page", SECS_PER_PAGE)
        if json_data is not None
        else SECS_PER_PAGE
    )
    secs_per_scroll = (
        json_data.get("timing_data", {}).get("secs_per_scroll", SECS_PER_PAGE)
        if json_data is not None
        else SECS_PER_SCROLL
    )
    pages = (
        json_data.get("url_data", {}).get("pages", PAGES)
        if json_data is not None
        else PAGES
    )
    return BrowsingConfig(loop_count, secs_per_page, secs_per_scroll, pages)


class Browsing(cbt.ChromeBrowserTest):
    """Browsing does the web browsing test."""

    required_vars = {
        f"{df.VAR_GAIA_ACCOUNT}": "The test account",
        f"{df.VAR_GAIA_PASSWORD}": "The password of the test account",
    }

    def _login(self) -> None:
        if os.name != "nt":
            # Only CrOS needs login
            option = base_host.LoginOption()
            option.username = self.var(df.VAR_GAIA_ACCOUNT)
            option.password = self.var(df.VAR_GAIA_PASSWORD)
            option.arc = True
            self.dut.auto_login(option)

    def _run(self) -> None:
        self._login()
        self.dut.set_volume(0)

        data = load_config(
            config_path=(
                "https://storage.googleapis.com/chromiumos-test-assets-public"
                "/power_LoadTest/v2_config/browsing.json"
            )
        )
        config = to_browsing_config(data)
        print(f"config: {config}")

        recorder = rd.TimerRecorder(
            self.test_name,
            summary.ListSummary(power.BatteryRemainingCapacity),
            summary.ListSummary(power.BatteryDischargeRate),
        )
        try:
            recorder.start()
            self._run_test(config)
        finally:
            recorder.stop()
        recorder.save(self.test_dir)

        process_power.calculate_battery_life(
            self.metric_file, self.test_name + ".t"
        )
        dashboard.save_html(
            self.metric_file,
            self.test_name + ".t",
            self.test_dir,
            ["minutes_battery_life"],
        )

    def _run_test(self, config: BrowsingConfig) -> None:
        driver = self.get_webdriver()

        base_url = (
            "https://storage.googleapis.com/chromiumos-test-assets-public"
            "/power_LoadTest/2023-08-09/{page}.html"
        )
        sites = [base_url.format(page=page) for page in config.pages]

        driver.get("about:blank")
        for i in range(0, config.loop_count):
            loop_name = f"loop_name_{i}"
            print(loop_name)
            for site in sites:
                start_time = datetime.datetime.now()
                driver.get(site)
                print(f"Navigate to {site}")
                scroll_amount = SCROLL_AMOUNT
                for sec in range(
                    config.secs_per_scroll,
                    config.secs_per_page,
                    config.secs_per_scroll,
                ):
                    end_time = start_time + datetime.timedelta(seconds=sec)
                    current_time = datetime.datetime.now()
                    interval = (end_time - current_time).total_seconds()
                    if interval > 0:
                        time.sleep(interval)
                    driver.execute_script(
                        f"window.scrollTo(0, {scroll_amount})"
                    )
                    scroll_amount = -scroll_amount
                end_time = start_time + datetime.timedelta(
                    seconds=config.secs_per_page
                )
                current_time = datetime.datetime.now()
                interval = (end_time - current_time).total_seconds()
                if interval > 0:
                    time.sleep(interval)
        driver.quit()
