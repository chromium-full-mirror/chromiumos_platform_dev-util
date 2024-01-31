# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""
Scripts contain some common utility helper function.
"""
import os
from datetime import datetime

from benchmarking.benchmarking_exception import BenchmarkingException

CURRENT_DIR_PATH = os.path.dirname(os.path.abspath(__file__))


def remove_multiple_from_list(lst1: list, lst2: list):
    """Function used to remove multiple values exist in list from
    another list.
    """
    for element in lst2:
        if element in lst1:
            lst1.remove(element)


def check_out_dir_creation_required():
    """Function to check output with time directory that contains
    logs and results if it exists or not, if not it will be created.

    Returns:
        The path of output with time directory.
    """
    current_time = datetime.now().strftime("%Y%m%d-%H%M%S")
    out_dir_path = os.path.join(CURRENT_DIR_PATH, f"../out/{current_time}")
    if not os.path.exists(out_dir_path):
        os.makedirs(out_dir_path)
    return out_dir_path


WEB_FILE_DIR_PATH = os.path.join(CURRENT_DIR_PATH, "../resources/web_app_files")


def get_file_path_to_upload(new_doc_name, ext_type):
    """Function to return absolute file path needs to be uploaded
    from local storge after rename existing file to uuid4.

    Args:
        new_doc_name: The new name of document that be represented by uuid4
            with file type extension.
        ext_type: Office file extension type.

    Returns: return absolute file path needs to be uploaded from local storge.

    Raises:
        BenchmarkingException: in case no file with provided extension found.
    """
    files_name = os.listdir(WEB_FILE_DIR_PATH)
    for file_name in files_name:
        if file_name.endswith(ext_type):
            file_name_path = os.path.abspath(
                WEB_FILE_DIR_PATH + rf"/{file_name}"
            )
            new_doc_name_path = os.path.abspath(
                WEB_FILE_DIR_PATH + rf"/{new_doc_name}"
            )
            os.rename(file_name_path, new_doc_name_path)
            break
    else:
        raise BenchmarkingException(
            f"Cannot find any file ends with extension type = {ext_type}"
            f" in WEB_FILE_DIR_PATH = {WEB_FILE_DIR_PATH}"
        )
    return new_doc_name_path
