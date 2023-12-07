#!/usr/bin/env python3
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Ignore line too long errors as many lines are large strings over 80 characters
# pylint: disable=line-too-long

"""Generate mobly metadata for mobly_mh_test blaze targets"""

import argparse
import logging
import pathlib

# pylint: disable=import-error
from chromiumos.test.api import test_case_metadata_pb2 as tc_metadata_pb
from chromiumos.test.api import test_case_pb2 as tc_pb
from chromiumos.test.api import test_harness_pb2 as th_pb
from google.protobuf import text_format


def _test_case_exec_factory() -> th_pb.TestHarness:
    """Factory method to build TestCaseExec proto objects"""
    return tc_metadata_pb.TestCaseExec(
        test_harness=th_pb.TestHarness(mobly=th_pb.TestHarness.Mobly())
    )


def generate_test_case_metadata_list(
    input_file: pathlib.Path,
) -> tc_metadata_pb.TestCaseMetadataList:
    """Generate test case metadata list.

    Args:
        input_file: pathlib.Path object for mobly test metadata text file

    Returns:
        tc_metadata_pb.TestCaseMetadataList
    """

    # TODO(jonfan): Introduce mobly_mh_test metadata schema that matches
    # the definition in Google3 blaze mobly_mh_test targets.

    # Sample Input:
    # //testing/mobly/platforms/cros/examples:hello_world_test_adhoc_testbed
    #
    # Sample output:
    #
    # values {
    #   test_case {
    #     id {
    #       value: "mobly.hello_world_test_adhoc_testbed"
    #     }
    #     name: "hello_world_test_adhoc_testbed"
    #     tags {
    #       value: "mobly_mh_target://testing/mobly/platforms/cros/examples:hello_world_test_adhoc_testbed"
    #     }
    #   }
    #   test_case_exec {
    #     test_harness {
    #       mobly {
    #       }
    #     }
    #   }
    #   test_case_info {
    #     owners {
    #       email: "chromeos-sw-engprod@google.com"
    #     }
    #   }
    # }

    content = input_file.read_text()
    targets = content.split("\n")

    test_case_metadata_list = []

    for target in targets:
        target_parts = target.rsplit(":", 1)
        if len(target_parts) != 2:
            raise ValueError(f"Incorrect target format: '{target}'")

        test_name = target_parts[1]

        tags = [tc_pb.TestCase.Tag(value=(f"mobly_mh_target:{target}"))]
        tc_id = tc_pb.TestCase.Id(value=f"mobly.{test_name}")

        test_case_pb = tc_pb.TestCase(id=tc_id, name=test_name, tags=tags)

        owner = "chromeos-sw-engprod@google.com"
        test_case_exec_pb = _test_case_exec_factory()
        test_case_info_pb = tc_metadata_pb.TestCaseInfo(
            owners=[tc_metadata_pb.Contact(email=owner)]
        )

        test_case_metadata_pb = tc_metadata_pb.TestCaseMetadata(
            test_case=test_case_pb,
            test_case_exec=test_case_exec_pb,
            test_case_info=test_case_info_pb,
        )

        test_case_metadata_list.append(test_case_metadata_pb)

    test_case_metadata_list_pb = tc_metadata_pb.TestCaseMetadataList(
        values=test_case_metadata_list
    )

    return test_case_metadata_list_pb


def generate_metadata_text_proto(
    input_file: pathlib.Path, output_file: pathlib.Path
):
    """Generate metadata proto.

    Args:
        input_file: pathlib.Path object for mobly test metadata text file
        output_file: pathlib.Path object for storing generated metadata proto
                     Supported proto format: 'pb','textproto'
    """
    test_case_metadata_list_pb = generate_test_case_metadata_list(input_file)
    output_file.parent.mkdir(parents=True, exist_ok=True)
    proto_format = output_file.suffix
    if proto_format == ".pb":
        output_file.write_bytes(test_case_metadata_list_pb.SerializeToString())
    elif proto_format == ".textproto":
        txt_proto_str = text_format.MessageToString(
            test_case_metadata_list_pb, as_utf8=True
        )
        logging.debug("Text Proto: %s", txt_proto_str)
        output_file.write_text(txt_proto_str)
    else:
        raise argparse.ArgumentTypeError(
            f"Unknown proto format: '{proto_format}'"
        )


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


def main():
    parser = argparse.ArgumentParser(
        description="Generate Mobly harness test case metadata"
    )
    parser.add_argument(
        "--input_file",
        help="Mobly test metadata text file",
        type=_argparse_existing_file_factory,
        required=True,
    )
    parser.add_argument(
        "--output_file",
        help="Output file to write proto metadata. Supported format: 'pb','textproto'",
        type=_argparse_file_factory,
        required=True,
    )
    parser.add_argument(
        "--log_level",
        help="Log level",
        choices=["DEBUG", "INFO", "WARNING", "ERROR"],
        default="INFO",
    )
    args = parser.parse_args()
    logging.basicConfig(level=args.log_level)

    generate_metadata_text_proto(args.input_file, args.output_file)


if __name__ == "__main__":
    main()
