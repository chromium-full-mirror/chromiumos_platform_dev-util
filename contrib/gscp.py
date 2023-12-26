#!/usr/bin/env python3
# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Transfers file via gs bucket.

A scp like utility script to transfer via gs bucket, which might be faster for
large files, especially when the ssh connection is over some complicated relays.
"""

import argparse
import json
import logging
import os
import pathlib
import shlex
import subprocess
import sys
from typing import List, Optional
import urllib.parse
import urllib.request
import uuid


def get_luci_token() -> str:
    scopes = [
        "https://www.googleapis.com/auth/userinfo.email",
        "https://www.googleapis.com/auth/devstorage.read_only",
    ]
    cmd = [
        "luci-auth",
        "token",
        "-scopes",
        " ".join(scopes),
    ]
    return subprocess.check_output(cmd, text=True)


def get_object_path() -> str:
    user = os.getlogin()
    unique_id = uuid.uuid4()
    return f"{user}/{unique_id}.gscp"


def get_access_boundary(bucket: str, object_path: str) -> dict:
    avail_res = f"//storage.googleapis.com/projects/_/buckets/{bucket}"
    prefix = f"projects/_/buckets/{bucket}/objects/{object_path}"
    prefix_cond = f"resource.name.startsWith('{prefix}')"
    return {
        "accessBoundary": {
            "accessBoundaryRules": [
                {
                    "availablePermissions": [
                        "inRole:roles/storage.objectViewer"
                    ],
                    "availableResource": avail_res,
                    "availabilityCondition": {
                        "expression": prefix_cond,
                    },
                }
            ]
        }
    }


def downscope_token(luci_token: str, access_boundary: dict) -> str:
    url = "https://sts.googleapis.com/v1/token"
    data = {
        "grant_type": "urn:ietf:params:oauth:grant-type:token-exchange",
        "subject_token_type": "urn:ietf:params:oauth:token-type:access_token",
        "requested_token_type": "urn:ietf:params:oauth:token-type:access_token",
        "subject_token": luci_token,
        "options": json.dumps(access_boundary),
    }

    data = urllib.parse.urlencode(data).encode("utf-8")
    req = urllib.request.Request(url, data=data, method="POST")

    with urllib.request.urlopen(req) as res:
        res_data = json.load(res)
        return res_data["access_token"]


def upload_file_to_gs(file: pathlib.Path, bucket: str, object_path: str):
    assert file.is_file()
    dst = f"gs://{bucket}/{object_path}"
    cmd = [
        "gsutil",
        "cp",
        str(file),
        dst,
    ]
    subprocess.check_call(cmd)


def download_file_remotely(
    target_host: str,
    target_path: str,
    source_path: pathlib.Path,
    token: str,
    bucket: str,
    object_path: str,
):
    header = f"Authorization: Bearer {token}"
    url = f"https://storage.googleapis.com/{bucket}/{object_path}"
    remote_sh = f"""
        # Do not quote so things like `~` would expand.
        output={target_path}
        if [[ -d "$output" ]]; then
            output=$output/{shlex.quote(source_path.name)}
        fi

        curl \
            --progress-bar \
            --header {shlex.quote(header)} \
            --output "$output" \
            {shlex.quote(url)}
    """

    # TODO(shik): Warmup the ssh connection earlier to reduce overhead.
    cmd = [
        "ssh",
        target_host,
        remote_sh,
    ]
    subprocess.check_call(cmd)


def remove_file_from_gs(bucket: str, object_path: str):
    dst = f"gs://{bucket}/{object_path}"
    cmd = [
        "gsutil",
        "rm",
        dst,
    ]
    subprocess.check_call(cmd)


def main(argv: Optional[List[str]] = None) -> Optional[int]:
    parser = argparse.ArgumentParser()
    parser.add_argument("source")
    parser.add_argument("target")
    parser.add_argument("--debug", action="store_true")
    args = parser.parse_args(argv)

    log_level = logging.DEBUG if args.debug else logging.INFO
    log_format = "%(asctime)s - %(levelname)s - %(funcName)s: %(message)s"
    logging.basicConfig(level=log_level, format=log_format)

    source_path = pathlib.Path(args.source)
    target_host, target_path = args.target.split(":", 1)

    luci_token = get_luci_token()
    logging.debug("Got luci_token")

    bucket = "chromeos-throw-away-bucket"
    object_path = get_object_path()
    access_boundary = get_access_boundary(bucket, object_path)

    downscoped_token = downscope_token(luci_token, access_boundary)
    logging.debug("Got downscoped_token")

    upload_file_to_gs(source_path, bucket, object_path)
    logging.debug("Uploaded file to gs")

    download_file_remotely(
        target_host,
        target_path,
        source_path,
        downscoped_token,
        bucket,
        object_path,
    )
    logging.debug("Downloaded file remotely")

    remove_file_from_gs(bucket, object_path)
    logging.debug("Removed file from gs")


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
