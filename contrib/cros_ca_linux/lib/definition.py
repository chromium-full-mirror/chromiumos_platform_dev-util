# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Project global definitions."""

from pathlib import Path


# ==== Path for project folders and files
PROJECT_LIB_DIR = Path(__file__).parent.resolve()
PROJECT_ROOT_DIR = PROJECT_LIB_DIR.parent
PROJECT_BIN_DIR = PROJECT_ROOT_DIR / "bin"
PROJECT_LOG_DIR = PROJECT_ROOT_DIR / "logs"
PROJECT_VENV_DIR = PROJECT_ROOT_DIR / "venv"  # virtual env dir

PORT_FORWARDING_PROG_PATH = str(PROJECT_BIN_DIR / "port_forwarding.py")
# Default SSH private key file.
SSH_PRIVATE_KEY_FILE = str(PROJECT_LIB_DIR / "host/ssh/ssh_keys/testing_rsa")

# Folders that need to be push to the DUT
SRC_SYNC_DIRS = ["bin", "lib", "README.md", "requirements.txt"]

# === Path for test execution folders and files
TEST_HOST_ROOT_DIR = Path(r"/tmp/cros_ca")
TEST_HOST_RESULTS_DIR = TEST_HOST_ROOT_DIR / "results"

TEST_WIN_ROOT_DIR = Path(r"C:/cros_ca")
TEST_WIN_SRC_DIR = TEST_WIN_ROOT_DIR / "src"
TEST_WIN_VENV_DIR = TEST_WIN_ROOT_DIR / "venv"
TEST_WIN_RESULTS_DIR = TEST_WIN_ROOT_DIR / "results"
TEST_WIN_RUNNER_PATH = TEST_WIN_SRC_DIR / "bin/test_win_local.py"

TEST_CROS_ROOT_DIR = Path(r"/home/cros_ca")
TEST_CROS_SRC_DIR = TEST_CROS_ROOT_DIR / "src"
TEST_CROS_VENV_DIR = TEST_CROS_ROOT_DIR / "venv"
TEST_CROS_RESULTS_DIR = TEST_CROS_ROOT_DIR / "results"
TEST_CROS_RUNNER_PATH = TEST_CROS_SRC_DIR / "bin/test_cros_local.py"

# === Other variables:
TEST_START_FILE = ".test_started.txt"
TEST_END_FILE = ".test_ended.txt"

# === Test constants
TEST_MAX_MINUTES = 60  # Max test duration
