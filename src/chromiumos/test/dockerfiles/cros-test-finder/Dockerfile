# Copyright 2021 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

FROM us-docker.pkg.dev/cros-registry/base-images/ubuntu:ubuntuProd
COPY cros-test-finder /usr/bin/
COPY test_finder_filter /usr/bin/


RUN mkdir -p /tmp/test
# Bring in metadata to 1 location
COPY *.pb /tmp/test/metadata/
COPY *.txtpb /tmp/test/metadata/
COPY suite_sets /tmp/test/suite_sets.pb
COPY suites /tmp/test/suites.pb


# Own cros-test-finder/test_finder_filter & metadata
RUN chmod 755 /usr/bin/cros-test-finder
RUN chmod 755 /usr/bin/test_finder_filter

RUN chmod 644 /tmp/test/metadata/*.pb /tmp/test/*.pb /tmp/test/metadata/*.txtpb
