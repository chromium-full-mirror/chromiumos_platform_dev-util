# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The CrOS SSH host implementation."""

# The following line disable the abstract function implementation warning.
# pylint: disable=W0223

import datetime
from pathlib import Path
import re
from typing import List

import cros_ca_lib.definition as df
from cros_ca_lib.host import base_host
from cros_ca_lib.host import ssh_host
from cros_ca_lib.utils import control
from cros_ca_lib.utils import logging as lg


logger = lg.logger


def parse_ls_output(output_string: str) -> List[base_host.DirEntry]:
    """Parse the output of the Windows cmd "dir" command.

    It extracts information about files and directories.

    Args:
        output_string: The output string from the "dir" command.

    Returns:
        list: A list of dictionaries containing information about files and
              directories.
              Each dictionary has keys:
                  - name (str): The name of the file or directory.
                  - mtime (float): The modification time as a POSIX timestamp
                                  (seconds since epoch).
                  - size (float): The size.
    """
    # Example output:
    # drwxr-xr-x. 2 root root  4096 Apr 11 2022 results
    # drwxr-xr-x. 5 root root  4096 Apr 11 09:12 src
    # -rw-------. 1 root root   169 Apr  8 14:55 test.py
    # drwxr-xr-x. 6 root root  4096 Apr  9 15:43 venv

    # Regular expression pattern to match directory and file entries
    # Pattern only matches the files created within one year
    patterns = [
        (  # Within one year
            r"(?P<mode>\S+)\s+(?P<links>\d+)\s+(?P<owner>\S+)\s+"
            + r"(?P<group>\S+)\s+(?P<size>\d+)\s+"
            + r"(?P<date>\S+\s+\d{1,2}\s+\d{2}:\d{2})\s+(?P<name>.+)",
            "%b %d %H:%M",
        ),
        (  # Beyond one year
            r"(?P<mode>\S+)\s+(?P<links>\d+)\s+(?P<owner>\S+)\s+"
            + r"(?P<group>\S+)\s+(?P<size>\d+)\s+"
            + r"(?P<date>\S+\s+\d{1,2}\s+\d{4})\s+(?P<name>.+)",
            "%b %d %Y",
        ),
    ]

    entries = []
    for line in output_string.splitlines():
        for i, p in enumerate(patterns):
            pattern = p[0]
            match = re.match(pattern, line)
            if match:
                # Parse the date and time string into a datetime object
                d_str_copy = match.group("date")
                date_str = d_str_copy
                fmt_str = p[1]
                if i == 0:  # Special hanlde for datetime without a year.
                    now = datetime.datetime.now()
                    fmt_str = f"%Y {p[1]}"
                    current_year = now.year
                    last_year = current_year - 1
                    date_str = f"{current_year} {d_str_copy}"

                    tmp_time = datetime.datetime.strptime(date_str, fmt_str)
                    if tmp_time > now:
                        date_str = f"{last_year} {d_str_copy}"
                modification_time = datetime.datetime.strptime(
                    date_str, fmt_str
                ).timestamp()
                entry = base_host.DirEntry(
                    match.group("name"), modification_time, True
                )
                if match.group("mode").startswith("d"):
                    if match.group("name") not in [".", ".."]:
                        entries.append(entry)
                else:
                    entry.is_dir = False
                    entry.size = int(match.group("size"))
                    entries.append(entry)
    return entries


def ls_output_as_entries(output) -> List[base_host.DirEntry]:
    return parse_ls_output(output)


def ls_output_as_existence(output):
    pattern = "ls: .*: No such file or directory"
    if re.search(pattern, output):
        return False
    return True


def ls_error_is_not_found(output):
    return not ls_output_as_existence(output)


class CrOSSSHHost(ssh_host.SSHHost):
    """A class representing a remote CrOS host"""

    test_dir: Path = df.TEST_CROS_ROOT_DIR  # Dir for the test
    src_dir: Path = df.TEST_CROS_SRC_DIR  # Dir for the source code
    results_dir: Path = df.TEST_CROS_RESULTS_DIR  # Dir for the test results
    local_runner_path: Path = (
        df.TEST_CROS_RUNNER_PATH
    )  # File of the test runner
    local_runner_dir: Path = (
        df.TEST_CROS_RUNNER_DIR
    )  # Dir for the test local runner.
    local_runner_func: str = df.TEST_CROS_RUNNER_FUNC  # Python func for poetry

    support_tgz: bool = True

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
        if not password:
            password = "test0000"
        super().__init__(hostname, username, port, password, key_filename)

    def name(self) -> str:
        """The name of the host"""
        # TODO: get version, platform, spec, etc.
        return "CrOS Remote"

    def lstat(self, path) -> base_host.DirEntry:
        """Get a file or dir's information."""
        command = f'ls -dla "{path}"'
        stdout, _, _ = self.exec_command(command, 20)
        entries = ls_output_as_entries(stdout)
        if len(entries) == 0:
            raise Exception(f"No dir entry found for {path}")
        return entries[0]

    def normalize_path(self, path: str) -> str:
        """Normalize the windows path for SSH"""
        # Add quote to the file path in case spaces are included.
        return f'\\"{path}\\"'

    def path_exist(self, path: str) -> bool:
        """Check if a file path exists.

        Args:
            path: The path to create the directory on the remote server.

        Returns:
            True if file exists. Otherwise False.
        """
        command = rf"ls \"{self.normalize_path(path)}\""
        return control.retry(lambda: self._path_exist(command))

    def _path_exist(self, command: str) -> bool:
        try:
            stdout, _, _ = self.exec_command(command, 20, 0)
        except control.RetryableError as e:
            if ls_error_is_not_found(repr(e)):
                return False
            raise
        return ls_output_as_existence(stdout)

    def dir_entries(self, path) -> List[base_host.DirEntry]:
        """Get the directory entries on the host

        Returns:
            entries: list of DirEntry
        """
        stdout, _, _ = self.exec_command(
            rf'ls -la "{self.normalize_path(path)}"', 30
        )
        return ls_output_as_entries(stdout)

    def remove_path(self, path):
        if not self.path_exist(path):
            return
        command = rf'rm -rf "{self.normalize_path(path)}"'
        self.exec_command(command, 30)

    def make_dir(self, dir_path):
        """Create a directory on a remote host with parent directory creation.

        Args:
            dir_path: The path to create the directory on the remote server.
        """
        if self.path_exist(dir_path):
            return
        command = rf'mkdir -p "{self.normalize_path(dir_path)}"'

        self.exec_command(command, 30)

    def update_dependency(self):
        self.make_dir(str(df.TEST_CROS_VENV_DIR))
        # Push the "venv" folder to the remote
        self.update_dir(
            "Dependency",
            df.PROJECT_VENV_DIR,
            df.TEST_CROS_VENV_DIR,
            # Use the dependency inputs as the version file
            ver_file=str(
                df.PROJECT_ROOT_DIR / "development" / "requirements.in"
            ),
        )
