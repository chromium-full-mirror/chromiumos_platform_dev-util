"""Module to generate Mobly testcase metadata."""

# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
import argparse
import json
import os
import pathlib
import subprocess


# Runs the given par file with the -l flag and returns a list of test names.
def get_test_names(par_file):
    try:
        result = subprocess.run(
            [par_file, "--", "-l"], capture_output=True, text=True, check=True
        )
        output = result.stdout
        # Assuming test names are listed after "==========> ... <=========="
        test_names = output.split("==========>")[1].strip().splitlines()[1:]
        return test_names
    except subprocess.CalledProcessError as e:
        print(f"Error running {par_file}: {e}")
        return []


# Generates a list of test dicts from par files in the given
# directory and writes it to a JSON file.
def generate_test_list(directory, suite_name, output_file):
    test_list = []
    for filename in os.listdir(directory):
        if filename.endswith(".par"):
            par_file = os.path.join(directory, filename)
            test_names = get_test_names(par_file)
            for test_name in test_names:
                test_list.append(
                    {
                        "name": test_name,
                        "tags": [f"suite:{suite_name}"],
                        "extra_info": [{"executable_name": filename}],
                    }
                )
    with open(output_file, "w", encoding="utf-8") as f:
        json.dump(test_list, f, indent=2)


def _argparse_file_factory(path: str) -> pathlib.Path:
    """Factory method that builds a pathlib.Path object"""
    return pathlib.Path(path)


# Only run (outside) chroot. Executing pars requires special rte
if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Emit TestCase Json from pars for use \
            in generic_md.py"
    )
    parser.add_argument(
        "--directory",
        help="directory for par files",
        type=_argparse_file_factory,
        required=True,
    )
    parser.add_argument(
        "--suite_name",
        help="name for test case suites",
        default=False,
        required=True,
    )
    parser.add_argument(
        "--output_file",
        help="output file name for the json",
        type=_argparse_file_factory,
        required=True,
    )
    args = parser.parse_args()
    generate_test_list(args.directory, args.suite_name, args.output_file)
