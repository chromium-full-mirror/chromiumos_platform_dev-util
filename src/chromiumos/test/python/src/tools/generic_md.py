"""Module to generate Mobly testcase metadata."""

# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

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

# pylint: disable=import-error,wrong-import-position
from chromiumos.test.api import test_case_metadata_pb2 as tc_metadata_pb
from chromiumos.test.api import test_case_pb2 as tc_pb
from chromiumos.test.api import test_harness_pb2 as th_pb
from google.protobuf import text_format


def dedupe_test_list(testList):
    known_names = set()
    final_test_list = []
    for test in testList:
        if test["name"] in known_names:
            continue
        known_names.add(test["name"])
        final_test_list.append(test)
    return final_test_list


def write_test_case_metadata(
    output_file: pathlib.Path, dump: bool, input_file: ""
):
    testList = []
    try:
        with open(input_file, "r", encoding="utf-8") as f:
            testList = json.load(f)
        testList = dedupe_test_list(testList)
    except FileNotFoundError:
        print(f"File not found: {input_file}")
        return
    except json.JSONDecodeError:
        print(f"Invalid JSON format in {input_file}")
        return

    vs = []
    for test in testList:
        vs.append(md_writer(test["name"], test["tags"], test["extra_info"]))
    test_case_metadata_list = tc_metadata_pb.TestCaseMetadataList(values=vs)

    # Make sure the appropriate directory structure is created.
    output_file.parent.mkdir(parents=True, exist_ok=True)
    # Write the protobuf message to the output file
    test_case_metadata_list_pb = test_case_metadata_list.SerializeToString()
    output_file.write_bytes(test_case_metadata_list_pb)

    if dump:
        print(text_format.MessageToString(test_case_metadata_list))


def md_writer(testName, tags, extra_info):
    name = testName
    tc_id = tc_pb.TestCase.Id(value=testName)
    params = []
    for tag in tags:
        params.append(tc_pb.TestCase.Tag(value=tag))

    deps = []
    test_case = tc_pb.TestCase(
        id=tc_id, name=name, tags=params, dependencies=deps
    )
    bug_component = ""

    case_info = tc_metadata_pb.TestCaseInfo(
        owners=[],
        bug_component=tc_metadata_pb.BugComponent(value=str(bug_component)),
        # DEBUG: hardcoded
        extra_info=extra_info[0],
    )
    case_exec = tc_metadata_pb.TestCaseExec(
        test_harness=th_pb.TestHarness(mobly=th_pb.TestHarness.Mobly())
    )
    return tc_metadata_pb.TestCaseMetadata(
        test_case=test_case,
        test_case_exec=case_exec,
        test_case_info=case_info,
    )


def _argparse_file_factory(path: str) -> pathlib.Path:
    """Factory method that builds a pathlib.Path object"""
    return pathlib.Path(path)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Generate CTP metadata from \
            extracted mobly test source code"
    )
    parser.add_argument(
        "--output_file",
        help="Output file to write proto metadata",
        type=_argparse_file_factory,
        required=True,
    )
    parser.add_argument(
        "--input_file",
        help="input testcase json file",
        type=_argparse_file_factory,
        required=True,
    )
    parser.add_argument(
        "--dump",
        help="Dump pretty printed protobuf to stdout. For debugging purposes",
        action="store_true",
        default=False,
        required=False,
    )

    args = parser.parse_args()
    write_test_case_metadata(args.output_file, args.dump, args.input_file)
