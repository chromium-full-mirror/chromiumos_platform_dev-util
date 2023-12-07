#!/usr/bin/env python3
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Ignore line too long errors as many lines are large strings over 80 characters
# pylint: disable=line-too-long

# Ignore protected member access as these are unit tests explicitly accessing
# pylint: disable=protected-access

"""Unit tests for generate_mobly_metadata.py script"""

import collections
import unittest
from unittest import main
from unittest import mock

# pylint: disable=import-error
from chromiumos.test.api import test_case_metadata_pb2 as tc_metadata_pb
import generate_mobly_metadata
from google.protobuf import text_format


class GenerateMoblyMetadataTest(unittest.TestCase):
    """Main test class for these unit tests"""

    @classmethod
    def setUpClass(cls) -> None:
        """Setup test cases"""
        Txt_TxtProto = collections.namedtuple(
            "Txt_TxtProto", ["txt", "textproto"]
        )
        cls._test_case_data = [
            Txt_TxtProto(
                VALID_SINGLE_CASE_SIMPLE,
                VALID_SINGLE_CASE_SIMPLE_TEXT_PROTO,
            ),
        ]

    def _generate_mobly_metadata(
        self,
        input_file: mock.MagicMock,
        output_file: mock.MagicMock,
    ) -> bytearray:
        generate_mobly_metadata.generate_metadata_text_proto(
            input_file=input_file,
            output_file=output_file,
        )
        proto_format = output_file.suffix
        if proto_format == ".pb":
            return output_file.write_bytes.call_args[0][0]
        else:
            return output_file.write_text.call_args[0][0]

    def test_generation_text_proto(self) -> None:
        """Test metadata text proto generation for a given file"""

        for tc in self._test_case_data:
            input_file = mock.MagicMock()
            input_file.read_text.return_value = tc.txt
            output_file = mock.MagicMock()
            output_file.suffix = ".textproto"
            actual_text_proto = self._generate_mobly_metadata(
                input_file, output_file
            )
            expected_text_proto = tc.textproto
            self.assertEqual(actual_text_proto, expected_text_proto)

    def test_generation_binary_proto(self) -> None:
        """Test metadata binary proto generationfor a given file"""

        for tc in self._test_case_data:
            input_file = mock.MagicMock()
            input_file.read_text.return_value = tc.txt
            output_file = mock.MagicMock()
            output_file.suffix = ".pb"

            actual_proto_bytes = self._generate_mobly_metadata(
                input_file, output_file
            )
            actual_proto_message = tc_metadata_pb.TestCaseMetadataList()
            actual_proto_message.ParseFromString(actual_proto_bytes)
            expected_text_proto = tc.textproto
            expected_proto_message = text_format.Parse(
                expected_text_proto, tc_metadata_pb.TestCaseMetadataList()
            )
            self.assertEqual(actual_proto_message, expected_proto_message)


# Test case definitions
VALID_SINGLE_CASE_SIMPLE = """//testing/mobly/platforms/cros/examples:hello_world_test_adhoc_testbed
//testing/mobly/platforms/cros/examples:advanced_features_example_test"""
VALID_SINGLE_CASE_SIMPLE_TEXT_PROTO = """values {
  test_case {
    id {
      value: "mobly.hello_world_test_adhoc_testbed"
    }
    name: "hello_world_test_adhoc_testbed"
    tags {
      value: "mobly_mh_target://testing/mobly/platforms/cros/examples:hello_world_test_adhoc_testbed"
    }
  }
  test_case_exec {
    test_harness {
      mobly {
      }
    }
  }
  test_case_info {
    owners {
      email: "chromeos-sw-engprod@google.com"
    }
  }
}
values {
  test_case {
    id {
      value: "mobly.advanced_features_example_test"
    }
    name: "advanced_features_example_test"
    tags {
      value: "mobly_mh_target://testing/mobly/platforms/cros/examples:advanced_features_example_test"
    }
  }
  test_case_exec {
    test_harness {
      mobly {
      }
    }
  }
  test_case_info {
    owners {
      email: "chromeos-sw-engprod@google.com"
    }
  }
}
"""

if __name__ == "__main__":
    main(verbosity=2)
