# Copyright 2022 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Prep everything for for the cros-test Docker Build Context."""

import os
import shutil
import sys


sys.path.append("../../../../")

from src.docker_libs.build_libs.shared import (  # noqa: E402 pylint: disable=import-error,wrong-import-position
    common_artifact_prep,
)


class CrosTestFinderArtifactPrep(common_artifact_prep.CrosArtifactPrep):
    """Prep Needed files for the Test Execution Container Docker Build."""

    def __init__(
        self,
        path: str,
        chroot: str,
        out_path: str,
        sysroot: str,
        force_path: bool,
    ):
        """@param args (ArgumentParser): .chroot, .sysroot, .path."""
        super().__init__(
            path=path,
            chroot=chroot,
            out_path=out_path,
            sysroot=sysroot,
            force_path=force_path,
            service="cros-test-finder",
        )

    def prep(self):
        """Run the steps needed to prep the container artifacts."""
        self.copy_service()
        shutil.copy(
            os.path.join(self.chroot_bin, "test_finder_filter"), self.full_out
        )
        self.copy_metadata()
        self.copy_dockercontext()
        self.copy_centralized_suites()

    def copy_centralized_suites(self):
        csuite_dir = "usr/share/centralized-suites"
        shutil.copyfile(
            os.path.join(self.build_path, csuite_dir, "suites.pb"),
            os.path.join(self.full_out, "suites"),
        )
        shutil.copyfile(
            os.path.join(self.build_path, csuite_dir, "suite_sets.pb"),
            os.path.join(self.full_out, "suite_sets"),
        )
