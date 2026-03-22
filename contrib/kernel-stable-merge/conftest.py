# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""
Local plugin for pytest

See Also:
    - https://docs.pytest.org/en/7.1.x/how-to/writing_plugins.html#local-conftest-plugins
"""

import os

import pytest

from cros_ksm.git import KernelVersion, Repo

# For test reproducibility, hardcode the kernel major version.
KERNEL_MAJOR_VERSION = KernelVersion(6, 12)

@pytest.fixture(
    # Reuse the same repo object for all tests in the test session. This speeds
    # up the tests.
    #
    # See Also:
    #   - [Fixture Scopes](https://docs.pytest.org/en/7.1.x/how-to/fixtures.html#fixture-scopes)
    scope = "session",
)
def repo() -> Repo:
    env_var = "CROS_KSM_TEST_KERNEL_GIT_DIR"

    git_dir = os.environ.get(env_var, "")
    if not git_dir:
        pytest.fail(f"env var {env_var} is empty")

    return Repo(
        git_dir = git_dir,
        kernel_major_version = KERNEL_MAJOR_VERSION,
    )
