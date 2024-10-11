#!/usr/bin/env python3
# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Generate mobly metadata proto from parsed repositories."""

import argparse
import ast
import io
import os
import pathlib
import sys
import tempfile
from typing import List
import urllib.request as urllib
import zipfile


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


class TestMethod:
    """Methods on a class and its method as an ast."""

    def __init__(self, parentClass: ast.ClassDef, method: ast.FunctionDef):
        self.parentClass = parentClass
        self.method = method

    def to_metadata(self) -> tc_metadata_pb.TestCaseMetadata:
        name = f"{self.parentClass.name}.{self.method.name}"
        tc_id = tc_pb.TestCase.Id(value=f"mobly.{name}")
        params = []
        params.extend(
            [
                tc_pb.TestCase.Tag(value=f"module:{self.parentClass.name}"),
                # TODO: Parse out a true suite name.
                tc_pb.TestCase.Tag(value="suite:betocq"),
            ]
        )
        deps = []
        test_case = tc_pb.TestCase(
            id=tc_id, name=name, tags=params, dependencies=deps
        )
        bug_component = ""
        case_info = tc_metadata_pb.TestCaseInfo(
            owners=[],
            bug_component=tc_metadata_pb.BugComponent(value=str(bug_component)),
        )
        case_exec = tc_metadata_pb.TestCaseExec(
            test_harness=th_pb.TestHarness(mobly=th_pb.TestHarness.Mobly())
        )
        return tc_metadata_pb.TestCaseMetadata(
            test_case=test_case,
            test_case_exec=case_exec,
            test_case_info=case_info,
        )


def findTestMethodsFromSource(source: str) -> List[TestMethod]:
    with tempfile.TemporaryDirectory() as tmpdir:
        pullFromSource(source, tmpdir)
        return findTestMethodsFromDirectory(tmpdir)


def findTestMethodsFromDirectory(directory: str) -> List[TestMethod]:
    testMethods = []
    for dirpath, _, filenames in os.walk(directory):
        for filename in filenames:
            if filename.endswith(".py"):
                testMethods.extend(
                    findTestMethodsFromFile(os.path.join(dirpath, filename))
                )
    return testMethods


def findTestMethodsFromFile(filepath: str) -> List[TestMethod]:
    testMethods = []
    with open(filepath, encoding="utf-8") as file:
        node = ast.parse(file.read())

    classes = [n for n in node.body if isinstance(n, ast.ClassDef)]
    for klass in classes:
        methods = [n for n in klass.body if isinstance(n, ast.FunctionDef)]
        testMethods = [
            TestMethod(klass, method)
            for method in methods
            if method.name.startswith("test_")
        ]

    return testMethods


def writeTestCaseMetadata(
    testMethods: List[TestMethod], output_file: pathlib.Path, dump: bool
):
    test_case_metadata_list = tc_metadata_pb.TestCaseMetadataList(
        values=[testMethod.to_metadata() for testMethod in testMethods]
    )

    # Make sure the appropriate directory structure is created.
    output_file.parent.mkdir(parents=True, exist_ok=True)
    # Write the protobuf message to the output file
    output_file.write_bytes(test_case_metadata_list.SerializeToString())

    if dump:
        print(text_format.MessageToString(test_case_metadata_list))


def pullFromSource(url: str, tmpdir: str):
    with urllib.urlopen(url) as response:
        with zipfile.ZipFile(io.BytesIO(response.read())) as zipped_file:
            zipped_file.extractall(path=tmpdir)


def main(output_file: pathlib.Path, sources: List[str], dump: bool):
    testMethods = []
    for source in sources:
        testMethods.extend(findTestMethodsFromSource(source))
    writeTestCaseMetadata(testMethods, output_file, dump)


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
        "--sources",
        help="",
        required=True,
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
    main(args.output_file, args.sources, args.dump)
