# Copyright 2022 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Prep everything for for the cros-test Docker Build Context."""

import os
import sys


sys.path.append("../../../../")

from src.docker_libs.build_libs.shared.common_artifact_prep import (  # noqa: E402 pylint: disable=import-error,wrong-import-position,import-modules-only
    CrosArtifactPrep,
)
from src.tools import (  # noqa: E402 pylint: disable=import-error,wrong-import-position
    container_util,
)


class CrosTestArtifactPrep(CrosArtifactPrep):
    """Prep Needed files for the Test Execution Container Docker Build."""

    def __init__(
        self,
        path: str,
        chroot: str,
        out_path: str,
        sysroot: str,
        force_path: bool,
        service: str,
    ):
        """@param args (ArgumentParser): .chroot, .sysroot, .path."""
        super().__init__(
            path=path,
            chroot=chroot,
            out_path=out_path,
            sysroot=sysroot,
            force_path=force_path,
            service=service,
        )

    def prep(self, is_public: bool = False):
        """Run the steps needed to prep the container artifacts."""
        self.create_tarball()
        self.copy_metadata()
        self.copy_python_protos()
        self.copy_dockercontext()
        self.untar()
        self.remove_unused_deps()
        self.remove_tarball()
        self.copy_private_key()
        self.copy_tast_tools()
        self.setup_cipd()
        self.ensure_packages(is_public)

    def create_tarball(self):
        """Copy the Stuff."""
        builder = container_util.AutotestTarballBuilder(
            os.path.dirname(self.full_autotest),
            self.full_out,
            self.chroot,
            self.out_path,
            self.sysroot,
        )

        builder.BuildFullAutotestandTastTarball()

    def untar(self):
        """Untar the package prior to depl."""
        os.system(
            f"tar -xvf {self.full_out}/"
            f"autotest_server_package.tar.bz2 -C {self.full_out} > /dev/null"
        )

    def remove_unused_deps(self):
        UNUSED = ["chrome_test", "telemetry_dep"]
        for dep in UNUSED:
            os.system(f"rm -r {self.full_out}/autotest/client/deps/{dep}")

    def remove_tarball(self):
        """Remove the autotest tarball post untaring."""
        os.system(f"rm -r {self.full_out}/autotest_server_package.tar.bz2")

    def setup_cipd(self):
        """Set up cipd so cipd packages can be installed"""
        self.install_cipd_package()
        self.cipd_init()

    def ensure_packages(self, is_public=False):
        ensure_pkgs_path = os.path.join(self.full_out, "pkgbin")
        ensure_file = os.path.join(self.full_out, "ensure.txt")
        with open(ensure_file, "w", encoding="utf-8") as f:
            f.write(
                "chromiumos/infra/cft/execution/cros-test/linux-amd64 prod\n"
            )
            f.write("infra/tools/luci/vpython3/linux-amd64 latest\n")
            if not is_public:
                f.write(
                    "chromeos_internal/infra/tools/chromeos_omnilab_gateway "
                    + "latest\n"
                )
        os.system(
            f"cipd ensure "
            f"-ensure-file {ensure_file} "
            f"-root {ensure_pkgs_path}"
        )
