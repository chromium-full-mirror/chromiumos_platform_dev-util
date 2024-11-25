#!/usr/bin/env python3
# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Generate xTS metadata proto from extracted metadata."""

import argparse
import json
import os
import pathlib
import sys


# Used to import the proto stack.
if "CONFIG_REPO_ROOT" in os.environ:
    sys.path.insert(1, os.path.join(os.getenv("CONFIG_REPO_ROOT"), "python"))
else:
    sys.path.insert(
        1,
        str(
            pathlib.Path(__file__).parent.resolve()
            / "../../../../../../../../config/python"
        ),
    )

from chromiumos.test.api import test_case_metadata_pb2 as tc_metadata_pb
from chromiumos.test.api import test_case_pb2 as tc_pb
from chromiumos.test.api import test_harness_pb2 as th_pb
from google.protobuf import text_format


def _test_case_factory(
    case_data: dict,
    suite_name: str,
    branch: str,
    targets: list,
    build_id: str,
    target_builds: list = [],
) -> tc_metadata_pb.TestCaseMetadata:
    """Factory method for build TestCase proto objects

    Args:
        case_data: dict representing the 'modules' JSON data
        suite_name: string representing the 'suite' JSON field
        branch: string representing the 'branch' JSON field
        targets: list of string representing 'targets' JSON field
        build_id: string representing the 'build_id' JSON field
        target_builds: list of target/build_id/abi objects
    """
    name = f'{suite_name}.{case_data["name"]}'
    tc_id = tc_pb.TestCase.Id(value=f"tradefed.{name}")
    params = [
        tc_pb.TestCase.Tag(value=f"param:{t}") for t in case_data["parameters"]
    ]
    params.extend(
        [
            tc_pb.TestCase.Tag(value=f"suite:{suite_name}"),
            tc_pb.TestCase.Tag(value=f"branch:{branch}"),
            tc_pb.TestCase.Tag(value=f"build_id:{build_id}"),
        ]
    )
    params.extend([tc_pb.TestCase.Tag(value=f"target:{t}") for t in targets])
    if len(target_builds) > 0:
        params.extend(
            [
                tc_pb.TestCase.Tag(
                    value=f"target_build=target:{tb['target']},"
                    f"build_id:{tb['build_id']},"
                    f"abi:{tb['abi']}"
                )
                for tb in target_builds
            ]
        )
    else:
        params.extend(
            [
                tc_pb.TestCase.Tag(value=f"abi:{t}")
                for t in case_data.get("abis", [])
            ]
        )
    test_case = tc_pb.TestCase(
        id=tc_id, name=name, tags=params, dependencies=[]
    )

    bug_component = (
        case_data.get("bug_components", [])[0]
        if case_data.get("bug_components", None)
        else ""
    )
    case_info = tc_metadata_pb.TestCaseInfo(
        owners=[
            tc_metadata_pb.Contact(email=x) for x in case_data.get("owners", [])
        ],
        bug_component=tc_metadata_pb.BugComponent(value=str(bug_component)),
    )

    case_exec = tc_metadata_pb.TestCaseExec(
        test_harness=th_pb.TestHarness(tradefed=th_pb.TestHarness.Tradefed())
    )

    return tc_metadata_pb.TestCaseMetadata(
        test_case=test_case, test_case_exec=case_exec, test_case_info=case_info
    )


def _test_case_list_factory(input_data: dict) -> list:
    """Factory method for batch building TestCaseMetadata objects"""
    version = input_data["version"]
    suite_name = input_data["suite"]
    branch = input_data["branch"]
    targets = input_data["targets"]
    build_id = input_data["build_id"]

    if version == "1.5":
        tb = input_data["target_builds"]
        test_case_metadata = [
            _test_case_factory(tc, suite_name, branch, targets, build_id, tb)
            for tc in input_data["modules"]
        ]
    else:
        test_case_metadata = [
            _test_case_factory(tc, suite_name, branch, targets, build_id)
            for tc in input_data["modules"]
        ]

    return test_case_metadata


def main(input_files: list, output_file: pathlib.Path, dump: bool):
    """Main entry point

    Args:
        input_files: list of pathlib.Path objects for yaml metadata files
        output_file: pathlib.Path object for storing generated proto metadata
        dump: if True, write the final proto in text format to stdout
    """
    test_case_metadata = []

    for f in input_files:
        input_data = json.loads(f.read_text())
        test_case_metadata.extend(_test_case_list_factory(input_data))

    test_case_metadata_list = tc_metadata_pb.TestCaseMetadataList(
        values=test_case_metadata
    )

    # Make sure the appropriate directory structure is created.
    output_file.parent.mkdir(parents=True, exist_ok=True)
    # Write the protobuf message to the output file
    output_file.write_bytes(test_case_metadata_list.SerializeToString())

    if dump:
        print(text_format.MessageToString(test_case_metadata_list))


def _argparse_existing_file_factory(path: str) -> pathlib.Path:
    """Create path objects

    Verify the files specified on the command line against:
        1) The file exists
        2) The file is a file, not a directory or other type

    Args:
        path: path to file specified (str)
    """
    p = pathlib.Path(path)
    if not p.exists():
        raise argparse.ArgumentTypeError(
            f"The specified file '{path}' does not exist!"
        )

    if not p.is_file():
        raise argparse.ArgumentTypeError(
            f"The specified file '{path}' is not a file!"
        )

    return p


def _argparse_file_factory(path: str) -> pathlib.Path:
    """Factory method that builds a pathlib.Path object"""
    return pathlib.Path(path)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Generate CTP metadata from extracted xTS metadata JSON"
    )
    parser.add_argument(
        "--output_file",
        help="Output file to write proto metadata",
        type=_argparse_file_factory,
        required=True,
    )
    parser.add_argument(
        "files",
        help="xTS metadata files",
        metavar="INPUT_JSON",
        type=_argparse_existing_file_factory,
        nargs="+",
    )
    parser.add_argument(
        "--dump",
        help="Dump pretty printed protobuf to stdout. For debugging purposes",
        action="store_true",
        default=False,
        required=False,
    )

    args = parser.parse_args()
    main(args.files, args.output_file, args.dump)
