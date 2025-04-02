# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

FROM us-docker.pkg.dev/cros-registry/base-images/ubuntu:ubuntuProd

RUN apt-get update \
    && apt-get install -y \
      ca-certificates \
      curl \
      iproute2 \
      iputils-ping \
      ssh \
      sudo \
      wget \
      xxd


# Config the user
RUN useradd -m chromeos-test
# Add passwordless sudo for chromeos-test.
# TODO, Try to remove SUDO from containers
RUN echo 'chromeos-test ALL=NOPASSWD:ALL' > /etc/sudoers.d/chromeos-test


# TODO (b/232943535): When multi-staged/layered images become a thing, this needs to be moved into a base layer as this dep will never change.
RUN mkdir -p /usr/local/dldir

# Copy CIPD ensured binaries.
# * cros-test
# * vpython3
# * gateway_go (if not public)
COPY --chown=chromeos-test:chromeos-test --chmod=755 pkgbin/ /usr/bin/

# ###############################################################################
# Everything above this point should not be changing often (ie not every build)
# Everything below this point is possible/likely to change per build
# ###############################################################################

# Copy Tast packages before Autotest as they are the same across all boards
COPY --chown=chromeos-test:chromeos-test tast/tast /usr/bin/tast
COPY --chown=chromeos-test:chromeos-test tast/remote_test_runner /usr/bin/remote_test_runner
COPY --chown=chromeos-test:chromeos-test tast/bundles/ /usr/libexec/tast/bundles
COPY --chown=chromeos-test:chromeos-test tast/data/ /usr/share/tast/data
COPY --chown=chromeos-test:chromeos-test tast/vars/ /etc/tast/vars

# Mark for Autotest/Tast to know they are within the CFT environment.
RUN touch /usr/local/f20container

# ###############################################################################
# Everything above this point should not change per BOARD
# Everything below this point is possible/likely to change per BOARD
# ###############################################################################

RUN mkdir -p /tmp/test
# Bring in metadata to 1 location (Can be different per board.)
COPY --chown=chromeos-test:chromeos-test *.pb /tmp/test/metadata/

# Configure the ssh for testing under the test user
# This **is** static between all boards/builds, but we rely on Autotest to get the SSH Key.
# Should look into investigating how we can pull this earlier in the Dockerbuild.
USER chromeos-test

RUN mkdir -p ~/.ssh

COPY --chown=chromeos-test:chromeos-test autotest/utils/frozen_chromite/ssh_keys/ /usr/local/autotest/utils/frozen_chromite/ssh_keys

RUN cp /usr/local/autotest/utils/frozen_chromite/ssh_keys/testing_rsa ~/.ssh/
RUN chmod 400 ~/.ssh/testing_rsa

COPY --chown=chromeos-test:chromeos-test ssh/ /home/chromeos-test/.ssh/
RUN chmod 400 /home/chromeos-test/.ssh/partner_testing_rsa || :

RUN  echo '\n\
Host android* brillo* chromeos1* chromeos2* chromeos3* chromeos4* chromeos5* chromeos6* chromeos7* chromeos9* autotest*.vrlab* jetstream* satlab* cros-* \n\
  GSSAPIKeyExchange no \n\
  GSSAPITrustDns no \n\
  GSSAPIDelegateCredentials no \n\
  GSSAPIAuthentication no \n\
  GSSAPIKeyExchange no \n\
  GSSAPIClientIdentity no \n\
  GSSAPIRenewalForcesRekey no \n\
  ForwardAgent no \n\
  ForwardX11 no \n\
  PreferredAuthentications publickey \n\
  StrictHostKeyChecking no \n\
  User root \n\
  IdentityFile %d/.ssh/testing_rsa \n\
  IdentityFile %d/.ssh/partner_testing_rsa \n\
\n\
Host * \n\
  StrictHostKeyChecking=no \n\
  IdentityFile %d/.ssh/testing_rsa \n\
  IdentityFile %d/.ssh/partner_testing_rsa \n\
  IdentityFile %d/.ssh/autotest_internal_rsa \n\
  UserKnownHostsFile=/dev/null \n\
'> ~/.ssh/config