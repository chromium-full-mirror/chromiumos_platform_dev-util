# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Prep everything for for the cros-pvs Docker Build Context."""

import os
import shutil
import sys

from src.docker_libs.build_libs.shared.common_artifact_prep import (  # noqa: E402 pylint: disable=import-error,wrong-import-position
    CrosArtifactPrep,
)


sys.path.append("../../../../")

PVS_CIPD_REF = "PVS_CIPD_REF"
CACHE_SERVER_CIPD_REF = "CACHE_SERVER_CIPD_REF"
CTRR_CIPD_REF = "CTRR_CIPD_REF"
CTR_CIPD_REF = "CTR_CIPD_REF"
PVS_LOCAL_PKG_PATH = "PVS_LOCAL_PKG_PATH"
CTRR_LOCAL_PKG_PATH = "CTRR_LOCAL_PKG_PATH"
CTR_LOCAL_PKG_PATH = "CTR_LOCAL_PKG_PATH"
DEFAULT_PVS_CIPD_REF = "latest"
PROD_REF = "prod"


class CrosPvsArtifactPrep(CrosArtifactPrep):
    """Prep Needed files for the PVS Container Docker Build."""

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
            service="cros-pvs",
        )

    def prep(self):
        """Run the steps needed to prep the container artifacts."""
        self.copy_dockercontext()
        share_dir = self.create_share_dir()
        self.setup_cipd()
        self.download_pvs_bin(share_dir)
        self.download_cft_binaries(share_dir)
        self.copy_ssh_keys()

    def create_share_dir(self):
        share_dir = os.path.join(self.full_out, "share")
        os.mkdir(share_dir)
        os.chmod(share_dir, 0o755)
        return share_dir

    def setup_cipd(self):
        """Set up cipd so cipd packages can be installed"""
        self.install_cipd_package()
        self.cipd_init()

    def download_pvs_bin(self, share_dir):
        self.download_cipd_artifact(
            package="infra_internal/tools/pvs/pvs/linux-amd64",
            ref=os.getenv(PVS_CIPD_REF, DEFAULT_PVS_CIPD_REF),
            local_path=os.getenv(PVS_LOCAL_PKG_PATH),
            artifact="pvs",
            artifact_dest=os.path.join(self.full_out, "pvs"),
            artifact_perm=0o755,
            version_file=".versions/pvs.cipd_version",
            version_file_dest=os.path.join(share_dir, "pvs.cipd_version"),
        )

    def copy_ssh_keys(self):
        dest = os.path.join(self.full_out, "ssh")
        shutil.copytree(
            os.path.join(self.cros_root_dir(), "chromite/ssh_keys"),
            dest,
            dirs_exist_ok=True,
        )
        os.chmod(os.path.join(dest, "testing_rsa"), 0o600)

    def download_cft_binaries(self, share_dir):
        """Download the cft binaries and their respective cipd version files"""
        artifact_dest_dir = os.path.join(share_dir, "cft")
        os.mkdir(artifact_dest_dir)
        os.chmod(artifact_dest_dir, 0o755)

        self.download_cipd_artifact(
            package="chromiumos/infra/cros-tool-runner/linux-amd64",
            ref=os.getenv(CTR_CIPD_REF, PROD_REF),
            local_path=os.getenv(CTR_LOCAL_PKG_PATH),
            artifact="cros-tool-runner",
            artifact_dest=os.path.join(artifact_dest_dir, "cros-tool-runner"),
            artifact_perm=0o755,
            version_file=".versions/cros-tool-runner.cipd_version",
            version_file_dest=os.path.join(
                artifact_dest_dir, "cros-tool-runner.cipd_version"
            ),
        )
        self.download_cipd_artifact(
            package="chromiumos/infra/cros_test_runner/linux-amd64",
            ref=os.getenv(CTRR_CIPD_REF, PROD_REF),
            local_path=os.getenv(CTRR_LOCAL_PKG_PATH),
            artifact="cros_test_runner",
            artifact_dest=os.path.join(artifact_dest_dir, "cros_test_runner"),
            artifact_perm=0o755,
            version_file=".versions/cros_test_runner.cipd_version",
            version_file_dest=os.path.join(
                artifact_dest_dir, "cros_test_runner.cipd_version"
            ),
        )
        self.download_cipd_artifact(
            package="chromiumos/infra/cache-server/linux-amd64",
            ref=os.getenv(CACHE_SERVER_CIPD_REF, PROD_REF),
            local_path=None,
            artifact="cacheserver",
            artifact_dest=os.path.join(artifact_dest_dir, "cacheserver"),
            artifact_perm=0o755,
            version_file=".versions/cache-server.cipd_version",
            version_file_dest=os.path.join(
                artifact_dest_dir, "cache-server.cipd_version"
            ),
        )

    def download_cipd_artifact(
        self,
        package,
        ref,
        local_path,
        artifact,
        artifact_dest,
        artifact_perm=None,
        version_file=None,
        version_file_dest=None,
    ):
        """Downloads the file from the given pacakge/ref to the dest"""
        if local_path:
            self.cipd_deploy(local_path)
        else:
            self.cipd_install(package, ref)

        # Copy the artifact into the artifact dir
        artifact_symlink_path = os.path.join(self.full_out, artifact)
        relative_artifact_path = os.readlink(artifact_symlink_path)
        abs_artifact_path = os.path.join(self.full_out, relative_artifact_path)
        os.remove(artifact_symlink_path)
        shutil.copy(abs_artifact_path, artifact_dest)
        if artifact_perm:
            os.chmod(artifact_dest, artifact_perm)

        # Copy the versionfile to its dest
        if version_file and version_file_dest:
            absolute_version_file_path = os.path.join(
                os.path.dirname(abs_artifact_path), version_file
            )
            shutil.copy(absolute_version_file_path, version_file_dest)

    def cros_root_dir(self):
        """Path to the root of the chromiumos root directory"""
        cwd = os.path.dirname(os.path.abspath(__file__))
        return os.path.join(cwd, "../../../../../../../../../../../")

    def copy_with_perm(self, src, dest, perm):
        """Copies the src file to the dest path with the given permissions"""
        shutil.copy(src, dest)
        os.chmod(dest, perm)
