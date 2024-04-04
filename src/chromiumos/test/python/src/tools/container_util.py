# Copyright 2022 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Utility to tar Autotest and Tast artifacts from given path to given path."""

import dataclasses
import fnmatch
import os
import pathlib
import sys
from typing import List, Optional


sys.path.insert(1, str(pathlib.Path(__file__).parent.resolve() / "../../"))

from src.tools import cmd_util  # noqa: E402


COMP_NONE = 0
COMP_GZIP = 1
COMP_BZIP2 = 2
COMP_XZ = 3


def _FindSourceRoot():
    """Try and find the root check out of the chromiumos tree"""
    source_root = path = os.path.realpath(
        os.path.join(os.path.abspath(__file__), "..", "..", "..")
    )
    while True:
        if os.path.isdir(os.path.join(path, ".repo")):
            return path
        elif path == "/":
            break
        path = os.path.dirname(path)
    return source_root


SOURCE_ROOT = _FindSourceRoot()


def FromChrootPath(chroot, file):
    return os.path.join(chroot, file)


def FromSysrootPath(sysroot, file):
    return os.path.join(sysroot, file)


def FindFilesMatching(pattern, target="./", cwd=os.curdir, exclude_dirs=[]):
    """Search the root directory recursively for matching filenames.

    The |target| and |cwd| args allow manipulating how the found paths are
    returned as well as specifying where the search needs to be executed.

    If our filesystem only has /path/to/example.txt, and our pattern is '*.txt':
    |target|='./', |cwd|='/path'  =>  ./to/example.txt
    |target|='to', |cwd|='/path'  =>  to/example.txt
    |target|='./', |cwd|='/path/to' =>  ./example.txt
    |target|='/path'        =>  /path/to/example.txt
    |target|='/path/to'       =>  /path/to/example.txt

    Args:
      pattern: the pattern used to match the filenames.
      target: the target directory to search.
      cwd: current working directory.
      exclude_dirs: Directories to not include when searching.

    Returns:
      A list of paths of the matched files.
    """
    assert cwd
    assert os.path.exists(cwd)

    # Backup the current working directory before changing it.
    old_cwd = os.getcwd()
    os.chdir(cwd)

    matches = []
    for directory, _, filenames in os.walk(target):
        if any(directory.startswith(e) for e in exclude_dirs):
            # Skip files in excluded directories.
            continue

        for filename in fnmatch.filter(filenames, pattern):
            matches.append(os.path.join(directory, filename))

    # Restore the working directory.
    os.chdir(old_cwd)

    return matches


# Directory within _SERVER_PACKAGE_ARCHIVE where Tast files needed to run
# with Server-Side Packaging are stored.
_TAST_SSP_SUBDIR = "tast"


@dataclasses.dataclass(frozen=True)
class PathMapping:
    """Container for mapping a source path to a destination."""

    raw_src: str
    raw_dst: Optional[str] = None

    def get_src(self, chroot: str, out: str, sysroot: str) -> str:
        """Get the source path for this mapping."""
        if self.raw_src.startswith("/"):
            # It's a chroot path.  See if it's in the per-board broot first.
            path = FromChrootPath(
                out,
                os.path.join(
                    sysroot, "build", "broot", self.raw_src.lstrip("/")
                ).lstrip("/"),
            )
            if not os.path.exists(path):
                path = FromChrootPath(chroot, self.raw_src.lstrip("/"))
            return str(path)
        else:
            # It's a source tree path.
            return os.path.join(SOURCE_ROOT, self.raw_src)

    def get_dst(self) -> str:
        """Get the destination path for this mapping."""
        if self.raw_dst is not None:
            return self.raw_dst
        return os.path.join(_TAST_SSP_SUBDIR, os.path.basename(self.raw_src))


