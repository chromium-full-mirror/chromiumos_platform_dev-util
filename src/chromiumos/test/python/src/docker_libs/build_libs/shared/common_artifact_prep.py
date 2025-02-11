# Copyright 2022 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Common artifact prepper."""

import os
from pathlib import Path
import shutil
import sys

import setup_chromite  # pylint: disable=unused-import,import-error
from src.common import utils
import src.common.exceptions as common_exception

from chromite.lib import path_util


# Point up a few directories to make the other python modules discoverable.
sys.path.append("../../../../")


class CrosArtifactPrep:
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
        self.path = path
        self.chroot = chroot
        self.out_path = Path(out_path) if out_path else None
        self.sysroot = sysroot
        self.force_path = force_path
        self.service = service

        self.full_autotest = ""
        self.full_bin = ""
        self.full_out = path
        self.build_path = ""
        self.chroot_bin = ""
        if self.sysroot.startswith("/"):
            self.sysroot = self.sysroot[1:]

        self.config_paths()

    def prep(self):
        """Run the steps needed to prep the container artifacts."""
        raise NotImplementedError

    def config_paths(self):
        """Build up the paths needed in local mem."""
        self.build_path = path_util.FromChrootPath(
            os.path.join(os.path.sep, self.sysroot),
            chroot_path=self.chroot,
            out_path=self.out_path,
        )
        self.full_autotest = os.path.join(
            self.build_path, "usr/local/build/autotest"
        )
        self.chroot_bin = os.path.join(self.chroot, "usr/bin")
        self.full_bin = os.path.join(self.build_path, "usr/bin")
        self.validate_paths()
        self.prep_artifact_dir()

    def validate_paths(self):
        """Verify the paths generated are valid/exist."""
        if not os.path.isdir(self.full_autotest):
            if not os.path.exists(self.full_autotest):
                raise FileNotFoundError(
                    "Autotest path %s does not exist" % self.full_autotest
                )
            raise common_exception.NotDirectoryException(
                "Autotest path %s is not a directory" % self.full_autotest
            )

        if not os.path.isdir(self.build_path):
            if not os.path.exists(self.build_path):
                raise FileNotFoundError(
                    "sysroot %s does not exist" % self.build_path
                )
            raise common_exception.NotDirectoryException(
                "sysroot %s is not a directory" % self.build_path
            )

    def prep_artifact_dir(self):
        """Prepare the artifact dir. If it does not exist, create it."""
        if os.path.exists(self.full_out):
            if self.force_path:
                print(f"Deleting existing prepdir {self.full_out}")
                shutil.rmtree(self.full_out)
            else:
                raise common_exception.ConfigError(
                    "outpath %s exists and force is not set." % self.full_out
                )
        os.makedirs(self.full_out, exist_ok=True)

    def copy_service(self):
        """Copy service needed for Docker."""
        if self.service == "cros-test-cq-light":
            shutil.copy(
                os.path.join(self.chroot_bin, "cros-test"), self.full_out
            )
        else:
            shutil.copy(
                os.path.join(
                    self.out_path,
                    self.sysroot,
                    "build/broot/usr/bin",
                    self.service,
                ),
                self.full_out,
            )

    def copy_custom_service(self, service_name):
        """Copy custom service needed for Docker."""
        shutil.copy(os.path.join(self.chroot_bin, service_name), self.full_out)

    def copy_python_protos(self):
        """Copy the python proto bindings."""
        shutil.copytree(
            os.path.join(
                self.chroot, "usr/lib64/python3.11/site-packages/chromiumos"
            ),
            os.path.join(self.full_out, "chromiumos"),
        )

    def copy_dockercontext(self):
        """Copy Docker Context to the output dir."""

        # TODO, dbeckett@: this is a hardcode back up to the execution dir
        # I need to figure out a better way to do this, but nothing comes
        # to mind.
        cwd = os.path.dirname(os.path.abspath(__file__))
        src = os.path.join(
            cwd, "../../../../../", f"dockerfiles/{self.service}/"
        )
        for item in os.listdir(src):
            s = os.path.join(src, item)
            d = os.path.join(self.full_out, item)
            shutil.copy2(s, d)

    def copy_metadata(self):
        """Return the absolute path of the metadata files."""
        # Relative to build
        _BUILD_METADATA_FILES = [
            (
                "usr/local/build/autotest/autotest_metadata.pb",
                os.path.join(self.full_out, "autotest_metadata.pb"),
            ),
            (
                "usr/share/tast/metadata/local/cros.pb",
                os.path.join(self.full_out, "local_cros.pb"),
            ),
            (
                "build/share/tast/metadata/local/crosint.pb",
                os.path.join(self.full_out, "local_crosint.pb"),
            ),
            (
                "build/share/tast/metadata/local/crosint_intel.pb",
                os.path.join(self.full_out, "local_crosint_intel.pb"),
            ),
        ]

        # relative to chroot.
        _CHROOT_METADATA_FILES = [
            (
                "usr/share/tast/metadata/remote/cros.pb",
                os.path.join(self.full_out, "remote_cros.pb"),
            ),
            (
                "usr/share/tast/metadata/remote/crosint.pb",
                os.path.join(self.full_out, "remote_crosint.pb"),
            ),
            (
                "usr/share/tast/metadata/remote/crosint_intel.pb",
                os.path.join(self.full_out, "remote_crosint_intel.pb"),
            ),
        ]

        _GTEST_METADATA_PATH = "usr/local/build/gtest/"

        for f, d in _BUILD_METADATA_FILES:
            full_md_path = FromSysrootPath(self.build_path, f)
            if not os.path.exists(full_md_path):
                print("Path %s does not exist, skipping" % full_md_path)
                continue
            shutil.copyfile(full_md_path, os.path.join(self.full_out, d))

        for f, d in _CHROOT_METADATA_FILES:
            # Search the broot first.
            full_md_path = FromSysrootPath(
                self.build_path, os.path.join("build", "broot", f)
            )
            if not os.path.exists(full_md_path):
                full_md_path = FromSysrootPath(self.chroot, f)
                if not os.path.exists(full_md_path):
                    print("Path %s does not exist, skipping" % full_md_path)
                    continue
            shutil.copyfile(full_md_path, os.path.join(self.full_out, d))

        full_gtest_path = FromSysrootPath(self.build_path, _GTEST_METADATA_PATH)
        if os.path.exists(full_gtest_path):
            for f in os.listdir(full_gtest_path):
                s = os.path.join(full_gtest_path, f)
                d = os.path.join(self.full_out, f)
                shutil.copyfile(s, d)

    def copy_private_key(self):
        """Copy private keys from src to docker context."""
        cwd = os.path.dirname(os.path.abspath(__file__))
        src = os.path.join(cwd, "../../../../../../../../../../../")
        ssh_keys = os.path.join(src, "sshkeys", "partner_testing_rsa")
        os.makedirs(os.path.join(self.full_out, "ssh"), exist_ok=True)
        if not os.path.exists(ssh_keys):
            return
        shutil.copy(ssh_keys, os.path.join(self.full_out, "ssh"))

    def copy_tast_tools(self):
        """Copy tools used by tast."""
        os.makedirs(os.path.join(self.full_out, "tast-tools"), exist_ok=True)
        _FILES = [
            "usr/bin/fsck.erofs",
        ]
        for f in _FILES:
            src_path = FromSysrootPath(self.chroot, f)
            if not os.path.exists(src_path):
                print("Path %s does not exist, skipping" % src_path)
                continue
            dst_path = os.path.join(
                self.full_out, "tast-tools", os.path.basename(src_path)
            )
            shutil.copyfile(src_path, dst_path)

    def install_cipd_package(self):
        """Install and unzip cipd package."""
        cipd_package_loc = (
            "https://chrome-infra-packages.appspot.com"
            "/dl/infra/tools/cipd/linux-amd64/+/latest"
        )
        downloaded_zip_name = "cipdzip.zip"
        out, err, code = utils.run(
            f"wget -O {self.full_out}/{downloaded_zip_name} {cipd_package_loc}"
        )
        if out != "":
            print(out)
        if err != "":
            print(err)
        if code != 0:
            print("downloading cipd package failed")

        unzip_cmd = (
            f"unzip -o {self.full_out}/{downloaded_zip_name}"
            f" -d {self.full_out}/"
        )
        out, err, code = utils.run(unzip_cmd)
        if out != "":
            print(out)
        if err != "":
            print(err)
        if code != 0:
            print("unzipping cipd package failed")

    def cipd_init(self):
        """Initialize Cipd."""

        out, err, code = utils.run(
            f"{self.full_out}/cipd init -verbose -force {self.full_out}"
        )
        if out != "":
            print(out)
        if err != "":
            print(err)
        if code != 0:
            print("Initializing cipd failed")

    def cipd_install(self, package, ref, json_output_file=None):
        """Install package using cipd."""
        install_cmd = (
            f"{self.full_out}/cipd install {package} {ref}"
            f" -root {self.full_out}"
        )
        if json_output_file:
            output_loc = f"{self.full_out}/{json_output_file}"
            install_cmd = f"{install_cmd} -json-output {output_loc}"

        out, err, code = utils.run(install_cmd)
        if out != "":
            print(out)
        if err != "":
            print(err)
        if code != 0:
            print(f"downloading {package} package failed")

    def cipd_deploy(self, package, json_output_file=None):
        """Install package using cipd."""
        deploy_cmd = (
            f"{self.full_out}/cipd pkg-deploy {package} -root {self.full_out}"
        )
        if json_output_file:
            output_loc = f"{self.full_out}/{json_output_file}"
            deploy_cmd = f"{deploy_cmd} -json-output {output_loc}"

        out, err, code = utils.run(deploy_cmd)
        if out != "":
            print(out)
        if err != "":
            print(err)
        if code != 0:
            print(f"deploying {package} package failed")


def FromSysrootPath(sysroot: str, file: str):
    return os.path.join(sysroot, file)
