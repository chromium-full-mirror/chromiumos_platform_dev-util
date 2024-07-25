#!/usr/bin/env python3
# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""This is a tool that creates merge CLs for Intel Core drop driver chnages."""

import argparse
import random

import git


# We require GitPython for this script to run properly
REMOTENAME = "iwl7000"
REMOTEURL = "git://git.kernel.org/pub/scm/linux/kernel/git/iwlwifi/chromeos.git"
GERRITREMOTEBRANCH = "refs/heads/intel-wifi"
CHROMEOSREMOTENAME = "cros"

glob_changeid = "I" + "".join(
    random.choice("0123456789abcdef") for _ in range(40)
)


class CoreDropHelper:
    """Includes all needed functions to create merge CLs."""

    def __init__(
        self,
        repo,
        kernel_version,
        merge_version,
        bug_num=197914719,
        name="Billy Zhao",
        email="billyzhao@chromium.org",
        cq_dep="chromium:2757212",
        changeid="None",
    ):
        self.repo = repo
        self.kv = kernel_version
        self.mv = merge_version
        self.bug_num = bug_num
        self.name = name
        self.email = email
        self.cq_dep = cq_dep
        self.changeid = changeid if changeid is not None else glob_changeid

    @property
    def gerrit_branch_name(self):
        """Returns the branch name.

        The branch name is the copy of the intel's core drop that should
        be uploaded to gerrit on cros.

        Example branchname: refs/heads/intel-wifi/v4.14-Core63-66
        """
        return f"{GERRITREMOTEBRANCH}/v{self.kv}-Core{self.mv}"

    @property
    def local_mirror_name(self):
        """Emits the branch name for the local mirror of Intel's remote branch.

        Example branchname:  iwl7000/v4.14-core63-66
        """
        core = "core" + self.mv
        return f"{REMOTENAME}/v{self.kv}-{core}"

    @property
    def local_mirror_sha(self):
        """Emits the git sha hash of the local_mirror_name."""

        return self.repo.git.rev_parse(self.local_mirror_name)

    @property
    def merge_branch_name(self):
        """Emits the branch name used for the merge commit.

        Example branchname: iwl7000/v4.14-core63-66-merge
        """
        return self.local_mirror_name + "-merge"

    @property
    def remote_branch_name(self):
        """Emits the branch name for the target branch on Intel's remote.

        This formatting may need to be adjusted if Intel updates their
        naming convention.

        Example branchname: chromeos-5.10__release/core63-66
        """
        core = "core" + self.mv
        return f"chromeos-{self.kv}__release/{core}"

    @property
    def remote_branch_sha(self):
        """Emits the git sha hash of the remote_branch_name."""

        return self.repo.git.rev_parse(
            REMOTENAME + "/" + self.remote_branch_name
        )

    @property
    def cros_kernel_branch_name(self):
        """Returns the kernel branch name for review.

        Example chromeos-4.14
        """
        return "chromeos-" + self.kv

    @property
    def cros_kernel_branch_sha(self):
        """Emits the git sha for the cros kernel branch name."""

        return self.repo.git.rev_parse(self.cros_main_branch)

    @property
    def test_platform(self):
        platform_map = {
            "5.4": "volteer/zork",
            "5.10": "brask,sarien,rammus,atlas",
            "5.15": "puff,hatch",
            "6.1": "rex,dedede",
            "6.6": "brya, nissa",
        }
        return platform_map[self.kv]

    @property
    def cros_main_branch(self):
        """The branch name that should be an alias for m/main (kernel specific)

        Example cros/chromeos-4.14
        """
        return CHROMEOSREMOTENAME + "/" + self.cros_kernel_branch_name

    def fetch_update_branch(self):
        """Clones the upstream core drop branch into a local mirror branch.

        Resulting branch name should be in the form of
        "iwl7000/v4.14-core63-66"

        Args:
            self.repo: self.repository object from GitPython
            kernel_version: kernel version number (e.g 4.14, 5.4, 5.10)
            merge_version: Core drop version and revision number (e.g 63-66)
        """

        print(f"Checking if remote,{REMOTENAME},exists.")
        remote = None
        try:
            remote = self.repo.remote(REMOTENAME)
            print(f"Found {REMOTENAME} in remotes.")
        except ValueError:
            print(f"Could not find {REMOTENAME} in remotes.")

        if not remote:
            print(f"Adding remote {REMOTENAME}.")
            remote = self.repo.create_remote(REMOTENAME, REMOTEURL)

        print(f"Fetching updates from {REMOTEURL}.")
        remote.fetch()

        lmn = self.local_mirror_name
        print(f"Checking to see if branch {lmn} exists.")
        try:
            branch = self.repo.heads[
                [r.name for r in self.repo.heads].index(lmn)
            ]
            print(f"Branch {branch} exists!")
            print(f"checking out to {lmn}.")
        except ValueError:
            print(f"Branch {lmn} does not exist.")
            print(f"Checking out to {self.cros_main_branch}.")
            self.repo.git.checkout(self.cros_main_branch)
            print(f"Creating branch {lmn}.")
            self.repo.git.checkout("-b", lmn)

        rbn = self.remote_branch_name
        print(
            f"Setting upstream target for branch {lmn} to {REMOTENAME}/{rbn}."
        )
        self.repo.git.branch("-u", f"{REMOTENAME}/{rbn}")
        print(f"Fetching from {REMOTENAME}/{rbn}")
        self.repo.git.fetch(REMOTENAME, rbn)
        print("Resetting hard to FETCH_HEAD")
        self.repo.git.reset("--hard", "FETCH_HEAD")

    def push_refs_to_gerrit(self):
        """Pushes the core drop to a branch on gerrit.

        This prevents gerrit from choking on the core drop.
        We should only push the refs if we're on the local mirror branch.

        In case something goes wrong, we can delete the branch here:
        https://chromium-review.googlesource.com/admin/self.repos/
        chromiumos%252Fthird_party%252Fkernel,branches/q/filter:intel-wifi

        Args:
            self.repo: self.repository object from GitPython
            kernel_version: kernel version number (e.g 4.14, 5.4, 5.10)
            merge_version: Core drop version and revision number (e.g 63-66)
        """
        print("Pushing refs to gerrit.")
        lmn = self.local_mirror_name
        if self.repo.active_branch.name != lmn:
            print(
                f"Active branch is not {lmn}, please fetch update branch or"
                f" swap to {lmn} first."
            )
            return
        print(f"{lmn}:{self.gerrit_branch_name}")
        self.repo.git.push(
            CHROMEOSREMOTENAME,
            f"{lmn}:{self.gerrit_branch_name}",
            "-o",
            "uploadvalidator~skip",
            "-o",
            f"push-justification=b/{self.bug_num}",
        )

    def generate_merge_commit_message(self):
        mergebase = self.repo.git.merge_base(
            f"{CHROMEOSREMOTENAME}/{self.cros_kernel_branch_name}",
            self.local_mirror_name,
        )
        shortlog = self.repo.git.shortlog(
            mergebase + "..." + self.local_mirror_sha
        )
        core = "core" + self.mv
        message = f"""CHROMIUM: iwl7000: Merge "{core}" driver updates

This is a merge commit of all Intel patches for the "{core}" driver
update since commit "{mergebase}"
and ending at commit "{self.remote_branch_sha}".

The original branch provided by Intel (Luca Coelho) is at branch
{self.remote_branch_name} on
https://git.kernel.org/cgit/linux/kernel/git/iwlwifi/chromeos.git and
has been mirrored as {self.gerrit_branch_name} on
{CHROMEOSREMOTENAME}.

Below is the complete shortlog of all the merged patches:
{shortlog}

BUG=b:{self.bug_num}
TEST=wifi_matfunc/wifi_perf on {self.test_platform}
Signed-off-by: {self.name} <{self.email}>

Cq-Depend: {self.cq_dep}
Change-Id: {self.changeid}
        """
        return message

    def create_merge(self):
        if self.merge_branch_name in [r.name for r in self.repo.branches]:
            print(
                f"Merge branch {self.merge_branch_name} already"
                " exists, quitting."
            )
            return False
        if self.repo.active_branch.name != self.local_mirror_name:
            print(
                f"Active branch is not {self.local_mirror_name}, please"
                " fetch update branch or swap to"
                f" {self.local_mirror_name} first."
            )
            return False
        if self.local_mirror_sha != self.remote_branch_sha:
            print(
                f"Update branch {self.local_mirror_name} not in sync with"
                f" remote branch {self.remote_branch_name}. Fetch"
                " update firsts. Hashes differ"
            )
            return False

        print(f"Checking out to merge branch {self.merge_branch_name}")
        self.repo.git.checkout("-b", self.merge_branch_name)
        print(
            f"Setting branch upstream to {CHROMEOSREMOTENAME}"
            f"/{self.cros_kernel_branch_name}"
        )
        self.repo.git.branch(
            "-u",
            f"{CHROMEOSREMOTENAME}/{self.cros_kernel_branch_name}",
        )
        print(
            f"Resetting hard to {CHROMEOSREMOTENAME}"
            f"/{self.cros_kernel_branch_name}"
        )
        self.repo.git.reset(
            "--hard",
            f"{CHROMEOSREMOTENAME}/{self.cros_kernel_branch_name}",
        )

        # We have had issues using the default recursive merge strategy which
        # selects the wrong parents and causes the merge to look very strange
        # on gerrit.
        print(
            f"Merging {self.local_mirror_name} into"
            f" {self.repo.active_branch.name} using subtree strategy"
        )
        self.repo.git.merge("-s", "subtree", self.local_mirror_name)

        print("Generating merge commit message.")
        message = self.generate_merge_commit_message()
        print("Committing with generated commit message.")
        self.repo.git.commit("--amend", "-m", message)
        return True

    def verify_merge(self):
        print("Checking that merge branch exists")
        if self.merge_branch_name not in [r.name for r in self.repo.branches]:
            print(
                f"Failed verification for kernel {self.kv},"
                f" merge branch {self.merge_branch_name} does"
                " not exist."
            )
            return False

        print("Checking that merge branch has correct parents")
        res = self.repo.git.show(
            "--summary", '--format="%P"', self.merge_branch_name
        )
        res = res.split('"')[1].split(" ")  # filter out the hashes
        print(*res)
        print(self.cros_kernel_branch_sha, self.local_mirror_sha)
        if (
            res[0] != self.cros_kernel_branch_sha
            or res[1] != self.local_mirror_sha
        ):
            print(res[0], res[1])
            print(
                f"Failed verification for kernel {self.kv}, not a merge commit"
                " or merge does not have right parents."
            )
            return False

        print("Checking that merge branch has all commits from both parents")
        parentA_commit_diff = (
            f"{self.merge_branch_name}..{self.cros_main_branch}"
        )
        parentB_commit_diff = (
            f"{self.merge_branch_name}..{self.local_mirror_name}"
        )
        if (
            self.repo.git.log(parentA_commit_diff) != ""
            or self.repo.git.log(parentB_commit_diff) != ""
        ):
            print(
                self.repo.git.log(parentA_commit_diff),
                self.repo.git.log(parentB_commit_diff),
            )
            print(
                f"Failed verification for kernel {self.kv},"
                " merge may be missing some"
                " commits from either or both parents."
            )
            return False

        diffpath = "drivers/net/wireless/iwl7000"
        print(
            f"Checking that merge branch has no diffs with"
            f" upstream {REMOTENAME} for"
            f" path {diffpath}"
        )
        if (
            self.repo.git.diff(
                self.local_mirror_name + ".." + self.merge_branch_name, diffpath
            )
            != ""
        ):
            print(
                "Failed verification for kernel {0}, iwl7000 doesn't"
                " match with upstream."
            )

        print("Merge looks good!")
        return True

    def clean(self):
        print("Cleaning up the repo.")
        self.repo.git.clean("-f", "-d")
        print(f"Checking out to {self.cros_main_branch}")
        self.repo.git.checkout(self.cros_main_branch)
        for head in self.repo.heads:
            self.repo.delete_head(head, "--force")

    def run_kernel_merge(self):
        self.clean()
        self.fetch_update_branch()
        self.push_refs_to_gerrit()
        if not self.create_merge():
            print("Merge failed, skipping validation.")
            return False
        if not self.verify_merge():
            print("Merge validation failed.")
            return False
        print("Merge validation succeeded.")
        return True


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--change_id",
        help="Fixed changeid for each kernel. Will generate a single random"
        " ChangeID if not specified",
    )
    parser.add_argument(
        "--operation",
        choices=[
            "run_kernel_merge",
            "fetch_update_branch",
            "push_refs_to_gerrit",
            "create_merge",
            "verify_merge",
            "clean",
        ],
    )
    parser.add_argument("--bug_id", help="buganizer number (e.g 123456)")
    parser.add_argument("--name", help="Your Gerrit Name (e.g Billy Zhao)")
    parser.add_argument(
        "--email", help="Your Gerrit Email (e.g billyzhao@chromium.org)"
    )
    parser.add_argument(
        "--cq_dep",
        help="Cq Dep commit ID for the ebuild change that usually goes"
        " with the core drop (e.g 54321)",
    )
    parser.add_argument(
        "merge_version", help="merge version number (e.g 63-66)"
    )
    parser.add_argument(
        "kernel_versions", nargs="+", help="kernel version number (e.g 4.14)"
    )
    args = parser.parse_args()
    print(args)
    flags = {}
    for kernel in args.kernel_versions:
        flags[kernel] = False
        print(f"Starting merge for kernel {kernel}.")
        repository = git.Repo("v" + kernel)
        cdh = CoreDropHelper(
            repository,
            kernel_version=kernel,
            merge_version=args.merge_version,
            bug_num=args.bug_id,
            name=args.name,
            email=args.email,
            cq_dep=args.cq_dep,
            changeid=args.change_id,
        )
        try:
            if args.operation is None or args.operation == "run_kernel_merge":
                print("Running kernel merge routine.")
                if cdh.run_kernel_merge():
                    print(f"Kernel {kernel} merged successfully!")
                    flags[kernel] = True
                else:
                    print(f"Kernel {kernel} merged failed!")
            elif args.operation == "fetch_update_branch":
                print("Fetching update branch")
                flags[kernel] = True
                cdh.fetch_update_branch()
            elif args.operation == "push_refs_to_gerrit":
                print("Pushing refs to gerrit")
                cdh.push_refs_to_gerrit()
            elif args.operation == "create_merge":
                print("Creating merge")
                if cdh.create_merge():
                    print(f"Kernel {kernel} merge created successfully!")
                    flags[kernel] = True
                else:
                    print(f"Kernel {kernel} merge created unsuccessfully!")
            elif args.operation == "verify_merge":
                print("Validating merge")
                if cdh.verify_merge():
                    print(f"Kernel {kernel} merge validated successfully!")
                    flags[kernel] = True
                else:
                    print(f"Kernel {kernel} merge validated unsuccessfully!")
            elif args.operation == "clean":
                print("Cleaning repo")
                cdh.clean()
                flags[kernel] = True
        except Exception as e:
            print(f"Kernel {kernel} operation failed with error {repr(e)}")
    print(flags)
