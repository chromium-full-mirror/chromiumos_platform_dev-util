# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Utils to manage logging."""

import logging

import cros_ca_lib.definition as df


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

# Disable the default logging that could be used by other libraries.
root_logger = logging.getLogger()
root_logger.addHandler(logging.NullHandler())

# Configure the logger for CrOS CA.
logger = logging.getLogger("cros_ca")
logger.propagate = False
logger.addHandler(stream_handler)
logger.addHandler(file_handler)
# Default level is INFO. But can be changed by config.
# See lib.config module.
logger.setLevel(logging.INFO)
