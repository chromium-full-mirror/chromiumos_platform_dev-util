# Copyright 2025 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Docker build Context prep for vm-provision."""

import sys


# Point up a few directories to make the other python modules discoverable.
sys.path.append("../../../../")

from src.docker_libs.build_libs.shared.base_prep import (  # noqa: E402 pylint: disable=import-error,wrong-import-position
    BaseDockerPrepper,
)
from src.docker_libs.build_libs.vm_provision.container_prep import (  # noqa: E402 pylint: disable=import-error,wrong-import-position
    VMProvisionArtifactPrepper,
)


class VMProvisionDockerPrepper(BaseDockerPrepper):
    """Prep Needed files for the Test Execution Container Docker Build."""

    def __init__(
        self,
        chroot: str,
        out_path: str,
        sysroot: str,
        tags: str,
        labels: str,
        service: str,
    ):
        """@param args (ArgumentParser): .chroot, .sysroot, .path."""
        super().__init__(
            chroot=chroot,
            out_path=out_path,
            sysroot=sysroot,
            tags=tags,
            labels=labels,
            service=service,
        )

    def prep_container(self, is_public: bool = False):
        VMProvisionArtifactPrepper(
            chroot=self.chroot,
            out_path=self.out_path,
            sysroot=self.sysroot,
            path=self.full_out_dir,
            force_path=True,
        ).prep()
