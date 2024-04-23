# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Implement a call that can do the SSH port forwarding."""

import subprocess
import time

import cros_ca_lib.definition as df
from cros_ca_lib.utils import logging as lg


logger = lg.logger


class SSHLocalForwarding:
    """A class to manage local port forwarding using SSH.

    This class provides methods to initiate and terminate local port forwarding
    tunnels using the `ssh` command with the `-N` and `-L` flags.
    """

    def __init__(
        self,
        remote_host: str,
        remote_port: int,
        local_port: int,
        ssh_host: str,
        username: str,
    ):
        """Initializes an SSHLocalForwarding object.

        Args:
            remote_host: The hostname or IP address of the remote server.
            remote_port: The port number on the remote server to forward
                               traffic from.
            local_port: The port number on the local machine to listen on.
            ssh_host: The SSH host.
            username: The user name for the SSH host.
        """
        self.remote_host = remote_host
        self.remote_port = remote_port
        self.local_port = local_port
        self.ssh_host = ssh_host
        self.username = username
        self.ssh_process = None

    def start(self):
        """Starts local port forwarding.

        Establishes a connection to the remote server using `ssh` with the `-N`
        flag (no pseudo-terminal) and the `-L` flag for local port forwarding.

        Raises:
            subprocess.CalledProcessError: If an error occurs during the SSH
                                           command execution.
        """
        try:
            # Attempt to execute with python3 (assuming Python 3 is preferred)
            subprocess.run(
                ["python3", "--version"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
                check=False,
            )
            python = "python3"
        except FileNotFoundError:
            python = "python"
        ssh_cmd = (
            f"{python} {df.PORT_FORWARDING_PROG_PATH} {self.local_port} "
            + f"{self.remote_host} {self.remote_port} "
            + f"{self.username}@{self.ssh_host} {df.SSH_PRIVATE_KEY_FILE}"
        )
        try:
            logger.info(
                "Port forwarding set up: localhost:%s -> host -> %s:%s",
                self.local_port,
                self.remote_host,
                self.remote_port,
            )
            # pylint: disable = consider-using-with
            self.ssh_process = subprocess.Popen(ssh_cmd.split())
            # pylint: enable = consider-using-with
            # Make sure the process is up and running before continuing
            time.sleep(2)

        except subprocess.CalledProcessError as e:
            logger.warning("Error setting up port forwarding: %s", e)
            self.ssh_process = None
            raise

    def stop(self):
        """Stops local port forwarding.

        Terminates the SSH process established for the port forwarding tunnel.
        """
        if self.ssh_process:
            self.ssh_process.kill()
            self.ssh_process = None
            logger.info(
                "Port forwarding for %s has been terminated.", self.local_port
            )
