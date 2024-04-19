# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The SSH Windows host implementation."""

# The following line disable the abstract function implementation warning.
# pylint: disable=W0223

import datetime
from pathlib import Path
import re
from typing import List

import lib.definition as df
from lib.host import base_host
from lib.host import ssh_host
from lib.utils import logging as lg


logger = lg.logger


def parse_cmd_dir_output(output_string: str) -> List[base_host.DirEntry]:
    """Parse the output of the Windows cmd "dir" command

    It extracts information about files and directories.

    Args:
        output_string: The output string from the "dir" command.

    Returns:
        entries: list of DirEntry
    """

    # Regular expression pattern to match directory and file entries
    patterns = [
        (
            r"(?P<date>\d{2}/\d{2}/\d{4}  \d{2}:\d{2} (AM|PM))\s+"
            + r"(?P<dir_or_space>\d+|<DIR>)\s+(?P<name>.+)",
            "%m/%d/%Y %I:%M %p",
        ),
        (
            r"(?P<date>\d{2}-\d{2}-\d{4}  \d{2}:\d{2} (AM|PM))\s+"
            + r"(?P<dir_or_space>\d+|<DIR>)\s+(?P<name>.+)",
            "%m-%d-%Y %I:%M %p",
        ),
    ]

    entries = []
    for line in output_string.splitlines():
        for p in patterns:
            pattern = p[0]
            match = re.match(pattern, line)
            if match:
                # Parse the date and time string into a datetime object
                date_str = match.group("date")
                modification_time = datetime.datetime.strptime(
                    date_str, p[1]
                ).timestamp()
                entry = base_host.DirEntry(
                    match.group("name"), modification_time, True
                )
                if match.group("dir_or_space") == "<DIR>":
                    if match.group("name") not in [".", ".."]:
                        entries.append(entry)
                else:
                    entry.is_dir = False
                    entry.size = int(match.group("dir_or_space"))
                    entries.append(entry)
    return entries


def parse_powershell_dir_output(output_string: str) -> List[base_host.DirEntry]:
    """Parse the output of the Windows powershell "dir" command.

    It extracts information about files and directories.

    Args:
        output_string: The output string from the "dir" command.

    Returns:
        entries: list of DirEntry
    """
    # Regular expression pattern to match directory and file entries
    patterns = [
        (
            r"(?P<mode>\S+)\s+"
            + r"(?P<date>\d{1,2}/\d{1,2}/\d{4}\s+\d{1,2}:\d{2} (AM|PM))\s+"
            + r"(?P<size>\d+)?\s+(?P<name>.+)",
            "%m/%d/%Y %I:%M %p",
        ),
        (
            r"(?P<mode>\S+)\s+"
            + r"(?P<date>\d{1,2}-\d{1,2}-\d{4}\s+\d{1,2}:\d{2} (AM|PM))\s+"
            + r"(?P<size>\d+)?\s+(?P<name>.+)",
            "%m-%d-%Y %I:%M %p",
        ),
    ]

    entries = []
    for line in output_string.splitlines():
        for p in patterns:
            pattern = p[0]
            match = re.match(pattern, line)
            if match:
                # Parse the date and time string
                date_str = match.group("date")
                modification_time = datetime.datetime.strptime(
                    date_str, p[1]
                ).timestamp()  # Adjust format for PS

                entry = base_host.DirEntry(
                    match.group("name").strip(), modification_time, True
                )
                if not match.group("mode").startswith("d"):
                    entry.is_dir = False
                    if match.group("size"):
                        entry.size = int(match.group("size"))
                entries.append(entry)
    return entries


def dir_output_as_entries(shell, output) -> List[base_host.DirEntry]:
    if shell == "cmd":
        return parse_cmd_dir_output(output)
    else:
        return parse_powershell_dir_output(output)


def dir_output_as_existence(shell, output):
    if shell == "powershell":
        pattern = "dir : Cannot find (path|driver)"
        if re.search(pattern, output):
            return False
    else:
        pattern = "(File Not Found|The system cannot find the file specified)"
        if re.search(pattern, output):
            return False
    return True


def dir_error_is_not_found(shell, output):
    return not dir_output_as_existence(shell, output)


def normalize_win_path_for_ssh(path: str):
    return path.replace("/", "\\").replace("\\", "\\\\")


class WinSSHHost(ssh_host.SSHHost):
    """A class representing a remote Windows host"""

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
        """Get the Powershell command."""
        if self._shell == "powershell":
            return command
        return "powershell " + command

    def _connect(self):
        """Connects to the remote host using ssh."""
        super(WinSSHHost, self)._connect()

        stdout, _, _ = self.exec_command("dir", 20)
        if "Volume Serial Number" in stdout:
            self._shell = "cmd"
        elif "LastWriteTime" in stdout:
            self._shell = "powershell"
        logger.debug("Remote shell: %s", self._shell)

    def lstat(self, path) -> base_host.DirEntry:
        command = self.get_ps_command(
            rf"Get-Item \"{normalize_win_path_for_ssh(path)}\""
        )
        stdout, _, _ = self.exec_command(command, 20)
        # Powershell Get-Item has same output as dir.
        entries = dir_output_as_entries("powershell", stdout)
        if len(entries) == 0:
            raise Exception(f"No dir entry found for {path}")
        return entries[0]

    def path_exist(self, path):
        """Check if a file path exists.

        Args:
            path: The path to create the directory on the remote server.

        Returns:
            True if file exists. Otherwise False.
        """
        try:
            command = rf"dir \"{normalize_win_path_for_ssh(path)}\""
            stdout, _, _ = self.exec_command(command, 20)
        except Exception as e:
            if dir_error_is_not_found(self._shell, repr(e)):
                return False
            raise
        return dir_output_as_existence(self._shell, stdout)

    def dir_entries(self, path) -> List[base_host.DirEntry]:
        """Get the directory entries on the host

        Returns:
            entries: list of DirEntry
        """
        stdout, _, _ = self.exec_command(
            rf"dir \"{normalize_win_path_for_ssh(path)}\"", 30
        )
        return dir_output_as_entries(self._shell, stdout)

    def remove_path(self, path):
        if not self.path_exist(path):
            return
        command = self.get_ps_command(
            rf"rm -r -force \"{normalize_win_path_for_ssh(path)}\""
        )
        self.exec_command(command, 30)

    def make_dir(self, dir_path):
        """Creates a directory on a remote host with parent directory creation.

        Args:
            dir_path: The path to create the directory on the remote server.
        """
        if self.path_exist(dir_path):
            return
        command = self.get_ps_command(
            "New-Item -force -ItemType Directory -Path "
            + rf"\"{normalize_win_path_for_ssh(dir_path)}\""
        )
        self.exec_command(command, 30)

    def _exec_unzip(self, command, timeout):
        command = self.get_ps_command(command)
        self.exec_command(command, timeout)
