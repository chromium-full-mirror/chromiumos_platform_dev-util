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

PVS_CIPD_REF="PVS_CIPD_REF"
CACHE_SERVER_CIPD_REF="CACHE_SERVER_CIPD_REF"
CTRR_CIPD_REF="CTRR_CIPD_REF"
CTR_CIPD_REF="CTR_CIPD_REF"
PROD_REF="prod"

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
        self.setup_cipd()
        self.download_pvs_bin()
        self.download_pvs_completion()
        self.download_cft_binaries()
        self.copy_pass_criteria()
        self.copy_ssh_keys()

    def setup_cipd(self):
        """Set up cipd so cipd packages can be installed"""
        self.install_cipd_package()
        self.cipd_init()

    def download_pvs_bin(self):
        self.download_cipd_artifact(
            package="chromiumos/infra/pvs/linux-amd64",
            # TODO: change the default to "prod" instead of
            # "continuous-integration" once this is being used in production
            ref=os.getenv(PVS_CIPD_REF, "continuous-integration"),
            artifact="pvs",
            artifact_dest=os.path.join(self.full_out, "pvs_bin"),
            artifact_perm=0o755,
        )

    def download_pvs_completion(self):
        self.download_cipd_artifact(
            package="chromiumos/infra/pvs/linux-amd64",
            # TODO: change the default to "prod" instead of
            # "continuous-integration" once this is being used in production
            ref=os.getenv(PVS_CIPD_REF, "continuous-integration"),
            artifact="pvs_completion",
            artifact_dest=os.path.join(self.full_out, "pvs_completion_bin"),
            artifact_perm=0o644,
        )

    def copy_pass_criteria(self):
        dest = os.path.join(self.full_out, "passcriteria")
        shutil.copytree(
            os.path.join(self.cros_root_dir(), "src/platform/pvs/passcriteria"),
            dest,
            dirs_exist_ok=True,
        )
        os.system(f"chmod -R a+r {dest}")

    def copy_ssh_keys(self):
        dest = os.path.join(self.full_out, "ssh")
        shutil.copytree(
            os.path.join(self.cros_root_dir(), "chromite/ssh_keys"),
            dest,
            dirs_exist_ok=True,
        )
        os.chmod(os.path.join(dest, 'testing_rsa'), 0o600)

    def download_cft_binaries(self):
        """Download the cft binaries and their respective cipd manifests"""

        share_dir = os.path.join(self.full_out, "share")
        os.mkdir(share_dir)
        artifact_dest_dir = os.path.join(share_dir, "cft")
        os.mkdir(artifact_dest_dir)
        manifest_dest_dir = os.path.join(share_dir, "cft_manifests")
        os.mkdir(manifest_dest_dir)

        self.download_cipd_artifact(
            package="chromiumos/infra/cros-tool-runner/linux-amd64",
            ref=os.getenv(CTR_CIPD_REF, PROD_REF),
            artifact="cros-tool-runner",
            artifact_dest=os.path.join(artifact_dest_dir, "cros-tool-runner"),
            artifact_perm=0o755,
            manifest_dest=os.path.join(manifest_dest_dir, "cros-tool-runner"),
        )
        self.download_cipd_artifact(
            package="chromiumos/infra/cros_test_runner/linux-amd64",
            ref=os.getenv(CTRR_CIPD_REF, PROD_REF),
            artifact="cros_test_runner",
            artifact_dest=os.path.join(artifact_dest_dir, "cros_test_runner"),
            artifact_perm=0o755,
            manifest_dest=os.path.join(manifest_dest_dir, "cros_test_runner"),
        )
        self.download_cipd_artifact(
            package="chromiumos/infra/cache-server/linux-amd64",
            ref=os.getenv(CACHE_SERVER_CIPD_REF, PROD_REF),
            artifact="cacheserver",
            artifact_dest=os.path.join(artifact_dest_dir, "cacheserver"),
            artifact_perm=0o755,
            manifest_dest=os.path.join(manifest_dest_dir, "cacheserver"),
        )

    def download_cipd_artifact(
        self, package, ref, artifact, artifact_dest,
        artifact_perm=None, manifest_dest=None
    ):
        """Downloads the file from the given pacakge/ref to the dest"""
        self.cipd_install(package, ref)

        # Copy the artifact into the artifact dir
        artifact_symlink_path = os.path.join(self.full_out, artifact)
        relative_artifact_path = os.readlink(artifact_symlink_path)
        abs_artifact_path = os.path.join(self.full_out, relative_artifact_path)
        if artifact_symlink_path != artifact_dest:
            shutil.copy(abs_artifact_path, artifact_dest)
        if artifact_perm:
            os.chmod(artifact_dest, artifact_perm)

        # Copy the manifest into the manifest dir
        if manifest_dest:
            absolute_manifest_path = os.path.join(
                os.path.dirname(abs_artifact_path), ".cipdpkg/manifest.json"
            )
            shutil.copy(absolute_manifest_path, manifest_dest)

    def cros_root_dir(self):
        """Path to the root of the chromiumos root directory"""
        cwd = os.path.dirname(os.path.abspath(__file__))
        return os.path.join(cwd, "../../../../../../../../../../../")

    def copy_with_perm(self, src, dest, perm):
        """Copies the src file to the dest path with the given permissions"""
        shutil.copy(src, dest)
        os.chmod(dest, perm)
