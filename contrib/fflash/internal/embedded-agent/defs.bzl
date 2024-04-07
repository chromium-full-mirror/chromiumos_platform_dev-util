# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

load("@bazel_skylib//rules:run_binary.bzl", "run_binary")

def zstd_compress(name, src, out):
    run_binary(
        name = name,
        srcs = [src],
        outs = [out],
        args = [
            "--force",
            "-18",
            "$(location {})".format(src),
            "-o",
            "$(location {})".format(out),
        ],
        tool = "@zstd//:zstd_cli",
    )
