# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Centralized Suite Proto generation script."""

import argparse
import pathlib
import sys


sys.path.insert(1, str(pathlib.Path(__file__).parent.resolve() / "../../"))

from src.tools import suite_set_utils  # pylint: disable=wrong-import-position


def main(args):
    suite_set_utils.prep_centralized_suites(
        args.suites_input,
        args.suites_output,
        args.suite_sets_input,
        args.suite_sets_output,
        args.test_metadata_dir,
    )


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        prog="prep_centralized_suites",
        description=("Prepares Centralized Suite proto files"),
    )
    parser.add_argument(
        "--suite_sets_input",
        help="Space separated list of paths to input SuiteSetList jsonpb files",
        nargs="+",
    )
    parser.add_argument(
        "--suite_sets_output",
        help="Path to write SuiteSets proto file to",
        required=True,
    )
    parser.add_argument(
        "--suites_input",
        help="Space separated list of paths to input SuiteList jsonpb files",
        nargs="+",
    )
    parser.add_argument(
        "--suites_output",
        help="Path to write Suites proto file to",
        required=True,
    )
    parser.add_argument(
        "--test_metadata_dir",
        help="Path to directory with test metadata files",
        required=True,
    )
    main(parser.parse_args())
