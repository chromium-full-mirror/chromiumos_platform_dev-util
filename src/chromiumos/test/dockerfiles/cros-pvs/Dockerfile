# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

FROM python:3.9.16-alpine

# Set environment variables
ENV USE_DOCKERIZED_PVS=1
ENV PVS_BIN_DIR=/usr/bin
ENV PVS_LIBEXEC_DIR=/usr/libexec
ENV PVS_SHARE_DIR=/usr/share/pvs
ENV COMPLETION_DIR=/usr/share/bash-completion/completions
ENV CROS_WORKON_SRCROOT=${PVS_SHARE_DIR}/chromiumos
ENV AUTOTEST_REPO_ROOT=${PVS_SHARE_DIR}/autotest
ENV PASS_CRITERIA_DIR=${PVS_SHARE_DIR}/pass_criteria
ENV SSH_CONFIG_DIR=/home/pvs/.ssh
ENV UPLOAD_RESULTS_BRANCH=release-R113-15393.B
ENV PY_VERSION=3
ENV HISTCONTROL=ignoreboth
ENV PVS_OUTPUT_DIR=/home/pvs/.pvs
ENV UPLOAD_CONFIG_DIR=${PVS_OUTPUT_DIR}/upload_config
ENV HISTFILE=${PVS_OUTPUT_DIR}/.bash_history

# Install gosu to allow user to be the same both inside and outside the container
RUN echo "@testing http://nl.alpinelinux.org/alpine/edge/testing" >> /etc/apk/repositories && apk add --update git bash bash-completion less shadow gosu@testing

# Install upload_results.py dependencies
RUN python3 -m pip install google.cloud google.cloud.pubsub google.cloud.storage \
    jsonrpclib protobuf==3.19.3 chardet

# Download the config proto files and autotest repo we need these to run the
# upload_results.py script in the container
WORKDIR ${CROS_WORKON_SRCROOT}/src/config
RUN git clone --depth=1 https://chromium.googlesource.com/chromiumos/config -b ${UPLOAD_RESULTS_BRANCH} .
WORKDIR ${AUTOTEST_REPO_ROOT}
RUN git clone --depth=1 https://chromium.googlesource.com/chromiumos/third_party/autotest -b ${UPLOAD_RESULTS_BRANCH} .

# Copy pvs/cft binaries/dependencies from host to container
COPY ssh/ ${SSH_CONFIG_DIR}
COPY passcriteria/ ${PASS_CRITERIA_DIR}
COPY share/ ${PVS_SHARE_DIR}
COPY pvs ${PVS_BIN_DIR}/pvs
RUN ${PVS_BIN_DIR}/pvs completion bash > ${COMPLETION_DIR}/pvs

# Build all the pvs artifacts on container initialization
WORKDIR /home/pvs
RUN chmod -R 777 /home/pvs

ENTRYPOINT ["/bin/sh", "-c", "useradd -u $LOCAL_UID -o -m pvs && groupmod -g $LOCAL_GID -n $LOCAL_GROUP pvs && exec /usr/bin/gosu pvs /bin/bash"]