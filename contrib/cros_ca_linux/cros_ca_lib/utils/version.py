# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Utils to get and update source code version."""

import datetime
import os
import re
import shutil


def get_timestamp_ver(directory, includes=None, relative_path="."):
    """Gets the latest (newest) timestamp and corresponding file name.

    Within the given directory, ignoring files and directories listed
    in ignore_patterns.

    Args:
        directory: Path to the directory to scan.
        includes: List of dir entities to check.
        relative_path: Relative path to the directory to scan.

    Returns:
        A tuple containing the filename.
        and the latest timestamp as a datetime object, or None if no files found
        or no valid files after ignoring based on .gitignore.
    """
    ignore_patterns = ["logs", "venv", "__pycache__", r".*\.py[cod]"]

    newest_file = None
    newest_timestamp = None

    for filename in os.listdir(directory):
        if includes is not None and filename not in includes:
            continue
        # Construct full path
        filepath = os.path.join(directory, filename)

        # Check if it should be ignored based on patterns
        if any(re.match(pattern, filename) for pattern in ignore_patterns):
            continue  # Skip ignored files/directories

        # Check if it's a file or directory (consider processing)
        if os.path.isfile(filepath):
            # Get the modification time
            modification_time = os.path.getmtime(filepath)

            # Convert timestamp to datetime object
            timestamp_datetime = datetime.datetime.fromtimestamp(
                modification_time
            )

            # Update if newer file found
            if (
                newest_timestamp is None
                or timestamp_datetime > newest_timestamp
            ):
                newest_file = os.path.join(
                    relative_path, filepath.replace(directory + os.sep, "")
                )  # Relative path
                newest_timestamp = timestamp_datetime

        # Recursive call for subdirectories (if enabled)
        elif os.path.isdir(filepath):
            relative = os.path.join(
                relative_path, filepath.replace(directory + os.sep, "")
            )
            latest_in_subdirectory = get_timestamp_ver(
                filepath, relative_path=relative
            )
            if (
                latest_in_subdirectory
                and latest_in_subdirectory[1]
                and (
                    not newest_timestamp
                    or latest_in_subdirectory[1] > newest_timestamp
                )
            ):
                newest_file, newest_timestamp = latest_in_subdirectory

    return newest_file, newest_timestamp


def remove_pycache(directory):
    """Recursively removes __pycache__ folders and .pyc files.

    Args:
        directory: The path to the directory to clean.
    """

    for root, dirs, filenames in os.walk(directory):
        for filename in filenames:
            if filename.endswith(".pyc"):
                os.remove(os.path.join(root, filename))
        if "__pycache__" in dirs:
            shutil.rmtree(os.path.join(root, "__pycache__"))
