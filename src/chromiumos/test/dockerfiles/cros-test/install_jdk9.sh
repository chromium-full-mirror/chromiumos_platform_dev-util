#!/bin/bash
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
set -eux

JDK_DIR=/usr/local/jdk
JDK_TAR=openjdk-9.0.4_linux-x64_bin.tar.gz
JDK_BIN=jdk-9.0.4/bin

mkdir -p "${JDK_DIR}"
cd "${JDK_DIR}"

# Download and extract the tarball. Remove tarball once done to save space.
gsutil cp "gs://chromiumos-test-assets-public/cts/${JDK_TAR}" .
tar -xvf "${JDK_TAR}"
rm -f "${JDK_TAR}"

# Symlink everything under JDK_BIN to /usr/local/bin.
# We cannot just include it in PATH because `sudo java` won't work (b/306145215)
cd "${JDK_BIN}"
for cmd in *; do
    ln -s "${JDK_DIR}/${JDK_BIN}/${cmd}" "/usr/local/bin/${cmd}"
done

# JDK 9 ships with an obsolete cacerts store from years ago. Import latest
# ca-certificates from the system to allow HTTPS to work.
# (Note: this is normally done by the ca-certificates-java package, but it
#  cannot be used with a manual JDK installation.)
yes | for cert in /etc/ssl/certs/*.pem; do
    keytool -importcert -trustcacerts -cacerts -storepass changeit \
        -file "${cert}" -alias "$(basename "${cert}")"
done
