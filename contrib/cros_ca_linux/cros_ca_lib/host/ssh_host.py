# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The SSH host that uses basic SSH commandline tools to communicate with"""

# The following line disable the abstract function implementation warning.
# pylint: disable=W0223

import os
import re
import subprocess
import tempfile
from typing import Tuple

import cros_ca_lib.definition as df
from cros_ca_lib.host import base_host
from cros_ca_lib.utils import control
from cros_ca_lib.utils import logging as lg


logger = lg.logger
SSH_EXECUTION_TIMEOUT = 20
SSH_SCP_TIMEOUT = 60


def execute_local_command(
    command: str,
    check=True,
    timeout=SSH_EXECUTION_TIMEOUT,
    retry_times=base_host.EXECUTION_DEFAULT_RETRY_TIMES,
) -> Tuple[str, str, int]:
    """Execute command in local system, retry if encounter retryable errors.

    Args:
        command: The command to execute
        check: Throw error exit code is not zero, if True.
        timeout: Execution timeout in seconds.
        retry_times: Retry times if encounter retryable errors.

    Returns:
        A tuple contains standard output, standard error and return code
    """
    logger.debug("command=%s", command)
    return control.retry(
        lambda: _execute_local_command(command, check, timeout),
        retry_times,
    )


def _execute_local_command(
    command: str, check: bool, timeout: int
) -> Tuple[str, str, int]:
    try:
        with open("/dev/null", "rb") as dev_null:
            process = subprocess.run(
                command,
                shell=True,
                timeout=timeout,
                stdin=dev_null,  # Hide the stdin
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                # Check the exit status so the failed commandwill throw
                # exception.
                check=check,
            )
            stdout_data = process.stdout.decode("utf-8")
            stderr_data = process.stderr.decode("utf-8")
            return stdout_data, stderr_data, process.returncode
    except subprocess.TimeoutExpired:
        raise control.RetryableError(
            f"command execution timed out after {timeout} seconds."
        ) from None
    except subprocess.CalledProcessError as e:
        raise control.RetryableError(
            "command failed with exit status "
            f"{e.returncode}:\n{e.stderr.decode('utf-8')}"
        ) from None
    except Exception as e:
        raise Exception(f"execute command failed: {e}") from None


def count_files_and_size(directory):
    """Counts the number of files and total size recursively.

    Args:
        directory: The path to the directory to start counting from.

    Returns:
        A tuple containing two integers: the total number of files and the
        total size of all files in bytes.
    """
    total_files = 0
    total_size = 0
    for root, _, files in os.walk(directory):
        for file in files:
            # Get file path
            file_path = os.path.join(root, file)
            # Get file size
            file_size = os.path.getsize(file_path)
            total_files += 1
            total_size += file_size
    return total_files, total_size


SSH_OPTIONS = "-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null "


