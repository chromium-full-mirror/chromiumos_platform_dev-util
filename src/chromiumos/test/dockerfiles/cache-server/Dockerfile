# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

FROM us-docker.pkg.dev/cros-registry/base-images/ubuntu:ubuntuProd
COPY cacheserver /usr/bin/cacheserver
RUN mkdir -p /etc/ssl/certs
# ca-certificates.crt to be obtained from existing docker images
COPY ca-certificates.crt /etc/ssl/certs