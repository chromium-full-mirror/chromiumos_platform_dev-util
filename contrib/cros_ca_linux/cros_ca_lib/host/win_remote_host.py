# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The remote Windows host."""

import abc
import pathlib
import time

import cros_ca_lib.definition as df
from cros_ca_lib.host import base_host
from cros_ca_lib.utils import logging as lg


logger = lg.logger


class WinRemoteHost(abc.ABC, base_host.BaseHost):
    """A class representing a remote Windows host"""

    test_dir: pathlib.Path = df.TEST_WIN_ROOT_DIR  # Dir for the test
    src_dir: pathlib.Path = df.TEST_WIN_SRC_DIR  # Dir for the source code
    results_dir: pathlib.Path = (
        df.TEST_WIN_RESULTS_DIR
    )  # Dir for the test results
    local_runner_dir: pathlib.Path = (
        df.TEST_WIN_RUNNER_DIR
    )  # Dir for the test runner program
    local_runner_path: pathlib.Path = (
        df.TEST_WIN_RUNNER_PATH
    )  # File of the test runner program
    local_runner_func: str = df.TEST_WIN_RUNNER_FUNC  # Called by poetry

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

    def name(self) -> str:
        """The name of the host"""
        # TODO: get version, platform, spec, etc.
        return "Windows Remote"

    def _connect(self):
        """Connects to the remote host using ssh."""
        super()._connect()

        stdout, _, _ = self.exec_command("dir", 20)
        if "Volume Serial Number" in stdout:
            self._shell = "cmd"
        elif "LastWriteTime" in stdout:
            self._shell = "powershell"
        logger.debug("Remote shell: %s", self._shell)

    def get_ps_command(self, command):
        """Get the Powershell command."""
        if self._shell == "powershell":
            return command
        return "powershell " + command

    def remove_path(self, path):
        """Remove a path on a remote host and all its content.

        Args:
            path: The path to remove the directory on the remote server.
        """
        if not self.path_exist(path):
            return
        command = self.get_ps_command(
            f'rm -r -force "{self.normalize_path(path)}"'
        )
        self.exec_command(command, 30)

    def make_dir(self, dir_path: str) -> None:
        """Creates a directory on a remote host with parent directory creation.

        Args:
            dir_path: The path to create the directory on the remote server.
        """
        if self.path_exist(dir_path):
            return
        command = self.get_ps_command(
            "New-Item -force -ItemType Directory -Path "
            + f'"{self.normalize_path(dir_path)}"'
        )
        self.exec_command(command, 30)

    def _exec_unzip(self, command: str, timeout: int) -> None:
        command = self.get_ps_command(command)
        self.exec_command(command, timeout)

    def reboot(self, timeout: int) -> None:
        logger.info("Start to reboot.")
        command = self.get_ps_command("shutdown /r /t 1")
        self.exec_command(command, 20)
        begin = time.time()
        # Wait for the DUT to be disconnected.
        while True:
            # Wait for the DUT to be disconnected
            try:
                command = "echo '1'"
                self.exec_command(command, 20, 0)
                time.sleep(3)
            except Exception:
                logger.info("The host is disconnected.")
                break
            if time.time() - begin > timeout:
                raise Exception(
                    f"The host doesn't reboot within {timeout} seconds"
                )
        time.sleep(10)
        # Wait for the DUT to reconnect.
        while True:
            try:
                self.reconnect()
                break
            except Exception as e:
                if time.time() - begin > timeout:
                    raise Exception(
                        "The host is not available "
                        + f"after waiting for {timeout} seconds"
                    ) from e
                else:
                    logger.debug("Retrying connecting to the host...")
        logger.info("The host is rebooted successfully.")

    def deploy(self) -> None:
        """Deploy Windows host for be ready to run test case."""
        logger.info("Start to check and deploy the Windows DUT")

        # The execution directory
        self.make_dir(str(df.TEST_WIN_EXECTION_DIR))

        self.update_dir(
            "Tools",
            df.PROJECT_SCRIPT_DIR / "win/tools",
            df.TEST_WIN_TOOLS_DIR,
        )

        if self.update_dir(
            "Starter scripts",
            df.PROJECT_SCRIPT_DIR / "win/starter",
            df.TEST_WIN_STARTER_DIR,
        ):
            script = df.PROJECT_SCRIPT_DIR / "win/starter/startup/cros_ca.bat"
            start_path = pathlib.Path(
                f"C:/Users/{self.username}/AppData/Roaming",
                "Microsoft/Windows/Start Menu/Programs/Startup",
            )
            self.upload(str(script), str(start_path))
            logger.info("Updated starter scripts; need reboot to take effect.")
            self.reboot(60 * 3)  # 3 minutes

        logger.info("The deployment is completed successfully.")

    def update_src_version(self):
        """Update the source code on the host"""
        if not self.src_dir:
            raise Exception("src_dir is not defined for the host")

        self.make_dir(str(self.src_dir))
        self.update_dir(
            "Source code",
            df.PROJECT_ROOT_DIR,
            self.src_dir,
            df.SRC_SYNC_DIRS + ["win"],
        )

    def update_dependency(self):
        """Update the dependencies of the test on the host"""
        logger.info("Updating dependencies...")
        script = str(df.TEST_WIN_TOOLS_DIR / "create_venv.ps1")
        command = self.get_ps_command(f"{script} {df.TEST_WIN_RUNNER_DIR}")
        self.exec_command(command, 600)
        logger.info("Dependencies have been completely updated.")

    def auto_login(self, option: base_host.LoginOption):
        """Automatically log into the DUT.

        Args:
            option: Login option.
        """