class SSHHost(base_host.BaseHost):
    """A remote host communicated with SSH command."""

    def __init__(
        self, hostname, username, port=22, password=None, key_filename=None
    ):
        """Initializes a Host object with connection details.

        Args:
            hostname: The hostname or IP address of the remote server.
            username: The username for authentication.
            port: The port to connect to.
            password: The password for authentication (optional).
            key_filename: The path to a private key file for authentication
                          (optional).
        """
        # Use the default private key file.
        self.key_filename = (
            key_filename if key_filename else df.SSH_PRIVATE_KEY_FILE
        )
        base_host.BaseHost.__init__(
            self,
            hostname,
            username,
            port,
            password,
            self.key_filename,
        )

        self._client = None  # Private attribute to store the SSH client
        self._sftp = None
        self._use_sshpass = True
        self.ssh_version = 8.9

    def _connect(self):
        """Connects to the remote host using ssh."""
        os.chmod(self.key_filename, 0o400)

        try:
            stdout, stderr, _ = execute_local_command("ssh -V", True, 1)
        except Exception:
            raise Exception(
                "Could not find ssh command. Please install ssh before "
                + "proceeding"
            )
        p = r"OpenSSH_(?P<version>\d+\.\d+).*"
        match = re.match(p, stdout + stderr)
        if not match:
            raise Exception(f"SSH version {stdout+stderr} is unknown.")
        self.ssh_version = float(match.group("version"))
        logger.debug("ssh client version: %f", self.ssh_version)
        try:
            execute_local_command("sshpass -V", True, 1)
            sshpass_available = True
        except Exception:
            sshpass_available = False

        self._use_sshpass = True
        if not sshpass_available:
            msg = (
                "SSHPASS checking:"
                + "\n  Could not find sshpass; ssh connection with password"
                + " will NOT be tried."
                + "\n  Install sshpass manually if you want to connect to the "
                + "remote host with a password"
            )
            logger.info(msg)
            self._use_sshpass = False
        else:
            sshpass_command = self.sshpass_command("echo '1'")
            try:
                _, stderr_data, code = execute_local_command(
                    sshpass_command, True, SSH_EXECUTION_TIMEOUT
                )
                self._use_sshpass = True
            except Exception:
                txt = (
                    "Failed to connect with sshpass and password. "
                    + "Trying connect with ssh and private key..."
                )

                logger.debug(txt)
                self._use_sshpass = False

        command = self.ssh_command("echo '1'")
        try:
            _, stderr_data, code = execute_local_command(
                command, False, SSH_EXECUTION_TIMEOUT
            )
            if code != 0:
                raise Exception(f"Error to connect to the host: {stderr_data}")
        except Exception as e:
            raise Exception(
                f"Error connecting to {self.hostname}: {e}.\nPlease check DUT "
                + "availability, or its password / publick key configuration."
            )

    def sshpass_command(self, command):
        prefix = (
            f"sshpass -p {self.password} ssh {SSH_OPTIONS} "
            + f"-p {self.port} {self.username}@{self.hostname} "
        )
        return prefix + command

    def ssh_command(self, command):
        if self._use_sshpass:
            return self.sshpass_command(command)
        prefix = (
            f"ssh {SSH_OPTIONS} -i {self.key_filename} "
            + "-o PreferredAuthentications=publickey "
            + f"-p {self.port} {self.username}@{self.hostname} "
        )
        return prefix + command

    def scp_download(self, remotepath, localpath):
        prefix = (
            f"scp -r {SSH_OPTIONS} -i {self.key_filename} -P {self.port} "
            + f"{self.username}@{self.hostname}:"
        )
        if self._use_sshpass:
            prefix = (
                f"sshpass -p {self.password} scp -r {SSH_OPTIONS} "
                + f"-P {self.port} {self.username}@{self.hostname}:"
            )
        # Add quote in case the path has space.
        command = f'{prefix}"\\"{remotepath}\\""  "{localpath}"'
        if self.ssh_version >= 9.0:
            command = f'{prefix}"{remotepath}"  "{localpath}"'
        # TODO: Handle the executing error and raise exception
        execute_local_command(command, True, SSH_SCP_TIMEOUT)

    def scp_upload(self, localpath, remotepath):
        prefix = (
            f"scp -r {SSH_OPTIONS} -i {self.key_filename} "
            + f'-P {self.port} "{localpath}" {self.username}@{self.hostname}:'
        )
        if self._use_sshpass:
            prefix = (
                f"sshpass -p {self.password} scp -r {SSH_OPTIONS} "
                + f"-P {self.port} {localpath} "
                + f"{self.username}@{self.hostname}:"
            )
        # Add quote in case the path has space.
        command = f'{prefix}"\\"{remotepath}\\""'
        if self.ssh_version >= 9.0:
            command = f'{prefix}"{remotepath}"'
        # TODO: Handle the executing error and raise exception
        execute_local_command(command, True, SSH_SCP_TIMEOUT)

    def close(self):
        """Disconnects from the remote host, if connected."""
        logger.info("Disconnected from host: %s", self.hostname)

    def exec_command(
        self,
        command: str,
        timeout=60,
        retry_times=base_host.EXECUTION_DEFAULT_RETRY_TIMES,
    ) -> Tuple[str, str, int]:
        """Executes a command on the remote host.

        Args:
            command: The command to execute on the remote server.
            timeout: The timeout value
            retry_times: Retry times if encounter retryable errors.

        Returns:
            standard output, standard error, and exit code
        """
        command = self.ssh_command(command)
        return execute_local_command(command, True, timeout, retry_times)

    def executing_command(self, command, timeout):
        """Executes a command on the remote host.

        It returns without waiting for exiting. Return even if the command
        is still being executed.

        Args:
            command: The command to execute on the remote server.
            timeout: timeout value.

        Returns:
            standard data
            standard err data
            a function to check if the execution is done. Get the return code,
                otherwise None.
        """
        command = self.ssh_command(command)
        # pylint: disable = consider-using-with
        p = subprocess.Popen(
            command, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE
        )
        # pylint: enable = consider-using-with

        def is_done():
            # TODO: For a long running ssh command, how to keep the ssh session
            #       alive?
            return p.poll()

        try:
            output, error = p.communicate(timeout=timeout)
        except subprocess.TimeoutExpired:
            return "", "", is_done

        return output.decode().strip(), error.decode().strip(), is_done

    def read_file(self, filepath):
        """Reads the content of a file on the remote host using ssh.

        Args:
            filepath: The path to the file on the remote server.

        Returns:
            The content of the file as a string, or None if there's an error.
        """
        try:
            with tempfile.NamedTemporaryFile(
                delete=True, buffering=0
            ) as temp_file:
                temp_file_path = temp_file.name
                self.download(filepath, temp_file_path)
                # Read the content of the temporary file
                with open(temp_file_path, "r", encoding="utf-8") as temp_file:
                    content = temp_file.read()
            return content
        except Exception as e:
            logger.warning("Failed to read file %s: %s", filepath, e)
            return None

    def write_file(self, filepath, content):
        """Writes content to a remote file on the specified server using SFTP.

        Args:
            filepath: The path to the file on the remote server.
            content: The content to write to the file (string).
        """

        with tempfile.NamedTemporaryFile(delete=True, buffering=0) as temp_file:
            temp_file.write(
                content.encode()
            )  # Encode content to bytes before writing

            # Copy the temporary file to the destination
            self.upload(temp_file.name, filepath)

    def _exec_unzip(self, command: str, timeout: int) -> None:
        self.exec_command(command, timeout)

    def upload(self, local_path, remote_path):
        """Copies a local file or directory to a remote SSH host.

        Args:
            local_path: The path to the local file on your system.
            remote_path: The path to the destination file on the remote server.
        """
        # If there are too large / many local files, we use tgz. Otherwise, we
        # still use scp because using tgz involves multiple command executions
        # and can have large overhead.
        fine_num, total_size = count_files_and_size(local_path)
        use_tgz = self.support_tgz and (
            fine_num > 100 or total_size > 80000000
        )  # 80MB
        if not use_tgz:
            try:
                self.scp_upload(local_path, remote_path)
                return
            except Exception as e:
                logger.warning(
                    "Error uploading file from %s to %s: %s",
                    local_path,
                    remote_path,
                    e,
                )
                raise

        remote_dir, remote_file_name = os.path.split(remote_path)
        _, file_name = os.path.split(local_path)

        if self.path_exist(remote_path):
            attr = self.lstat(remote_path)
            if attr.is_dir:
                remote_dir = remote_path
                remote_file_name = file_name
            elif os.path.isdir(local_path):
                raise ValueError(
                    f"remote file {remote_path} exists when uploading local dir"
                )
        else:
            try:
                attr = self.lstat(remote_dir)
            except:
                raise ValueError(f"remote directory {remote_dir} doesn't exist")
            if not attr.is_dir:
                raise ValueError(f"remote path {remote_dir} is not a directory")
        try:
            if os.path.isfile(local_path):
                remote_file = os.path.join(remote_dir, remote_file_name)
                self.scp_upload(local_path, remote_file)
            elif os.path.isdir(local_path):
                target_dir = os.path.join(remote_dir, remote_file_name)
                self.make_dir(target_dir)
                if use_tgz:
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
                        self.scp_upload(temp_file_path, target_tar)
                        self._exec_unzip(
                            f"tar -xzf {target_dir}/{tar_file} -C {target_dir}",
                            60,
                        )
                        self.remove_path(target_tar)
                else:
                    self.scp_upload(local_path, remote_path)
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
        try:
            self.scp_download(remote_path, local_path)
        except Exception as e:
            logger.warning(
                "Error downloading file from %s to %s: %s",
                remote_path,
                local_path,
                e,
            )
            raise

    def find_under(self, remote_path, regex, after_time=0):
        """Finds under a path with names regex and newer modification time

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

        entries = self.dir_entries(remote_path)
        # Compile the regex for faster matching
        compiled_regex = re.compile(regex)
        matches = []

        for entry in entries:
            filename = entry.name
            if compiled_regex.match(filename):
                mtime = entry.mtime
                if mtime >= after_time:
                    matches.append(entry)
        return matches
