# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

FROM ubuntu:focal

ENV PATH="/home/builder/luci/:$PATH:/home/builder/depot_tools/"
ENV PYTHONPATH=/home/builder/chromiumos/src/platform/dev/contrib/kernel-rebase/bisection:/home/builder/chromiumos/src/platform/dev/contrib/kernel-rebase/bisection/test

ADD "https://packages.cloud.google.com/apt/doc/apt-key.gpg" /usr/share/keyrings/cloud.google.gpg

RUN useradd -m -G users builder && \
    apt-get update -y && \
    apt-get upgrade -y && \
    apt-get install -y \
        python3-pip \
        gnupg2 \
        iproute2 \
        git \
        net-tools \
        python3 \
        curl \
        python3-virtualenv \
        python3-oauth2client \
        vim \
        sudo

RUN echo "deb http://packages.cloud.google.com/apt cloud-sdk main" \
    > /etc/apt/sources.list.d/google-cloud-sdk.list && \
    apt-key add /usr/share/keyrings/cloud.google.gpg && \
    apt-get update -y && apt-get install -y google-cloud-sdk && \
    echo "builder ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/builder

USER builder:builder
WORKDIR /home/builder

ADD --chown=builder:builder kcr-bisect.key /home/builder/kcr-bisect.key
ADD --chown=builder:builder ensure.txt /home/builder/luci/ensure.txt
ADD --chown=builder:builder bootstrap-builder /usr/local/bin/

RUN git clone https://chromium.googlesource.com/chromium/tools/depot_tools.git
WORKDIR luci
RUN cipd ensure -root . -ensure-file=ensure.txt
WORKDIR ../

RUN pip3 install --require-hashes -r requirements.txt

RUN gcloud auth activate-service-account kcr-bisection@chromeos-missing-patches.google.com.iam.gserviceaccount.com --key-file=/home/builder/kcr-bisect.key

ENTRYPOINT ["/home/builder/depot_tools/luci-auth", "context", "-service-account-json=/home/builder/kcr-bisect.key", "--", "/bin/bash"]
