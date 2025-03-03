# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import argparse
from datetime import datetime
import json
import os
import re
import shutil
import subprocess


Info = "[INFO] "
Info_Pass = Info + "PASS:"
Info_Fail = Info + "FAIL:"

test_status_NA = "N/A"
test_status_PASS = "PASS"
test_status_FAIL = "FAIL"

chromeos_directory = os.environ.get("HOME") + "/chromiumos"
capabilities_path = (
    chromeos_directory
    + "/src/platform/tast-tests/src/go.chromium.org/tast-tests/cros/remote/bundles/cros/wwcb/data/Capabilities.json"
)
tast_out_directory = chromeos_directory + "/out/"
result_directory = tast_out_directory + "tmp/qpop_"

report_path = (
    chromeos_directory
    + "/src/platform/dev/contrib/QPOP/test_report.html"
)
report_NA = "<td style='background-color:yellow;'>NA</td>"
report_Pass = "<td style='background-color:LimeGreen;'>Pass</td>"
report_Fail = "<td style='background-color:red;'>Fail</td>"


# Test na error.
class TestNAError(Exception):
    pass


# Test fail error.
class TestFailError(Exception):
    pass


class Test_Case:
    def __init__(
        self,
        name,
        category,
        externalUsbDeviceCount=0,
        externalDisplayCount=0,
        externalEthernetCount=0,
    ):
        self._name = name
        self._status = ""
        self._error_message = ""
        self._crashed = []
        self._attempts = ""
        self._category = category
        self._external_usb_device_count = externalUsbDeviceCount
        self._external_display_count = externalDisplayCount
        self._external_ethernet_count = externalEthernetCount


