# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The base host defintion as the parent for all hosts."""

# The following line disable the abstract function implementation warning.
# pylint: disable=W0223

import datetime
import os
from pathlib import Path
import time
from typing import List, Tuple, Union

import cros_ca_lib.definition as df
from cros_ca_lib.utils import logging as lg
from cros_ca_lib.utils import version
from selenium import webdriver


logger = lg.logger

EXECUTION_DEFAULT_RETRY_TIMES: int = 3


class DirEntry:
    """Directory entry class with required information."""

    def __init__(self, name, mtime, is_dir, size=0):
        self.name = name
        self.mtime = mtime
        self.is_dir = is_dir
        self.size = size

    def __repr__(self):
        return (
            f"name: {self.name}, mtime: {self.mtime}, "
            + f"is_dir: {self.is_dir}, size: {self.size}"
        )


class LoginOption:
    """The login options for CrOS"""

    username: str = None
    password: str = None
    arc: bool = False  # For CrOS to bring up ARC.
    chrome_args: List[str] = None
    enable_features: List[str] = None


class BaseHost:
    """A class representing a host."""

    test_dir: Path = None  # Dir for the test
    src_dir: Path = None  # Dir for the source code
    results_dir: Path = None  # Dir for the test results
    local_runner_dir: Path = None  # Dir for the test local runner.
    local_runner_path: Path = None  # File of the test local runner.
    local_runner_func: str = None  # Python func for poetry

    _connected: bool = False
    support_tgz: bool = False

    def __init__(
        self,
        hostname="",
        username="",
        port=22,
        password=None,
        key_filename=None,
    ):
        """Initializes a Host object with connection details.

        Args:
            hostname: The hostname or IP address of the remote server.
            username: The username for authentication.
            port: The port to connect to the host.
            password: The password for authentication (optional).
            key_filename: The path to a private key file for authentication
                          (optional).
        """

        self.port = port
        self.hostname = hostname
        self.username = username
        self.password = password
        # Use the default private key file.
        self.key_filename = key_filename

    def name(self) -> str:
        """The name of the host"""
        raise NotImplementedError("name() must be implemented in a subclass")

    def _connect(self):
        """Connects to the remote host using"""
        raise NotImplementedError(
            "_connect() must be implemented in a subclass"
        )

    def connect(self):
        """Connects to the remote host."""
        if self._connected:
            return
        try:
            self._connect()
            if self.hostname:
                logger.info("Connected to host: %s", self.hostname)
            self._connected = True
        except Exception:
            self._connected = False

            raise  # Re-raise the exception for caller handling

    def reconnect(self):
        """Reconnect to the remote host"""
        self._connected = False
        self.connect()

    def close(self):
        """Disconnects from the host, if connected."""
        raise NotImplementedError("close() must be implemented in a subclass")

    def executing_command(self, command, timeout):
        """Executes a command on the remote host and return without waiting."

        Return even if the command is still being executed.

        Args:
            command: The command to execute on the remote server.
            timeout: timeout value to wait.

        Returns:
            standard data
            standard err data
            a function to check if the execution is done. Get the return code,
                otherwise None.
        """
        raise NotImplementedError(
            "executing_command() must be implemented in a subclass"
        )

    def exec_command(
        self,
        command: str,
        timeout: int,
        retry_times: int = EXECUTION_DEFAULT_RETRY_TIMES,
    ) -> Tuple[str, str, int]:
        """Executes a command on the host.

        Args:
            command: The command to execute on the server.
            timeout: timeout value.
            retry_times: Retry times if encounter retryable errors.

        Returns:
            standard data, standard err data, and exit code
        """
        raise NotImplementedError(
            "exec_command() must be implemented in a subclass"
        )

    def get_pid(self, program):
        """Attempts to get the process ID (PID) of a program on a server.

        Args:
            program: The name of the program to find the PID for.

        Returns:
            The PID of the program as an integer if found, otherwise None.

        Raises:
            NotImplementedError: Always raised as this method is meant to be
                                overridden in subclasses.
        """
        raise NotImplementedError("get_pid() must be implemented in a subclass")

    def kill_pid(self, pid):
        """Attempts to kill a program by its PID.

        Args:
            pid: The process ID of the program to kill.

        Returns:
            True if the kill command execution was successful, False otherwise.

        Raises:
            NotImplementedError: Always raised as this method is meant to be
                                overridden in subclasses.
        """

        raise NotImplementedError(
            "kill_pid() must be implemented in a subclass"
        )

    def auto_login(self, option: LoginOption):
        """Automatically log into the DUT.

        Args:
            option: Login option.

        Raises:
            NotImplementedError: Always raised as this method is meant to be
                                overridden in subclasses.
        """

        raise NotImplementedError(
            "auto_login() must be implemented in a subclass"
        )

    def run_chrome(self):
        """Bring up the Chrome browser.

        Returns:
            Remote debugging port of the chrome browser.

        Raises:
            NotImplementedError: Always raised as this method is meant to be
                                overridden in subclasses.
        """

        raise NotImplementedError(
            "run_chrome() must be implemented in a subclass"
        )

    def run_chromedriver(self):
        """Bring up the chromedriver on the DUT.

        Returns:
            Listening port of the chromedriver process.

        Raises:
            NotImplementedError: Always raised as this method is meant to be
                                overridden in subclasses.
        """

        raise NotImplementedError(
            "run_chromedriver() must be implemented in a subclass"
        )

    def chrome_webdriver(
        self, opts: webdriver.ChromeOptions
    ) -> Union[webdriver.Chrome, webdriver.Remote]:
        """Create a selenium webdriver connecting to the browser on the DUT.

        Args:
            opts: The chrome options. Test case can pass the options
                  it desires to use.

        Returns:
            selenium webdriver.

        Raises:
            NotImplementedError: Always raised as this method is meant to be
                                overridden in subclasses.
        """

        raise NotImplementedError(
            "chrome_webdriver() must be implemented in a subclass"
        )

    def read_file(self, filepath):
        """Reads the content of a file on the remote host using Paramiko.

        Args:
            filepath: The path to the file on the remote server.

        Returns:
            The content of the file as a string, or None if there's an error.
        """
        raise NotImplementedError(
            "read_file() must be implemented in a subclass"
        )

    def write_file(self, filepath, content):
        """Writes content to a remote file on the specified server using SFTP.

        Args:
            filepath: The path to the file on the remote server.
            content: The content to write to the file (string).

        Returns:
            The content of the file as a string, or None if there's an error.
        """

        raise NotImplementedError(
            "write_file() must be implemented in a subclass"
        )

    def lstat(self, path) -> DirEntry:
        raise NotImplementedError("lstat() must be implemented in a subclass")

    def upload(self, local_path, remote_path):
        """Upload a local file or directory to the host.

        Args:
            local_path: The path to the local file on your system.
            remote_path: The path to the destination file on the remote server.
        """
        raise NotImplementedError("upload() must be implemented in a subclass")

    def download(self, remote_path, local_path):
        """Download a remote file or directory to a local path.

        Args:
            remote_path: The path to the destination file on the remote server.
            local_path: The path to the local file on your system.
        """
        raise NotImplementedError(
            "download() must be implemented in a subclass"
        )

    def make_dir(self, dir_path):
        """Create a directory on a remote host with parent directory creation.

        Args:
            dir_path: The path to create the directory on the remote server.
        """
        raise NotImplementedError(
            "make_dir() must be implemented in a subclass"
        )

    def remove_path(self, path):
        """Remove a path on a remote host and all its content.

        Args:
            path: The path to remove the directory on the remote server.
        """
        raise NotImplementedError(
            "remove_dir() must be implemented in a subclass"
        )

    def path_exist(self, path):
        """Check if a file path exists.

        Args:
            path: The path to create the directory on the remote server.

        Returns:
            True if file exists. Otherwise False.
        """
        raise NotImplementedError(
            "path_exist() must be implemented in a subclass"
        )

    def dir_entries(self, path) -> List[DirEntry]:
        """Get the directory entries on the host

        Returns:
            entries: list of DirEntry
        """
        raise NotImplementedError(
            "dir_entries() must be implemented in a subclass"
        )

    def find_under(self, remote_path, regex, after_time=0):
        """Find files or directories under a path.

        The entries must have names matching the regex and with modification
        times newer than the given time.

        Args:
            remote_path: The base path to search under.
            regex: The regular expression to match file/directory names.
            after_time: The reference time (POSIX timestamp) for finding files
                        modified after this time.

        Returns:
            A list of dictionaries containing information about matching
            files/directories:
                - name (str): The filename or folder name.
                - path (str): The full path to the file/directory on the remote
                              server.
                - mtime (float): The modification time as a POSIX timestamp
                                 (seconds since epoch).
                - is_dir (bool): Whether it's a directory.
        """
        raise NotImplementedError(
            "find_under() must be implemented in a subclass"
        )

    def get_download_path(self) -> str:
        """Returns the default downloads path for linux or windows"""
        if os.name == "nt":
            import winreg

            sub_key = (
                r"SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\Shell"
                + " Folders"
            )
            downloads_guid = "{374DE290-123F-4565-9164-39C4925E467B}"
            with winreg.OpenKey(winreg.HKEY_CURRENT_USER, sub_key) as key:
                location = winreg.QueryValueEx(key, downloads_guid)[0]
            return location
        else:
            return "/home/chronos/user/MyFiles/Downloads"

    def deploy(self) -> None:
        """Deploy the environment for testing"""

    def reboot(self, timeout: datetime.timedelta) -> None:
        """Reboot the host and wait until rebooted or timeout

        Args:
            timeout: The timeout waiting for host been access again
        """
        raise NotImplementedError("reboot() must be implemented in a subclass")

    def update_src_version(self):
        """Update the source code on the host"""
        if not self.src_dir:
            raise Exception("src_dir is not defined for the host")

        self.make_dir(str(self.src_dir))
        self.update_dir(
            "Source code",
            df.PROJECT_ROOT_DIR,
            self.src_dir,
            df.SRC_SYNC_DIRS,
        )

    def update_dependency(self):
        pass

    def normalize_path(self, path: str) -> str:
        """Normalize the path for the specific OS"""
        return path

    def update_dir(
        self,
        name: str,
        here_dir: Path,
        there_dir: Path,
        includes: List[str] = None,  # List of dir entries to update
        rm_there_dir: bool = True,
        ver_file: str = None,  # Version file to use
    ) -> bool:
        there_ver_file = str(there_dir / ".program_ver")
        if ver_file:
            with open(ver_file, "r", encoding="utf-8") as file:
                # Read the entire content of the file into a string
                local_ver = file.read()
        else:
            f, t = version.get_timestamp_ver(str(here_dir), includes=includes)
            local_ver = f"{f}: {t}"

        remote_ver = None
        update = True
        if self.path_exist(there_ver_file):
            remote_ver = self.read_file(there_ver_file)
            if remote_ver == local_ver:
                update = False
        if update:
            logger.info("Updating the %s...", name.lower())
            if ver_file:
                logger.info("%s has a newer version file %s", name, ver_file)
            else:
                logger.info("%s version (here): %s", name, local_ver)
                if remote_ver:
                    logger.info("%s version (there): %s", name, remote_ver)
            if rm_there_dir:
                self.remove_path(str(there_dir))
            there_parent = there_dir
            version.remove_pycache(str(here_dir))
            parent = here_dir
            if not includes:
                # Upload the whole directory
                parent = here_dir.parent
                includes = [here_dir.name]
                there_parent = there_dir.parent
            self.make_dir(str(there_parent))
            for entity in includes:
                f = str(parent / entity)
                self.upload(f, str(there_parent))
            # The dir has been uploaded. Write the version file.
            self.write_file(there_ver_file, local_ver)
        else:
            logger.info("%s version is up to date.", name)
        return update

    def mute(self, flag=True):
        """Mute or unmute all active speakers of the device.

        Args:
            flag: Mute the device if True, otherwise unmute.
        """
        raise NotImplementedError("mute() must be implemented in a subclass")

    def set_volume(self, percent: int) -> None:
        """Set the sound volume of all the active device speakers.

        Args:
            percent: The volume percentage, ranging from 0 to 100.
        """
        raise NotImplementedError(
            "set_volume must be implemented in a subclass"
        )

    def sleep(self, time_in_seconds: float):
        """Do sleep

        Args:
            time_in_seconds: sleep duration in seconds
        """
        time.sleep(time_in_seconds)

    def __enter__(self):
        """Context manager: connects automatically on entering the block."""
        self.connect()
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager: disconnects automatically on exiting the block."""
        self.disconnect()
