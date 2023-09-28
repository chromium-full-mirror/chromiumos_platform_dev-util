# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

FROM us-docker.pkg.dev/cros-registry/test-services/base_images:ubuntuProd

RUN apt-get update \
    && apt-get install -y curl

COPY pre-process /usr/bin/