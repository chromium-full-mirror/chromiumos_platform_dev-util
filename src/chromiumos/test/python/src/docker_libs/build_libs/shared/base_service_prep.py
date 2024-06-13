# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Docker build Context prep for Common Services."""

import os
import shutil
import sys


# Point up a few directories to make the other python modules discoverable.
sys.path.append("../../../../")

from src.docker_libs.build_libs.shared.base_prep import BaseDockerPrepper


class BaseServicePrepper(BaseDockerPrepper):
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
        """Will ONLY cp the dockerfile + context to the build context."""
        if os.path.exists(self.full_out_dir):
            print(f"Deleting existing prepdir {self.full_out_dir}")
            shutil.rmtree(self.full_out_dir)
        os.makedirs(self.full_out_dir, exist_ok=True)

        cwd = os.path.dirname(os.path.abspath(__file__))
        src = os.path.join(
            cwd, "../../../../../", f"dockerfiles/{self.service}/"
        )
        for item in os.listdir(src):
            s = os.path.join(src, item)
            d = os.path.join(self.full_out_dir, item)
            shutil.copy2(s, d)
