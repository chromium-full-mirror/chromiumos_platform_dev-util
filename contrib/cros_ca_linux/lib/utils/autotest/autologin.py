# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Does the CrOS auto login. Borrowed from autotest."""

import sys
from typing import List


# pylint: disable=wrong-import-position

# Import autotest client
client_dir = "/usr/local/autotest/bin"
sys.path.insert(0, client_dir)
# "common" does all the autotest lib setup.
import common  # pylint: disable=unused-import


sys.path.pop(0)

from autotest_lib.client.common_lib.cros import chrome


# pylint: enable=wrong-import-position


def autologin(
    arc: bool = False,
    arc_timeout: int = None,
    no_arc_syncs: bool = False,
    disable_arc_cpu_restriction: bool = False,
    username: str = None,
    password: str = None,
    enable_default_apps: bool = False,
    dont_override_profile: bool = False,
    no_startup_window: bool = False,
    url: str = "",
    enable_features: List[str] = None,
    disable_features: List[str] = None,
    extra_args: List[str] = None,
):
    """ChromeOS auto login.

    Args:
        arc: Flag to enable ARC.
        arc_timeout: The timeout value waiting for ARC.
        no_arc_syncs: Flag to disable ARC syncs.
        disable_arc_cpu_restriction: Flag to disable ARC CPU restriction.
        username: User name. Do GAIA login if the user is given.
        password: Password for the user.
        enable_default_apps: Flag to enable default apps.
        dont_override_profile: Flag to disable profile overriding.
        no_startup_window: Don't open a Chrome window on startup.
        url: The URL to navigate to when starting up the Chrome.
        enable_features: Enable the given features.
        disable_features: Disable the given features.
        extra_args: Extra chrome args to pass to the Chrome process.
            For example:
            - pass in "--use-fake-ui-for-media-stream" to grant media
              permission for Google Meet.
            - pass in "--enable-benchmarking" to enable benchmark testing.
    """
    browser_args = []
    if no_startup_window:
        browser_args.append("--no-startup-window")
    if enable_features and len(enable_features) > 0:
        browser_args.append("--enable-features=%s" % ",".join(enable_features))
    if disable_features and len(disable_features) > 0:
        browser_args.append(
            "--disable-features=%s" % ",".join(disable_features)
        )
    if extra_args and len(extra_args) > 0:
        browser_args.extend(extra_args)

    # Avoid calling close() on the Chrome object; this keeps the session active.
    cr = chrome.Chrome(
        extra_browser_args=browser_args,
        arc_mode=("enabled" if arc else None),
        arc_timeout=arc_timeout,
        disable_arc_cpu_restriction=disable_arc_cpu_restriction,
        disable_app_sync=no_arc_syncs,
        disable_play_auto_install=no_arc_syncs,
        username=username,
        password=(password if username else None),
        gaia_login=(username is not None),
        disable_default_apps=(not enable_default_apps),
        dont_override_profile=dont_override_profile,
    )
    if url:
        tab = cr.browser.tabs[0]
        tab.Navigate(url)