class QPOP:
    def _generation_report_table(self, tc, no):
        """
        Convert each test result into an HTML table.
        """
        try:
            status = ""
            error_message = tc._error_message
            if tc._status == test_status_NA:
                status = report_NA
            elif tc._status == test_status_PASS:
                status = report_Pass
                error_message = ""
            else:
                status = report_Fail
            self._html_content += f"""
            <tr>
            <td> {no} </td>
            <td> {tc._name} </td>
            {status}
            <td> {error_message} </td>
            <td> [] </td>
            <td> </td>
            </tr>
            """
        except Exception as error:
            print(f"Failed to generation report table: {error}")

    def _generation_report(self, na_count, pass_count, fail_count, total_count):
        """
        Read the report HTML template file and replace its contents with the actual content.
        """
        try:
            with open(report_path, "r") as f:
                f.seek(0)
                data = f.read()
                data = (
                    data.replace("QPOP_body", self._html_content)
                    .replace("QPOP_na", str(na_count))
                    .replace("QPOP_passed", str(pass_count))
                    .replace("QPOP_failed", str(fail_count))
                    .replace("QPOP_total", str(total_count))
                )
                with open(self._result_path + "/test_report.html", "w") as file:
                    file.write(data)
        except Exception as error:
            print(f"Failed to generation report: {error}")

    def _read_config(self):
        """
        Read Capabilities.json and pass the parameter to tast script.
        """
        try:
            with open(capabilities_path, "r") as f:
                ctrl_file = json.load(f)
        except:
            raise TestFailError("Cannot find the Capabilities JSON file")
        if (
            self._category == "docking"
            or self._category == "docking_daisychain"
        ):
            if len(str(ctrl_file["utils.wwcbIPPowerIp"]))<=0:
                raise TestFailError("Cannot read ip power ip")
            self._varslist.append(
                f'utils.wwcbIPPowerIp={ctrl_file["utils.wwcbIPPowerIp"]}'
            )
            if len(str(ctrl_file["Upstream"]["DockingID"]))<=0:
                raise TestFailError
            self._varslist.append(
                "DockingID=" + str(ctrl_file["Upstream"]["DockingID"])
            )
            USBTypeAIDArray = "USBTypeAIDArray="
            ExternalUSBTypeACount = 0
            ExternalDisplayCount = 0
            ExternalEthernetCount = 0
            for cf_key, cf_value in ctrl_file["Downstream"].items():
                if cf_key == "USB Type A":
                    for _, port_value in cf_value.items():
                        for usb_key, usb_value in port_value.items():
                            if usb_key == "USBTypeAIDArray" and len(usb_value)>0:
                                USBTypeAIDArray += f"{usb_value},"
                                ExternalUSBTypeACount += 1
                elif cf_key == "Display":
                    for _, port_value in cf_value.items():
                        for dis_key, dis_value in port_value.items():
                            if "ExtDispID" in dis_key and len(dis_value)>0:
                                self._varslist.append(f"{dis_key}={dis_value}")
                                ExternalDisplayCount += 1
                elif cf_key == "Ethernet":
                    for eth_key, eth_value in cf_value.items():
                        if "EthernetID" in eth_key and len(eth_value)>0:
                            self._varslist.append(f"{eth_key}={eth_value}")
                            ExternalEthernetCount += 1
            if ExternalUSBTypeACount > 0:
                self._varslist.append(USBTypeAIDArray[:-1])
            self._external_usb_device_count = ExternalUSBTypeACount
            self._external_display_count = ExternalDisplayCount
            self._external_ethernet_count = ExternalEthernetCount
            return
        if (
            self._category == "monitor"
            or self._category == "monitor_daisychain"
        ):
            ExternalDisplayCount = 0
            for cf_key, cf_value in ctrl_file["Downstream"].items():
                for _, port_value in cf_value.items():
                    for dis_key, dis_value in port_value.items():
                        if "ExtDispID" in dis_key and len(dis_value)>0:
                            self._varslist.append(f"{dis_key}={dis_value}")
                            ExternalDisplayCount += 1

            self._external_display_count = ExternalDisplayCount
            return

    def _collect_logs(self, tast_result, test_case):
        """
        Copy the results of tast run to the QPOP result directory.
        """
        try:
            if os.path.exists(tast_out_directory + tast_result):
                os.rename(
                    tast_out_directory + tast_result,
                    self._result_path + f"/results-{test_case}",
                )
            else:
                raise FileNotFoundError("The tast result not found")
        except Exception as error:
            print(f"Failed to collect logs: {error}")

    def _execute_tast(self, test_case):
        """
        Execute tast run.
        """
        args = []
        for var in self._varslist:
            args.append("-var=%s" % var)
        command = f'tast run {" ".join(str(arg) for arg in args)} {self._ip} {test_case}'
        print(
            datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%S.%fZ")
            + " "
            + Info
            + command
        )
        # Execute command by subprocess.
        process = subprocess.Popen(
            command,
            shell=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
        )

        # Read command output.
        output = ""
        for line in iter(process.stdout.readline, b""):
            output += line.decode().strip() + "\n"
            print(line.decode().strip())

        # Wait for command finish.
        process.wait()
        pattern_msg = r"\S+\s+\[\s*(PASS|FAIL)\s*\].*"
        pattern_result = r"Results saved to \S*"
        result_text = ""
        message = None
        try:
            # Matches pass or fail message.
            matches_msg = re.search(pattern_msg, output)
            # Matches result path.
            matches_result = re.findall(pattern_result, output)
            if matches_msg:
                message = matches_msg.group(0)
            if matches_result:
                result_text = matches_result[0].replace("Results saved to ", "")
        except Exception as error:
            print(f"Failed to matches result: {error}")
            raise TestFailError(
                f"Failed to execute tast run, Please check tast environment"
            )
        finally:
            # Collect result to QPOP result folder.
            self._collect_logs(result_text, test_case)
            # Check result.
            if "[ FAIL ]" in output:
                return False, message
            elif "[ PASS ]" in output:
                return True, message
            else:
                return False,"Failed to execute tast run, Please check tast environment"



    def __init__(self, category, ip, test_case):
        self._category = category
        self._ip = ip
        self._test_case = test_case
        self._result_path = ""
        self._varslist = []
        self._varslist.append(f"category={self._category}")
        self._varslist.append(f"newTestItem=testitem")
        self._external_usb_device_count = 0
        self._external_display_count = 0
        self._external_ethernet_count = 0
        self._html_content = ""
        self._test_cases = []
        # Create test case table.
        self._test_cases.append(
            Test_Case(
                name="wwcb.BootDUTWithDockConnected.clamshell_mode",
                externalDisplayCount=1,
                externalUsbDeviceCount=1,
                category=["docking"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.BootDUTWithDockConnected.tablet_mode",
                externalDisplayCount=1,
                externalUsbDeviceCount=1,
                category=["docking"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.ChangeExternalDisplayPosition.*",
                externalDisplayCount=1,
                category=["docking", "monitor"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.ChangeExternalDisplayResolution.*",
                externalDisplayCount=1,
                category=["docking", "monitor"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.ConnectDisplayBeforeBootDUT.*",
                externalDisplayCount=1,
                category=["docking", "monitor"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.CopyFilesViaDock.*",
                externalUsbDeviceCount=1,
                category=["docking"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.DaisyChain",
                externalDisplayCount=2,
                category=["docking_daisychain", "monitor_daisychain"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.DisconnectDisplayWhileShutdownDUT.*",
                externalDisplayCount=1,
                category=["docking", "monitor"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.DisconnectDisplayWhileSuspendDUT.*",
                externalDisplayCount=1,
                category=["docking", "monitor"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.NetworkSwitchingWithDock.*",
                externalDisplayCount=1,
                externalEthernetCount=1,
                category=["docking"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.NightLightViaDock.*",
                externalDisplayCount=1,
                category=["docking"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.PlugUnplugExternalDisplay.*",
                externalDisplayCount=1,
                category=["docking", "monitor"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.PlugUnplugFlipDock.clamshell_mode",
                externalDisplayCount=1,
                externalUsbDeviceCount=1,
                category=["docking"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.PlugUnplugFlipDock.tablet_mode",
                externalDisplayCount=1,
                externalUsbDeviceCount=1,
                category=["docking"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.PowerOffInternalDisplay.*",
                externalDisplayCount=1,
                category=["docking", "monitor"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.RebootDUTSwitchDockPower.clamshell_mode",
                externalDisplayCount=1,
                externalUsbDeviceCount=1,
                category=["docking"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.RebootDUTSwitchDockPower.tablet_mode",
                externalDisplayCount=1,
                externalUsbDeviceCount=1,
                category=["docking"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.ReconnectExternalDisplay.*",
                externalDisplayCount=1,
                category=["docking", "monitor"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.ShutdownDUTWithExternalDisplay.*",
                externalDisplayCount=1,
                category=["docking", "monitor"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.SuspendResumeDockPlugUnplug.clamshell_mode",
                externalDisplayCount=1,
                externalUsbDeviceCount=1,
                category=["docking"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.SuspendResumeDockPlugUnplug.tablet_mode",
                externalDisplayCount=1,
                externalUsbDeviceCount=1,
                category=["docking"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.SuspendResumeWithExternalDisplay.*",
                externalDisplayCount=1,
                category=["docking", "monitor"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.USBChargingViaDock.*",
                externalDisplayCount=1,
                category=["docking"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.WindowsPersistenceWithDualDisplay.*",
                externalDisplayCount=2,
                category=["docking", "monitor"],
            )
        )
        self._test_cases.append(
            Test_Case(
                name="wwcb.WindowsPersistenceWithSingleDisplay.*",
                externalDisplayCount=1,
                category=["docking", "monitor"],
            )
        )

    # Clear_config_file clears the file and add the string "{}" into the file.
    def _clear_config_file(self):
        try:
            with open(capabilities_path, "r+") as f:
                f.seek(0)
                # Clear data.
                f.truncate()
                f.write("{}")
            print(
                datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%S.%fZ")
                + " "
                + Info
                + f"Clear the capabilities file."
            )
        except Exception as error:
            print(f"Failed to clear config: {error}")

    def run(self):
        # Check the category is legal.
        if self._category not in [
            "docking",
            "monitor",
            "docking_daisychain",
            "monitor_daisychain",
        ]:
            raise TestNAError(f"The category is not legal: {self._category}")

        # Create result folder.
        self._result_path = result_directory + datetime.now().strftime(
            "%Y%m%d%H%M%S"
        )
        if not os.path.exists(self._result_path):
            os.mkdir(self._result_path)

        if (self._category == "docking" or self._category == "docking_daisychain"):
            # Execute self test ip power.
            status, msg = self._execute_tast("wwcb.SelftestIppower")
            if not status:
                raise TestFailError(f"Failed to self test for ip power: {msg}")
            # Read the ip power ip.
            try:
                with open(capabilities_path, "r") as f:
                    ctrl_file = json.load(f)
                    self._varslist.append(
                        f'utils.wwcbIPPowerIp={ctrl_file["utils.wwcbIPPowerIp"]}'
                    )
            except:
                raise TestFailError("Cannot find the Capabilities JSON file")

        # Execute self test fixture.
        status, msg = self._execute_tast("wwcb.SelftestFixture")
        if not status:
            raise TestFailError(f"Failed to self test for fixture: {msg}")

        # Execute self test webcam.
        status, msg = self._execute_tast("wwcb.SelftestWebcam")
        if not status:
            raise TestFailError(f"Failed to self test for webcam: {msg}")

        # Execute search system information.
        status, msg = self._execute_tast("wwcb.ChromeOsInfo")
        if not status:
            raise TestFailError(
                f"Failed to search chrome os informatioin: {msg}"
            )

        # After finish self-test & show the capability info, interact with user.
        command = f"python -m json.tool {capabilities_path}"
        # Execute command by subprocess.
        process = subprocess.Popen(
            command,
            shell=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
        )

        # Read command output.
        for line in iter(process.stdout.readline, b""):
            print(line.decode().replace("\n", ""))

        # Wait for command finish.
        process.wait()

        # After finish self-test & show the capability info, interact with user.
        print(
            datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%S.%fZ")
            + " "
            + Info
            + "Please check the capability information."
        )
        print(
            datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%S.%fZ")
            + " "
            + Info
            + "If the information was wrong, please check the capability first."
        )
        ans = input(
            datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%S.%fZ")
            + " "
            + Info
            + "Would you continue to proceed the test scripts? (Y/n)"
        )
        if ans.lower() != "y":
            return

        # Read capabilities.
        self._read_config()

        # Remove capabilities duplicates items.
        self._varslist = list(set(self._varslist))

        # Copy capabilities to result folder.
        if os.path.exists(capabilities_path):
            shutil.copyfile(
                capabilities_path, self._result_path + "/capabilitites.json"
            )

        NA_Count = 0
        Pass_Count = 0
        Fail_Count = 0
        Total_Count = 0
        # Single test.
        if self._test_case:
            singleTest = next(
                iter(
                    [
                        tc
                        for tc in self._test_cases
                        if tc._name == self._test_case
                    ]
                ),
                None,
            )
            if singleTest and self._category in singleTest._category:
                Total_Count += 1
                # Check device count.
                if (
                    self._external_display_count
                    < singleTest._external_display_count
                    or self._external_usb_device_count
                    < singleTest._external_usb_device_count
                    or self._external_ethernet_count
                    < singleTest._external_ethernet_count
                ):
                    NA_Count += 1
                    singleTest._status = test_status_NA
                    singleTest._error_message = "Insufficient devices to conduct the test, skipping this test case."
                else:
                    # Execute test cases.
                    status, msg = self._execute_tast(singleTest._name)
                    singleTest._error_message = msg
                    if status:
                        singleTest._status = test_status_PASS
                        Pass_Count += 1
                    else:
                        singleTest._status = test_status_FAIL
                        Fail_Count += 1
                self._generation_report_table(singleTest, Total_Count)
            else:
                print(
                    datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%S.%fZ")
                    + " "
                    + Info_Fail
                    + "Unable to find the test item or category error."
                )
        else:
            for tc in self._test_cases:
                # Category not match, skip this test case.
                if not self._category in tc._category:
                    continue
                Total_Count += 1
                # Check device count.
                if (
                    self._external_display_count < tc._external_display_count
                    or self._external_usb_device_count
                    < tc._external_usb_device_count
                    or self._external_ethernet_count
                    < tc._external_ethernet_count
                ):
                    NA_Count += 1
                    tc._status = test_status_NA
                    tc._error_message = "Insufficient devices to conduct the test, skipping this test case."
                else:
                    # Execute test cases.
                    status, msg = self._execute_tast(tc._name)
                    tc._error_message = msg
                    if status:
                        tc._status = test_status_PASS
                        Pass_Count += 1
                    else:
                        tc._status = test_status_FAIL
                        Fail_Count += 1
                self._generation_report_table(tc, Total_Count)

        # Create report html.
        self._generation_report(NA_Count, Pass_Count, Fail_Count, Total_Count)

        # Show the result on the terminal.
        print(
            datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%S.%fZ")
            + " "
            + Info
            + f"Your result would be:file://{self._result_path}"
        )

        # Clear config file.
        self._clear_config_file()


if __name__ == "__main__":
    # Create argparse.
    parser = argparse.ArgumentParser()

    # Add parameters from input.
    parser.add_argument(
        "-c", help="Please input category of test case", required=True
    )
    parser.add_argument(
        "-ip", help="Please input IP of chromebook", required=True
    )
    # This parameter determines whether to run tests by category or to specify a single test case for testing.
    parser.add_argument(
        "-tc", help="Please input name of test case", required=False
    )

    args = parser.parse_args()
    qpop = QPOP(args.c, args.ip, args.tc)
    qpop.run()
