# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The remote runner that calls the local runner by pushing test scripts."""

import json
import time
from typing import List

import cros_ca_lib.config as cf
import cros_ca_lib.definition as df
from cros_ca_lib.host import base_host
from cros_ca_lib.runner import base_runner as br
from cros_ca_lib.runner import remote_runner
from cros_ca_lib.runner import test_result as tr


# cros_ca execution dir on the Windows DUT
# TODO: Move this to DUT as its member
starter_path = str(df.TEST_WIN_EXECTION_DIR)


class RemoteRunnerScript(remote_runner.RemoteRunner):
    """Trigger tests on a DUT from the host by push test script.

    Tests are started by pushing script to the DUT.
    This is currently used by Windows DUTs.
    """

    def call_local_test(
        self,
        parsed_test: br.ParsedTest,
        variables: dict,
        test_dir: str,
        result: tr.TestResult,  # pylint: disable=unused-argument
    ):
        # TODO: Check if the DUT supports poetry with:
        #       if self.dut.support_poetry:
        #           use poetry script
        #       else:
        #           use python to call
        script = f"cd {self.dut.local_runner_dir}\n"
        script += f"set PYTHONPATH={self.dut.src_dir}\n"
        script += f"python -m poetry run {self.dut.local_runner_func}"
        script += f" {cf.config_to_args()}"
        run_id = cf.config.run_id
        if not cf.config.run_id:
            # Generate a run_id.
            run_id = str(time.time())
            script += f' --run_id="{run_id}"'
        if not cf.config.no_console_output:
            script += " --no_console_output"  # Suppress local console output.
        for k in variables:
            if k in df.VARS_REMOTE_ONLY:
                # Ignore remote only vars.
                continue
            # Escape before single quote the value.
            escaped_v = variables[k].replace('"', '\\"')
            script += f' --var={k}="{escaped_v}"'
        script += f" {parsed_test.test_name}\n"
        script += "exit\n"

        self.dut.make_dir(starter_path)
        self.dut.remove_path(f"{starter_path}/test_ended.txt")
        self.logger.debug("Run the script locally on the DUT:\n%s", script)
        self.dut.write_file(f"{starter_path}/run.bat", script)
        self.dut.write_file(f"{starter_path}/start_test.txt", "")

        entries = self.dut.dir_entries(starter_path)
        mtime = 0
        for e in entries:
            if e.name == "run.bat":
                mtime = e.mtime
                break

        start_time = time.time()  # Record the start time
        last_log_time = (
            start_time - 50
        )  # Make the first log sooner by setting a past time.
        self.logger.info("Checking the local test progress...")
        has_start_file = False
        start_file = f"{self.dut.src_dir}/logs/{df.TEST_START_FILE}"
        end_file = f"{self.dut.src_dir}/logs/{df.TEST_END_FILE}"
        while True:
            time.sleep(5)
            current_time = time.time()
            elapsed_time = (current_time - start_time) / 60

            try:
                if not has_start_file and self.dut.path_exist(start_file):
                    start_json = self.dut.read_file(start_file)
                    self.logger.debug("start_json: %s", start_json)
                    try:
                        start_obj = json.loads(start_json)
                        has_start_file = start_obj["run_id"] == run_id
                    except Exception as e:
                        self.logger.warning("start json load error: %s", e)
                if has_start_file and self.dut.path_exist(end_file):
                    end_json = self.dut.read_file(end_file)
                    try:
                        end_obj = json.loads(end_json)
                        if end_obj["run_id"] == run_id:
                            self.logger.debug("end_json: %s", end_json)
                            self.logger.info(
                                "Test completed. Test duration: %.2f minutes.",
                                elapsed_time,
                            )
                            if end_obj["error"]:
                                self.logger.info(
                                    "Local test execution error: %s",
                                    end_obj["error"],
                                )
                            break
                    except Exception as e:
                        self.logger.warning("end json load error: %s", e)
            except Exception:
                # TODO: If the host cannot be reached for certain time,
                #       we should terminate the test
                pass

            matches = self.dut.find_under(
                f"{starter_path}", "test_ended.txt", mtime
            )
            if len(matches) > 0:
                if not has_start_file:
                    self.logger.warning(
                        "Test hasn't started but the run has completed. "
                        + "Please check run-time error on the DUT."
                    )
                else:
                    # No test_end file from the local test runner.
                    self.logger.warning(
                        "Test hasn't ended but the run has completed. "
                        + "Please check run-time error on the DUT."
                    )
                break

            if current_time - last_log_time >= 60:  # Log every 60 seconds
                self.logger.debug(
                    "Test has not finished yet. Time elapsed: %.2f"
                    + " minutes. Continue checking...",
                    elapsed_time,
                )
                last_log_time = current_time

            # Calculate elapsed time in minutes
            # Break after 30 minutes
            if elapsed_time >= df.TEST_MAX_MINUTES:
                self.logger.info(
                    "Test max time limit ({%d minutes) reached.",
                    df.TEST_MAX_MINUTES,
                )
                raise RuntimeError("Timeout waiting for test to complete")

        # Ensure the local results path exist
        if not self.results_dir.exists():
            self.results_dir.mkdir(parents=True)
        result_dir_patterm = r"(\d{8})-(\d{6})"
        entries: List[base_host.DirEntry] = self.dut.find_under(
            str(self.dut.results_dir), result_dir_patterm, mtime
        )

        results_found = False
        for match in entries:
            if match.is_dir:
                results_found = True
                remote_dir_name = match.name
                remote_path = f"{self.dut.results_dir}/{remote_dir_name}"
                self.logger.debug(
                    "Retrieving result directory %s...", remote_path
                )
                self.dut.download(remote_path, test_dir)
                self.dut.remove_path(remote_path)
                self.logger.info("Test results saved in %s", test_dir)
                self._merge_test_dir(
                    test_dir, remote_dir_name, parsed_test.test_name
                )

        if results_found:
            self.log_post_process(test_dir)
        else:
            self.logger.warning("No test results found.")
