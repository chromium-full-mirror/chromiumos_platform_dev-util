#!/usr/bin/env python3
# -*- coding: utf-8 -*-

# Copyright 2021 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Main file for pacman utility"""

import logging
import sys

logger = logging.getLogger(__name__)


def main():
    logger.error(
        "DEPRECATION ERROR: PACman project has been deprecated since June "
        "2024 (crrev.com/c/5613551) and have now been fully removed. Please "
        "migrate the usage to successor Pacina, available here: \n"
        "https://chromium.googlesource.com/chromiumos/platform/dev-util/+/refs/heads/main/contrib/power_measurement/pacina/"  # noqa
    )
    sys.exit(1)


if __name__ == "__main__":
    main()