class AutotestTarballBuilder(object):
    """Builds autotest tarballs for testing."""

    # Archive file names.
    _SERVER_PACKAGE_ARCHIVE = "autotest_server_package.tar.bz2"

    # Tast files and directories to include in AUTOTEST_SERVER_PACKAGE relative
    # to the build root.
    _TAST_SSP_CHROOT_FILES = [
        # Main Tast executable.
        PathMapping("/usr/bin/tast"),
        # Runs remote tests.
        PathMapping("/usr/bin/remote_test_runner"),
        # Dir containing test bundles.
        PathMapping("/usr/libexec/tast/bundles"),
        # Dir containing test data.
        PathMapping("/usr/share/tast/data"),
        # Secret variables.
        PathMapping("/etc/tast/vars"),
    ]
    # Tast files and directories stored in the source code.
    _TAST_SSP_SOURCE_FILES = [
        # Helper script to run SSP tast.
        PathMapping("src/platform/tast/tools/run_tast.sh"),
        # Public variables first.
        PathMapping(
            "src/platform/tast-tests/vars",
            "tast/vars/public",
        ),
        # Secret variables last.
        PathMapping(
            "src/platform/tast-tests-private/vars",
            "tast/vars/private",
        ),
    ]

    def __init__(
        self,
        archive_basedir,
        output_directory,
        chroot_path,
        out_path,
        sysroot_path,
        tko_only=False,
    ):
        """Init function.

        Args:
          archive_basedir (str): The base directory from which the archives
            will be created. This path should contain the `autotest`
            directory.
          output_directory (str): The directory where the archives will be
            written.
          chroot_path (str): Path to the chroot fs.
          out_path (str): Path to the output state.
          sysroot_path (str): Path to the sysroot.
          tko_onky (bool): Flag to only bundle tko & required content.
        """
        self.archive_basedir = archive_basedir
        self.output_directory = output_directory
        self.chroot_path = chroot_path
        self.out_path = out_path
        self.sysroot_path = sysroot_path
        self.tko_only = tko_only

    def BuildFullAutotestandTastTarball(self):
        """Tar all needed Autotest & Tast files required by Docker Container.

        NOTE: This does not include autotest/packages, but will include all
        tests and client/deps.

        Returns:
          str|None - The path of the autotest server package tarball if
            created.
        """
        if not self.tko_only:
            tast_files, transforms = self._GetTastServerFilesAndTarTransforms()
        else:
            tast_files, transforms = [], []

        autotest_files = FindFilesMatching(
            "*",
            target="autotest",
            cwd=self.archive_basedir,
            exclude_dirs=("autotest/packages",),
        )

        tarball = os.path.join(
            self.output_directory, self._SERVER_PACKAGE_ARCHIVE
        )
        if self._BuildTarball(
            autotest_files + tast_files, tarball, extra_args=transforms
        ):
            return tarball
        else:
            return None

    def _BuildTarball(self, input_list, tarball_path, extra_args=None):
        """Tar and zip files and directories from input_list to tarball_path.

        Args:
          input_list: A list of files and directories to be archived.
          tarball_path: Path of output tar archive file.
          extra_args: extra arguments to pass to CreateTarball.

        Returns:
          Return value of CreateTarball.
        """
        for pathname in input_list:
            if os.path.exists(os.path.join(self.archive_basedir, pathname)):
                break
        else:
            # If any of them exist we can create an archive, but if none
            # do then we need to stop. For now, since we either pass in a
            # handful of directories we don't necessarily check, or actually
            # search the filesystem for lots of files, this is far more
            # efficient than building out a list of files that do exist.
            return None
        compressor = COMP_BZIP2
        chroot = self.chroot_path

        return cmd_util.CreateTarball(
            tarball_path,
            self.archive_basedir,
            compression=compressor,
            chroot=chroot,
            inputs=input_list,
            extra_args=extra_args,
        )

    def _GetTastServerFilesAndTarTransforms(self):
        """Returns Tast server files and corresponding tar transform flags.

        The returned paths should be included in AUTOTEST_SERVER_PACKAGE. The
        --transform arguments should be passed to GNU tar to convert the paths
        to appropriate destinations in the tarball.

        Returns:
            (files, transforms), where files is a list of absolute paths to Tast
            server files/directories and transforms is a list of --transform
            arguments to pass to GNU tar when archiving those files.
        """
        files = []
        transforms = []

        for mapping in self._GetTastSspFiles():
            src = mapping.get_src(
                self.chroot_path, self.out_path, self.sysroot_path
            )
            if not os.path.exists(src):
                continue

            files.append(src)
            transforms.append(
                "--transform=s|^%s|%s|"
                % (os.path.relpath(src, "/"), mapping.get_dst())
            )

        return files, transforms

    def _GetTastSspFiles(self) -> List[PathMapping]:
        """Build out the paths to the tast SSP files.

        Returns:
            The paths to the files.
        """
        return self._TAST_SSP_CHROOT_FILES + self._TAST_SSP_SOURCE_FILES
