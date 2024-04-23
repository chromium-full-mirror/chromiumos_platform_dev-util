# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The paramiko utilities used by the paramiko host."""

import errno
import os
import select
import stat

from cros_ca_lib.utils import logging as lg


logger = lg.logger


def read_outputs_with_timeout(stdout, stderr, timeout):
    stdout_data = []
    stderr_data = []
    while True:
        ready, _, _ = select.select(
            [stdout.channel, stderr.channel], [], [], timeout
        )
        if stdout.channel in ready:
            data = stdout.channel.recv(1024)
            if not data:  # EOF indicates process might have finished
                break
            stdout_data.append(data.decode("utf-8").strip())
        elif stderr.channel in ready:
            data = stderr.channel.recv(1024)
            if not data:  # EOF indicates process might have finished
                break
            stderr_data.append(data.decode("utf-8").strip())
        else:
            break
    return stdout_data, stderr_data


def walk_sftp(sftp, path):
    """Walks through a directory tree yielding each directory and file.

    Args:
        sftp: An open SFTP client object.
        path: The path to the directory to start walking from.

    Yields:
        A tuple containing:
            - directory (str): The full path of the current directory.
            - subdirs (list): A list of subdirectory names within the current
                              directory.
            - files (list): A list of filenames within the current directory.
    """
    try:
        # Get directory listing with attributes
        listdir_attr = sftp.listdir_attr(path)
    except IOError as e:
        logger.warning("Error listing directory '%s': %e", path, e)
        return

    # Separate directory entries and files
    directories = []
    files = []
    for entry in listdir_attr:
        # Skip "." and ".." entries
        if entry.filename in (".", ".."):
            continue
        # Check if entry is a directory or file based on its attributes
        if stat.S_ISDIR(entry.st_mode):
            directories.append(entry.filename)
        else:
            files.append(entry.filename)

    yield path, directories, files

    # Recursively walk through subdirectories
    for subdir in directories:
        full_subdir_path = os.path.join(path, subdir)
        yield from walk_sftp(sftp, full_subdir_path)


def download_dir(sftp, remote_path, local_path):
    """Downloads a directory and creates it locally.

    Args:
        sftp: The SFTP client.
        remote_path: The path to the directory on the remote server.
        local_path: The path to create the local directory (will be created if
                    it doesn't exist).
    """
    # Create the local directory (if it doesn't exist)
    os.makedirs(local_path, exist_ok=True)

    for dirpath, _, filenames in walk_sftp(sftp, remote_path):
        # Create local subdirectories within the local_path
        local_dir = os.path.join(
            local_path, os.path.relpath(dirpath, remote_path)
        )
        os.makedirs(local_dir, exist_ok=True)

        # Download files
        for filename in filenames:
            remote_file = os.path.join(dirpath, filename)
            local_file = os.path.join(local_dir, filename)
            sftp.get(remote_file, local_file)


def remove_directory(sftp, remote_path):
    """Removes a directory from the remote SSH host using SFTP.

    Args:
        sftp: An established SFTP client connection.
        remote_path: The path to the directory on the remote SSH host.
    """

    for filename in sftp.listdir(remote_path):
        file_path = os.path.join(remote_path, filename)

        if filename in (".", ".."):
            continue  # Skip special entries (. and ..)

        # Check if it's a directory
        try:
            attr = sftp.lstat(file_path)
        except OSError as e:
            if e.errno != errno.ENOENT:  # Handle errors except "file not found"
                raise e
            else:
                # File might have been deleted concurrently, continue
                continue

        # If the file doesn't have write permission, change the file mode.
        if not stat.S_IWRITE & attr.st_mode:
            sftp.chmod(file_path, 0o755)
        if stat.S_ISDIR(attr.st_mode):
            # Recursively remove subdirectories
            remove_directory(sftp, file_path)
        else:
            # Remove the file
            sftp.remove(file_path)
    sftp.rmdir(remote_path)
