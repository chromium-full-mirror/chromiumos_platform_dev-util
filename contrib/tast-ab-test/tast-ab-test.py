#!/usr/bin/env python3
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Tool to run a tast test with two different binaries.

This script is supposed to be used with the colab notebook at
go/cgd-tast-ab-template.
"""

import argparse
import datetime
import os
import subprocess
import sys

# pylint: disable-next=import-error
import colorama

# pylint: disable-next=import-error
# pylint: disable-next=import-error
from colorama import Fore
from colorama import Style


parsed_args = None
test_date = ""


def die(message):
    print(Style.BRIGHT + Fore.RED + f"FATAL: {message}")
    sys.exit(1)


def log(message):
    print(Style.BRIGHT + Fore.GREEN + f"LOG: {message}")


def warn(message):
    print(Style.BRIGHT + Fore.RED + f"WARN: {message}")


def get_buildbundle_arg(testname):
    if testname.startswith("arc.RuntimePerf"):
        return ["-buildbundle", "crosint"]
    return []


def get_result_dir(testname):
    # pylint: disable-next=line-too-long
    return (
        f"{parsed_args.outdir}/{parsed_args.tast_name}-{test_date}/{testname}"
    )


def cmd(command, run_always=False, capture_output=False):
    dryrun = parsed_args.dryrun and not run_always
    if dryrun:
        print(f"Run: {command}")
        return ""
    else:
        if parsed_args.verbose:
            print(f"RUN: {command}")

        if capture_output:
            ret = subprocess.run(command, capture_output=True, check=True)
            return str(ret.stdout)
        else:
            return subprocess.run(command, check=True)


def main():
    colorama.init(autoreset=True)

    parser = argparse.ArgumentParser()
    parser.add_argument("--dut", type=str, required=True, help="ssh host name")
    parser.add_argument(
        "--tast-name",
        type=str,
        required=True,
        help="tast test name (e.g. arc.RegularBoot.vm)",
    )
    parser.add_argument(
        "--count",
        type=int,
        default=10,
        help="the number of test iteration for each binary",
    )
    parser.add_argument(
        "--path",
        metavar="PATH",
        type=str,
        required=True,
        help="path to the binary to be replaced",
    )
    parser.add_argument(
        "--binary",
        metavar="PATH",
        nargs="+",
        required=True,
        help="binary to be tested. 2 or more binaries must be specified.",
    )
    parser.add_argument(
        "--outdir",
        metavar="PATH",
        default="tast-abtest-results",
        help="path to a directory where results are stored",
    )
    parser.add_argument("-d", "--dryrun", action="store_true", help="dry run")
    parser.add_argument(
        "-v", "--verbose", action="store_true", help="for debugging"
    )

    args = parser.parse_args()
    print(args)

    if args.dryrun:
        args.verbose = True

    if not args.binary or len(args.binary) < 2:
        die(f"Expected to get 2+ binary paths to test, but got {args.binary}")

    binaries = args.binary
    testcases = [os.path.basename(b) for b in binaries]
    if len(set(testcases)) != len(binaries):
        die("Use different basename for test binaries")

    # pylint: disable-next=global-statement
    global parsed_args
    parsed_args = args

    # pylint: disable-next=global-statement
    global test_date
    test_date = datetime.datetime.now().strftime("%Y-%m-%d-%H%M%S")

    result_dir = f"{args.outdir}/{args.tast_name}-{test_date}"

    for t in testcases:
        d = get_result_dir(t)
        cmd(["mkdir", "-p", d])

    build_bundle = get_buildbundle_arg(args.tast_name)

    # Check if the given tast test exists.
    supported_tests = cmd(
        ["tast", "list"] + build_bundle + [args.dut],
        run_always=True,
        capture_output=True,
    )
    if args.tast_name not in supported_tests:
        die(f"{args.tast_name} doesn't exit")

    for test_count in range(1, args.count + 1):
        log(f"Test #{test_count}")
        for testcase, binary in zip(testcases, binaries):
            log(f"Test #{test_count}: {testcase}")
            try:
                cmd(["ssh", args.dut, "restart ui"])
                cmd(["ssh", args.dut, "cp", binary, args.path])
                cmd(["ssh", args.dut, "echo 3 > /proc/sys/vm/drop_caches"])

                parent_dir = get_result_dir(testcase)
                result_dir = f"{parent_dir}/{test_count}"
                cmd_args = (
                    ["tast", "run", "-resultsdir", result_dir]
                    + build_bundle
                    + [args.dut, args.tast_name]
                )
                cmd(cmd_args)
                cmd(
                    [
                        "cp",
                        # pylint: disable-next=line-too-long
                        f"{result_dir}/tests/{args.tast_name}/results-chart.json",
                        f"{parent_dir}/{testcase}-{test_count}.json",
                    ]
                )
            except Exception as e:
                warn(f"Failed to run a test with {testcase}: {e}")


main()
