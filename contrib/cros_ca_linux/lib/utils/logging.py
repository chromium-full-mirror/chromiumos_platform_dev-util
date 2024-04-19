# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Utils to manage logging."""

import logging

import lib.definition as df


LOG_FILE = df.PROJECT_LOG_DIR / "full.log"

# Custom formatter
formatter = logging.Formatter("%(asctime)s [%(levelname)s]: %(message)s")

# Stream handler for stdout
stream_handler = logging.StreamHandler()
stream_handler.setFormatter(formatter)

if not df.PROJECT_LOG_DIR.exists():
    df.PROJECT_LOG_DIR.mkdir(parents=True)
# File handler for a log file
# TODO: Set file rotation.
file_handler = logging.FileHandler(LOG_FILE)
file_handler.setFormatter(formatter)

# Configure the logger
logger = logging.getLogger("cros_ca")
logger.addHandler(stream_handler)
logger.addHandler(file_handler)
# TODO: control level from a argument
logger.setLevel(logging.DEBUG)
