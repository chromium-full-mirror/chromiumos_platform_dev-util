# Copyright 2021 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

FROM us-docker.pkg.dev/cros-registry/test-services/base_images:ubuntuProd
COPY testplan /usr/bin/testplan
RUN chmod +x /usr/bin/testplan
ENTRYPOINT [ "/usr/bin/testplan" ]
