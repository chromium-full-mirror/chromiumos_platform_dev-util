#!/usr/bin/env python3
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Generate create_test_ready.jsonpb for checking DUT readiness for testing."""

import argparse
import hashlib
import os
import sys

from chromiumos.test.api import cros_test_ready_cli_pb2 as test_ready_pb
from google.protobuf.json_format import MessageToJson
import six


# If running in Autotest dir, keep this.
os.environ["PY_VERSION"] = "3"

# NOTE: this MUST be run in Python3, if we get configured back to PY2, exit.
if six.PY2:
    sys.exit(1)


def autotest_checksum_dirs():
    """Return a mapping between source and destination paths of directories.

    This function return a mapping between source and destination paths
    of autotest directories.
    """

    return {
        "/usr/local/autotest": "/usr/local/autotest",
        # Test images for modlab builds have different autotest path.
        "/usr/local/autodir": "/usr/local/autodir",
    }


def default_checksum_files():
    """Return a mapping between source and destination paths of files.

    For test image only sources that are not in /user/local will be
    moved to /user/local in a DUT.
    """

    return {
        # Autotest related files in regular test images.
        "/usr/local/autotest/bin/autotest":
        "/usr/local/autotest/bin/autotest",

        "/usr/local/autotest/tools/autotest":
        "/usr/local/autotest/tools/autotest",
        # Test images for modlab builds have different autotest path.
        "/usr/local/autodir/bin/autotest":
        "/usr/local/autodir/bin/autotest",

        "/usr/local/autodir/tools/autotest":
        "/usr/local/autodir/tools/autotest",

        # Tast related files.
        "/usr/libexec/tast/bundles/local/cros":
        "/usr/local/libexec/tast/bundles/local/cros",

        "/usr/bin/local_test_runner":
        "/usr/local/bin/local_test_runner",
    }


def parse_local_arguments(args):
    """Parse the CLI."""
    parser = argparse.ArgumentParser(
        description="Create config file for checking if a DUT is test ready."
    )
    parser.add_argument(
        "-src_root",
        dest="src_root",
        default=None,
        help="Path to root directory of the source files or directories.",
    )
    parser.add_argument(
        "-output_file",
        dest="output_file",
        default=None,
        help="Where to write the cros_test_ready.jsonpb.",
    )
    return parser.parse_args(args)


def md5(filename):
    with open(filename, "rb") as f:
        content = f.read()
        readable_hash = hashlib.md5(content).hexdigest()
    return readable_hash


def serialize_checksum_from_file(src, dest):
    """Calculate checksum of a file."""
    try:
        md5_value = md5(src)
        return test_ready_pb.CrosTestReadyConfig.KeyValue(
            key=dest, value=md5_value
        )
    except Exception as e:
        raise Exception(
            "Failed to calculate checksum for file %s: %s" % (src, e)
        )


def serialize_autotest_checksum_from_dir(src_dir, dest_dir):
    """Calculate checksums of non-hidden _python_ files under a directory."""
    results = []
    try:
        for root, _, filenames in os.walk(src_dir):
            for filename in filenames:
                # Ignore hidden files.
                if filename.startswith("."):
                    continue
                # Only check for python files for autotest.
                if not filename.endswith(".py"):
                    continue
                src_file_path = os.path.join(root, filename)
                dest_file_path = src_file_path.replace(src_dir, dest_dir)
                results.append(
                    serialize_checksum_from_file(src_file_path, dest_file_path)
                )
    except Exception as e:
        raise Exception(
            "Failed to calculate checksum for files under %s: %s" % (src_dir, e)
        )
    return results


def serialize_checksum(src_root):
    """Return a serialized TestCaseInfo obj."""
    checksums = []
    filenames = default_checksum_files()
    for src, dest in filenames.items():
        src_path = src_root + src
        if not os.path.exists(src_path):
            continue
        checksums.append(serialize_checksum_from_file(src_path, dest))
    # Calculate checksum for python files under autotest directories.
    dirs = autotest_checksum_dirs()
    for src_dir, dest_dir in dirs.items():
        src_path = src_root + src_dir
        if not os.path.exists(src_path):
            continue
        checksums_for_dir = serialize_autotest_checksum_from_dir(
            src_path, dest_dir
        )
        checksums = checksums + checksums_for_dir
    return test_ready_pb.CrosTestReadyConfig(checksums=checksums)


def main():
    """Generate the metadata, and if an output path is given, save it."""
    args = parse_local_arguments(sys.argv[1:])
    serialized = serialize_checksum(args.src_root)
    json_obj = MessageToJson(serialized)

    if args.output_file:
        with open(args.output_file, "w", encoding="utf-8") as wf:
            wf.write(json_obj)


if __name__ == "__main__":
    main()
