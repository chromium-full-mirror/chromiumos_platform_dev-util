# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The SSH Windows host implementation."""

# The following line disable the abstract function implementation warning.
# pylint: disable=W0223

import datetime
import re
from typing import List

from cros_ca_lib.host import base_host
from cros_ca_lib.host import ssh_host
from cros_ca_lib.host import win_remote_host
from cros_ca_lib.utils import control
from cros_ca_lib.utils import logging as lg


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
        pattern = (
            "(File Not Found|The system cannot find the (file|path) specified)"
        )
        if re.search(pattern, output):
            return False
    return True


def dir_error_is_not_found(shell, output):
    return not dir_output_as_existence(shell, output)


class WinSSHHost(win_remote_host.WinRemoteHost, ssh_host.SSHHost):
    """A class representing a remote Windows host operating with ssh command"""

    def normalize_path(self, path: str) -> str:
        """Normalize the windows path for SSH"""
        # Going through SSH commandline, a singe backslash should be
        # double backslashed.
        p = path.replace("/", "\\").replace("\\", "\\\\")
        # Add quote to the file path in case spaces are included.
        return f'\\"{p}\\"'

    def lstat(self, path) -> base_host.DirEntry:
        command = self.get_ps_command(
            rf"Get-Item \"{self.normalize_path(path)}\""
        )
        stdout, _, _ = self.exec_command(command, 20)
        # Powershell Get-Item has same output as dir.
        entries = dir_output_as_entries("powershell", stdout)
        if len(entries) == 0:
            raise Exception(f"No dir entry found for {path}")
        return entries[0]

    def path_exist(self, path) -> bool:
        """Check if a file path exists.

        Args:
            path: The path to create the directory on the remote server.

        Returns:
            True if file exists. Otherwise False.
        """
        command = f'dir "{self.normalize_path(path)}"'
        return control.retry(lambda: self._path_exist(command))

    def _path_exist(self, command: str) -> bool:
        try:
            stdout, _, _ = self.exec_command(command, 20, 0)
        except control.RetryableError as e:
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
            f'dir "{self.normalize_path(path)}"', 30
        )
        return dir_output_as_entries(self._shell, stdout)
