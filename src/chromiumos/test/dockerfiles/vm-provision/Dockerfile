# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

FROM ubuntu:bionic
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        libssl-dev \
        && \
    rm -rf /var/lib/apt/lists/*
COPY vm-provision /usr/bin/vm-provision
