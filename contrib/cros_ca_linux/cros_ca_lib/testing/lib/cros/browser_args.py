# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""CrOS browser args for testing."""

from typing import List

from cros_ca_lib.host import base_host


def default_args(option: base_host.LoginOption) -> None:
    """Default browser args"""

    chrome_args = [
        # Let Chrome choose its own debugging port.
        "--remote-debugging-port=0",
        # Disable redirection of Chrome logging into cryptohome.
        "--disable-logging-redirect",
        # Disable system startup sound.
        "--ash-disable-system-sounds",
        # Disable first-login educational nudges.
        "--ash-no-nudges",
        # Allow Chrome to use the Chrome Automation API.
        "--enable-experimental-extension-apis",
        # Redirect libassistant logging to /var/log/chrome/.
        "--redirect-libassistant-logging",
        # Prevent showing up offer pages, e.g. google.com/chromebooks.
        "--no-first-run",
        # Don't try to detect and ignore unintended clicks.
        "--disable-input-event-activation-protection",
        # Disable In Product Help notifications and UI.
        # Pass additional parameters to enable.
        # See b/296141011 for details.
        "--propagate-iph-for-testing",
        # Force rendering to run in the sRGB color space.
        # See b/221643955 for details.
        "--force-raster-color-profile=srgb",
        # Used when full session restore is disabled.
        "--hide-crash-restore-bubble",
    ]
    if option.chrome_args is None:
        option.chrome_args = []
    option.chrome_args.extend(chrome_args)


def add_args(opt: base_host.LoginOption, args: List[str]) -> None:
    """Add args to browser"""

    if not args or len(args) == 0:
        return
    if not opt.chrome_args:
        opt.chrome_args = []
    opt.chrome_args.extend(args)
