#!/bin/bash -e
#
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
#
# Sets up the web server on the amarisoft callbox.

# If cargo is not installed:
#   curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

systemctl stop file_server
cp file_server.service /etc/systemd/system/
cp -r file_server /root/ || exit
cargo build --manifest-path /root/file_server/Cargo.toml --release
systemctl daemon-reload
systemctl enable file_server
systemctl start file_server
