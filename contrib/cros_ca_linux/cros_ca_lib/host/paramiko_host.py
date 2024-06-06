# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The paramiko host implementation.

It uses paramiko library to communicate with the host
"""

# The following line disable the abstract function implementation warning.
# pylint: disable=W0223

import os
import re
import stat
import subprocess
import tempfile
import time
from typing import List, Tuple

import cros_ca_lib.definition as df
from cros_ca_lib.host import base_host
from cros_ca_lib.host import paramiko_utils
from cros_ca_lib.utils import logging as lg
import paramiko


logger = lg.logger


class ParamikoHost(base_host.BaseHost):
    """A class representing a remote host with Paramiko SSH connection."""

    def __init__(
        self,
        hostname: str,
        username: str,
        port: int = 22,
        password: str = None,
        key_filename: str = None,
    ):
        """Initializes a Host object with connection details.

        Args:
            hostname: The hostname or IP address of the remote server.
            username: The username for authentication.
            port: the SSH port.
            password: The password for authentication.
            key_filename: The path to a private key file for authentication.
        """

        self.port = port
        self.hostname = hostname
        self.username = username
        self.password = password
        # Use the default private key file.
        self.key_filename = (
            key_filename if key_filename else df.SSH_PRIVATE_KEY_FILE
        )
        base_host.BaseHost.__init__(
            self, hostname, username, port, password, self.key_filename
        )

        self._client = None  # Private attribute to store the SSH client
        self._sftp = None

    def _connect(self):
        """Connects to the remote host using Paramiko."""
        self._client = paramiko.SSHClient()
        self._client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
        os.chmod(self.key_filename, 0o400)
        self._client.connect(
            hostname=self.hostname,
            port=self.port,
            username=self.username,
            password=self.password,
            key_filename=self.key_filename,
        )
        self._connected = True
        self._sftp = None  # Invalidate the sftp client.

    def get_sftp(self):
        if not self._sftp:
            self._sftp = self._client.open_sftp()  # Open an SFTP client
        return self._sftp

    def close(self):
        """Disconnects from the remote host, if connected."""
        if self._sftp is not None:
            self._sftp.close()
        if self._connected and self._client is not None:
            self._client.close()
            logger.info("Disconnected from host: %s", self.hostname)

    def executing_command(self, command, timeout):
        """Executes a command on the remote host and return without waiting.

        Return even if the command is still being executed.

        Args:
            command: The command to execute on the remote server.
            timeout: timeout value.

        Returns:
            standard data
            standard err data
            a function to check if the execution is done. Get the return code,
                otherwise None.
        """

        if not self._connected:
            raise AssertionError("Not connected to host. Please connect first.")
        _, stdout, stderr = self._client.exec_command(command)
        stdout_data = b""
        stderr_data = b""
        end_time = time.time() + timeout
        while not stdout.channel.exit_status_ready():
            if stdout.channel.recv_ready():
                stdout_data += stdout.channel.recv(1024)
            if stderr.channel.recv_ready():
                stderr_data += stderr.channel.recv(1024)
            time.sleep(0.1)
            if time.time() >= end_time:
                break

        def is_done():
            if not stdout.channel.exit_status_ready():
                self._client.exec_command("")  # Keep alive
                return None
            return stdout.channel.recv_exit_status()

        return (
            stdout_data.decode().strip(),
            stderr_data.decode().strip(),
            is_done,
        )

    def exec_command(
        self,
        command: str,
        timeout: int = 30,
        retry_times: int = base_host.EXECUTION_DEFAULT_RETRY_TIMES,
    ) -> Tuple[str, str, int]:
        """Runs a program on the remote host using Paramiko.

        It returns the command's exit status.

        Args:
            command: The command to execute on the remote server.
            timeout: timeout value.
            retry_times: Retry times if encounter retryable errors.

        Returns:
            The standard output data, standard error data, and exit status.
        """

        # TODO: Implement retry
        logger.debug("command=%s", command)
        if not self._connected:
            raise AssertionError("Not connected to host. Please connect first.")
        _, stdout, stderr = self._client.exec_command(command)
        end_time = time.time() + timeout
        while not stdout.channel.exit_status_ready():
            time.sleep(0.1)
            if time.time() >= end_time:
                raise Exception("Timeout while executing the command")
        exit_status = stdout.channel.recv_exit_status()
        if exit_status != 0:
            raise Exception(
                "command failed with exit status "
                + f"{exit_status}:\n{stderr.read().decode().strip()}"
            )
        return (
            stdout.read().decode().strip(),
            stderr.read().decode().strip(),
            exit_status,
        )

    def read_file(self, filepath):
        """Reads the content of a file on the remote host using Paramiko.

        Args:
            filepath: The path to the file on the remote server.

        Returns:
            The content of the file as a string, or None if there's an error.
        """
        sftp = self.get_sftp()

        try:
            with sftp.open(filepath) as remote_file:
                data = remote_file.read()  # Read the file content
                return (
                    data.decode()
                )  # Decode bytes to string (assuming text file)
        except Exception as e:
            logger.warning("Error reading file %s: %s", filepath, e)
            raise

    def write_file(self, filepath, content):
        """Writes content to a remote file on the specified server using SFTP.

        Args:
            filepath: The path to the file on the remote server.
            content: The content to write to the file (string).

        Returns:
            The content of the file as a string, or None if there's an error.
        """
        sftp = self.get_sftp()

        try:
            with sftp.open(filepath, "w") as remote_file:
                remote_file.write(content.encode())
        except Exception as e:
            logger.warning("Error writing to remote file %s: %s", filepath, e)
            raise

    def lstat(self, path) -> base_host.DirEntry:
        sftp = self.get_sftp()
        attr = sftp.lstat(path)
        _, file_name = os.path.split(path)

        return base_host.DirEntry(
            file_name, attr.st_mtime, stat.S_ISDIR(attr.st_mode), attr.st_size
        )

    def upload(self, local_path, remote_path):
        """Copies a local file or directory to a remote SSH host.

        Args:
            local_path: The path to the local file on your system.
            remote_path: The path to the destination file on the remote server.
        """
        if not os.path.exists(local_path):
            raise ValueError(f"local file {local_path} does not exist")

        sftp = self.get_sftp()

        remote_dir, remote_file_name = os.path.split(remote_path)
        _, file_name = os.path.split(local_path)

        if self.path_exist(remote_path):
            entry = self.lstat(remote_path)
            if entry.is_dir:
                remote_dir = remote_path
                remote_file_name = file_name
            elif os.path.isdir(local_path):
                raise ValueError(
                    f"remote file {remote_path} exists when uploading local dir"
                )
        else:
            try:
                entry = self.lstat(remote_dir)
            except:
                raise ValueError(f"remote directory {remote_dir} doesn't exist")
            if not entry.is_dir:
                raise ValueError(f"remote path {remote_dir} is not a directory")
        try:
            if os.path.isfile(local_path):
                remote_file = os.path.join(remote_dir, remote_file_name)
                sftp.put(local_path, remote_file)
            elif os.path.isdir(local_path):
                target_dir = os.path.join(remote_dir, remote_file_name)
                self.make_dir(target_dir)
                if self.support_tgz:
                    with tempfile.NamedTemporaryFile(
                        delete=True, buffering=0
                    ) as temp_file:
                        temp_file_path = temp_file.name
                        _, tar_file = os.path.split(temp_file_path)
                        p = subprocess.run(
                            f"tar -czf {temp_file_path} -C {local_path} .",
                            shell=True,
                            check=True,
                        )
                        if p.returncode != 0:
                            raise Exception(
                                "tar the local directory failed "
                                + f"with code {p.returncode}"
                            )
                        target_tar = os.path.join(target_dir, tar_file)
                        sftp.put(temp_file_path, target_tar)
                        self.exec_command(
                            f"tar -xzf {target_dir}/{tar_file} -C {target_dir}",
                            60,
                        )
                        self.remove_path(target_tar)
                else:
                    for root, dirs, files in os.walk(local_path):
                        for file in files:
                            file_path = os.path.join(root, file)
                            target_file = file_path.replace(
                                local_path, target_dir
                            )
                            sftp.put(file_path, target_file)
                        for d in dirs:
                            # Construct the full path for subdirectory
                            sub_dir = os.path.join(root, d).replace(
                                local_path, target_dir, 1
                            )
                            self.make_dir(sub_dir)
            else:
                raise Exception(
                    "Only file and dir type is supported for uploading."
                )
        except Exception as e:
            logger.warning(
                "Error uploading file from %s to %s: %s",
                local_path,
                remote_path,
                e,
            )
            raise

    def download(self, remote_path, local_path):
        """Copies a remote file or directory from SSH host to a local path.

        Args:
            remote_path: The path to the destination file on the remote server.
            local_path: The path to the local file on your system.
        """

        if os.path.isfile(local_path):
            raise ValueError(f"local file {local_path} exists")

        # TODO: If the host supports zip/unzip, upload zip files
        if os.path.isdir(local_path):
            # Copy to an existing dir. Use the remote name.
            _, remote_file_name = os.path.split(remote_path)
            local_path = os.path.join(local_path, remote_file_name)
            if os.path.exists(local_path):
                raise ValueError(f"local path {local_path} exists")
        else:
            # Copy to a new target in a directory
            head, _ = os.path.split(local_path)
            if not os.path.isdir(head):
                raise ValueError(f"local directory {head} doesn't exist")

        sftp = self.get_sftp()

        try:
            attrs = sftp.lstat(remote_path)
            if stat.S_ISREG(attrs.st_mode):
                sftp.get(remote_path, local_path)
            elif stat.S_ISDIR(attrs.st_mode):
                # TODO: support_tgz check and use tgz file
                paramiko_utils.download_dir(sftp, remote_path, local_path)
            else:
                raise Exception("Target is neither a file nor a dir")
        except Exception as e:
            logger.warning(
                "Error downloading file from %s to %s: %s",
                remote_path,
                local_path,
                e,
            )
            raise

    def make_dir(self, dir_path):
        """Creates a directory on a remote host with parent directory creation.

        Args:
            dir_path: The path to create the directory on the remote server.
        """
        current_path = ""
        dir_path = dir_path.replace("\\", "/")
        if dir_path.startswith("/"):
            current_path = "/"
            dir_path = dir_path[1:]
        components = dir_path.split("/")

        sftp = self.get_sftp()

        try:
            for component in components:
                if not component:
                    continue
                current_path += f"{component}/"
                try:
                    sftp.lstat(current_path)
                except FileNotFoundError:
                    sftp.mkdir(current_path)
        except Exception as e:
            logger.warning("Error making directory %s: %s", dir_path, e)
            raise

    def path_exist(self, path):
        """Check if a file path exists.

        Args:
            path: The path to create the directory on the remote server.

        Returns:
            True if file exists. Otherwise False.
        """
        sftp = self.get_sftp()

        try:
            sftp.lstat(path)
            return True
        except Exception:
            return False

    def remove_path(self, path):
        if not self.path_exist(path):
            return
        sftp = self.get_sftp()
        attr = sftp.lstat(path)
        if not stat.S_ISDIR(attr.st_mode):
            sftp.remove(path)
            return
        paramiko_utils.remove_directory(sftp, path)

    def find_under(
        self, remote_path, regex, after_time=0
    ) -> List[base_host.DirEntry]:
        """Finds files or directories under a path.

        The entries with names matching a regex and modification times newer
        than a given time are returned.

        Args:
            remote_path: The base path to search under.
            regex: The regular expression to match file/directory names.
            after_time: The reference time (POSIX timestamp) for finding files
                        modified after this time.

        Returns:
            A list of dictionaries containing information about matching
            files/directories:
                - name (str): The filename or folder name.
                - mtime (float): The modification time as a POSIX timestamp
                                (seconds since epoch).
                - is_dir (bool): Whether it's a directory.
        """
        # Compile the regex for faster matching
        compiled_regex = re.compile(regex)
        matches = []
        sftp = self.get_sftp()
        for attr in sftp.listdir_attr(remote_path):
            filename = attr.filename
            if compiled_regex.match(filename):
                mtime = attr.st_mtime
                if mtime >= after_time:
                    matches.append(
                        base_host.DirEntry(
                            filename, mtime, stat.S_ISDIR(attr.st_mode)
                        )
                    )

        return matches

    def dir_entries(self, path) -> List[base_host.DirEntry]:
        """Get the directory entries on the host

        Returns:
            entries: list of DirEntry
        """
        entries = []
        sftp = self.get_sftp()
        for attr in sftp.listdir_attr(path):
            filename = attr.filename
            mtime = attr.st_mtime
            if stat.S_ISDIR(attr.st_mode):
                entries.append(base_host.DirEntry(filename, mtime, True))
            else:
                entries.append(base_host.DirEntry(filename, mtime, False))
        return entries
