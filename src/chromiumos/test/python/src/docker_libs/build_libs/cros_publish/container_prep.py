# Copyright 2022 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Prep everything for the cros publish Docker Build Context."""

import os
import shutil
import sys

from src.docker_libs.build_libs.shared import (  # noqa: E402 pylint: disable=import-error,wrong-import-position
    common_artifact_prep,
)
from src.tools import (  # noqa: E402 pylint: disable=import-error,wrong-import-position
    container_util,
)


sys.path.append("../../../../")


class CrosPublishArtifactPrep(common_artifact_prep.CrosArtifactPrep):
    """Prep Needed files for the Cros Publish Container Docker Build."""

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
            service="cros-publish",
        )

    def prep(self):
        """Run the steps needed to prep the container artifacts."""
        # Copy all required cros-publish services
        self.copy_custom_service("tko-publish")
        self.setup_cipd()
        self.ensure_packages()

        # Copy CPCon upload dependencies
        self.copy_cpcon_upload()

        self.create_tarball()
        self.copy_dockercontext()
        self.untar()
        self.remove_extra()
        self.remove_tarball()

    def untar(self):
        """Untar the package prior to depl."""
        os.system(
            f"tar -xvf {self.full_out}/"
            f"autotest_server_package.tar.bz2 -C {self.full_out} > /dev/null"
        )

    def remove_extra(self):
        """Removes not required autotest code from the bundle."""
        UNUSED = [
            "autotest/client/deps/",
            "autotest/client/site_tests/",
            "autotest/moblab/*",
            "autotest/server/site_tests/",
            "autotest/test_suites/",
            "autotest/frontend/client/src/autotest/*",
        ]
        for fn in UNUSED:
            os.system(f"rm -r {self.full_out}/{fn}")

    def create_tarball(self):
        """Copy the Stuff."""
        builder = container_util.AutotestTarballBuilder(
            os.path.dirname(self.full_autotest),
            self.full_out,
            self.chroot,
            self.out_path,
            self.sysroot,
            tko_only=True,
        )

        builder.BuildFullAutotestandTastTarball()

    def remove_tarball(self):
        """Remove the autotest tarball post untaring."""
        os.system(f"rm -r {self.full_out}/autotest_server_package.tar.bz2")

    def copy_cpcon_upload(self):
        """Copy upload_results for CPCon upload"""
        cwd = os.path.dirname(os.path.abspath(__file__))
        src = os.path.join(cwd, "../../../../../../../../../../")
        autotest_contrib = os.path.join(
            src, "third_party", "autotest", "files", "contrib"
        )
        shutil.copytree(autotest_contrib, self.full_out + "/contrib/")
        shutil.copy(
            os.path.join(
                src,
                "config",
                "python",
                "chromiumos",
                "test",
                "artifact",
                "tko_pb2.py",
            ),
            self.full_out,
        )

    def setup_cipd(self):
        """Set up cipd so cipd packages can be installed"""
        self.install_cipd_package()
        self.cipd_init()

    def ensure_packages(self):
        ensure_pkgs_path = os.path.join(self.full_out, "pkgbin")
        ensure_file = os.path.join(self.full_out, "ensure.txt")
        with open(ensure_file, "w", encoding="utf-8") as f:
            f.write(
                "chromiumos/infra/cft/publish/gcs-publish/linux-amd64 prod\n"
            )
            f.write(
                "chromiumos/infra/cft/publish/rdb-publish/linux-amd64 prod\n"
            )
            f.write(
                "chromiumos/infra/cft/publish/cpcon-publish/linux-amd64 prod\n"
            )
            f.write("infra/tools/result_adapter/linux-amd64 prod\n")
            f.write("infra/tools/rdb/linux-amd64 latest\n")
        os.system(
            f"cipd ensure "
            f"-ensure-file {ensure_file} "
            f"-root {ensure_pkgs_path}"
        )
