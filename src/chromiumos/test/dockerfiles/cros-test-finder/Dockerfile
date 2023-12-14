# Copyright 2021 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

FROM us-docker.pkg.dev/cros-registry/test-services/base_images:ubuntuProd
COPY cros-test-finder /usr/bin/


RUN mkdir -p /tmp/test
# Bring in metadata to 1 location
COPY *.pb /tmp/test/metadata/
COPY suite_sets /tmp/test/suite_sets.pb
COPY suites /tmp/test/suites.pb


# Own cros-test-finder & metadata
RUN chmod 755 /usr/bin/cros-test-finder
RUN chmod 644 /tmp/test/metadata/*.pb /tmp/test/*.pb