# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The paramiko Windows host implementation."""

# The following line disable the abstract function implementation warning.
# pylint: disable=W0223

from pathlib import Path

import lib.definition as df
from lib.host import paramiko_host
from lib.utils import logging as lg


logger = lg.logger


class WinParamikoHost(paramiko_host.ParamikoHost):
    """A class representing a remote ChromeOS host"""

    test_dir: Path = df.TEST_WIN_ROOT_DIR  # Dir for the test
    src_dir: Path = df.TEST_WIN_SRC_DIR  # Dir for the source code
    results_dir: Path = df.TEST_WIN_RESULTS_DIR  # Dir for the test results
    local_runner_path: Path = (
        df.TEST_WIN_RUNNER_PATH
    )  # Dir for the test results

    support_tgz: bool = True  # Supported by powershell

    def __init__(
        self, hostname, username, port=22, password=None, key_filename=None
    ):
        """Initializes a Host object with connection details.

        Args:
            hostname: The hostname or IP address of the remote server.
            username: The username for authentication.
            port: The SSH port of the remote server.
            password: The password for authentication.
            key_filename: The path to a private key file for authentication.
        """
        self._shell = "cmd"
        if not password:
            password = "test0000"
        super().__init__(hostname, username, port, password, key_filename)

    def get_ps_command(self, command):
        if self._shell == "powershell":
            return command
        return "powershell " + command

    def _connect(self):
        """Connects to the remote host."""
        super(WinParamikoHost, self)._connect()

        stdout, _, _ = self.exec_command("dir", 20)
        if "Volume Serial Number" in stdout:
            self._shell = "cmd"
        elif "LastWriteTime" in stdout:
            self._shell = "powershell"
        logger.debug("Remote shell: %s", self._shell)

    def make_dir(self, dir_path):
        """Creates a directory on a remote host with parent directory creation.

        Args:
            dir_path: The path to create the directory on the remote server.
        """
        if self.path_exist(dir_path):
            return
        command = self.get_ps_command(
            f'New-Item -force -ItemType Directory -Path "{dir_path}"'
        )
        self.exec_command(command, 30)

    def _exec_unzip(self, command, timeout):
        command = self.get_ps_command(command)
        self.exec_command(command, timeout)
