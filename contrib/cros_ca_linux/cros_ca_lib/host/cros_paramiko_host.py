# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The CrOS paramiko host implementation."""

# The following line disable the abstract function implementation warning.
# pylint: disable=W0223

from pathlib import Path
from typing import Union

import cros_ca_lib.definition as df
from cros_ca_lib.host import base_host
from cros_ca_lib.host import paramiko_host
from cros_ca_lib.host.ssh import port_forwarding as pf
from cros_ca_lib.utils import logging as lg
import paramiko
from selenium import webdriver
from selenium.webdriver.chrome.options import Options


logger = lg.logger


class CrOSParamikoHost(paramiko_host.ParamikoHost):
    """A class representing a remote ChromeOS host"""

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
        self._chromedriver_pid = None
        self._chromedriver_port_forwarding = None
        self._chrome_debugging_port_forwarding = None
        super().__init__(hostname, username, port, password, key_filename)

        self._chromedriver_port = 0

    def name(self) -> str:
        """The name of the host"""
        # TODO: get version, platform, spec, etc.
        return "CrOS Remote"

    def get_pid(self, program):
        """Attempts to get the process ID (PID) of a program on a remote server.

        Args:
            program: The name of the program to find the PID for.

        Returns:
            The PID of the program as an integer if found, otherwise None.
        """
        command = f"pidof {program}"

        try:
            stdout_data, _, _ = self.exec_command(command, 10)
            if len(stdout_data) > 0:
                try:
                    pid = int(stdout_data)
                    return pid
                except ValueError:
                    print("Error: Could not parse PID from output.")
                    return None
            else:
                return None
        except Exception as e:
            logger.warning("SSH error while getting PID: %s", e)

        return None

    def auto_login(self, option: base_host.LoginOption):
        """Automatically log into the DUT.

        Args:
            option: Login option.

        Raises:
            RuntimeError: if autologin fails.
        """
        logger.info("ChromeOS auto login...")

        command = "/usr/local/autotest/bin/autologin.py "
        # Don't keep state. Delete existing profile.
        # This is the only way it can work together with gaia login. Without
        # this option, if another account has already exists on the DUT,
        # the GAIA login would fail on the screen of "Choose your setup"
        # for personal use, for a child or for work.
        command += "-d "
        if option.arc:
            command += "--arc --no-arc-syncs "
        if option.username and option.password:
            command += f"-u {option.username} -p {option.password} "
        # TODO: Other Chrome options can be added here. These options
        #       can be set in the "option" class and applied to the
        #       autologin.py argument here.

        _, _, result = self.exec_command(command, 90)
        if result != 0:
            logger.warning("Failed to do autologin. code: %d", result)
            raise RuntimeError("Failed to do autologin")

    def run_chrome(self):
        """Bring up the Chrome browser.

        Returns:
            Remote debugging port of the chrome browser.
        """

        raise NotImplementedError(
            "run_chrome() must be implemented in a subclass"
        )

    def run_chromedriver(self):
        """Bring up the chromedriver on the DUT."""

        # Check if the chromedriver is already running.
        pid = self.get_pid("chromedriver")
        if pid:
            if not self.kill_pid(pid):
                raise RuntimeError("Failed to kill existing chromedirver")

        cdriver_port = 9515
        # Start new chromedriver process
        stdout_data, stderr_data, _ = self.executing_command(
            f"/usr/local/chromedriver/chromedriver -p {cdriver_port}", 5
        )

        # Check the process and get it's ID
        pid = self.get_pid("chromedriver")

        if pid:
            self._chromedriver_pid = pid
        else:
            if len(stderr_data) > 0:
                logger.info("Error output from chromedriver:")
                logger.info(stderr_data)
            elif len(stdout_data) > 0:
                logger.info("Output from chromedriver:")
                logger.info(stdout_data)
            raise RuntimeError("Failed to run chromedriver")

        self._chromedriver_port = cdriver_port

    def chrome_webdriver(
        self, opts: webdriver.ChromeOptions = None
    ) -> Union[webdriver.Chrome, webdriver.Remote]:
        """Create a selenium webdriver connecting to the browser on the DUT.

        Args:
            opts: The chrome options. Test case can pass the options
                  it desires to use.

        Returns:
            selenium remote webdriver.
        """
        self.run_chromedriver()

        file_data = self.read_file("/home/chronos/DevToolsActivePort")
        if not file_data:
            raise RuntimeError("Failed to get active debugging port")
        lines = file_data.splitlines()  # Split the string into lines
        first_line = lines[
            0
        ].strip()  # Get the first line and remove whitespaces
        debugging_port = int(
            first_line.split()[0]
        )  # Convert the first word to integer

        if (
            self._chromedriver_port_forwarding
            and self._chromedriver_port_forwarding.local_port
            == self._chromedriver_port
        ):
            pass
        else:
            if self._chromedriver_port_forwarding:
                self._chromedriver_port_forwarding.stop()
            self._chromedriver_port_forwarding = pf.SSHLocalForwarding(
                "127.0.0.1",
                self._chromedriver_port,
                self._chromedriver_port,
                self.hostname,
                self.username,
            )
            self._chromedriver_port_forwarding.start()
        if (
            self._chrome_debugging_port_forwarding
            and self._chrome_debugging_port_forwarding.local_port
            == debugging_port
        ):
            pass
        else:
            if self._chrome_debugging_port_forwarding:
                self._chrome_debugging_port_forwarding.stop()
            self._chrome_debugging_port_forwarding = pf.SSHLocalForwarding(
                "127.0.0.1",
                debugging_port,
                debugging_port,
                self.hostname,
                self.username,
            )
            self._chrome_debugging_port_forwarding.start()

        executorURL = f"http://localhost:{self._chromedriver_port}"
        ip_address = "127.0.0.1"
        options = opts if opts is not None else Options()
        options.add_experimental_option(
            "debuggerAddress", f"{ip_address}:{debugging_port}"
        )
        return webdriver.Remote(
            command_executor=executorURL,
            options=options,
        )

    def kill_pid(self, pid):
        """Attempts to kill a remote program by its PID.

        Args:
            pid: The process ID of the program to kill.

        Returns:
            True if the kill command execution was successful, False otherwise.
        """

        command = (
            f"kill -9 {pid}"  # Use a strong signal for reliable termination
        )

        try:
            _, _, exit_status = self.exec_command(command, 10)
            return exit_status == 0  # True if successful, False otherwise
        except paramiko.SSHException as e:
            logger.warning("SSH error while killing process: %s", e)
            return False

    def make_dir(self, dir_path):
        """Creates a directory on a remote host with parent directory creation.

        Args:
            dir_path: The path to create the directory on the remote server.
        """
        try:
            create_command = f"mkdir -p {dir_path}"
            _, stderr_data, exit_status = self.exec_command(create_command, 10)
            if exit_status != 0:
                raise Exception(f"Error creating directory: {stderr_data}")

        except Exception as e:
            logger.warning("Error creating remote directory: %s", e)
            raise

    def close(self):
        if self._chromedriver_pid:
            self.kill_pid(self._chromedriver_pid)
        if self._chromedriver_port_forwarding:
            self._chromedriver_port_forwarding.stop()
            self._chromedriver_port_forwarding = None
        if self._chrome_debugging_port_forwarding:
            self._chrome_debugging_port_forwarding.stop()
            self._chrome_debugging_port_forwarding = None

        super().close()

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
