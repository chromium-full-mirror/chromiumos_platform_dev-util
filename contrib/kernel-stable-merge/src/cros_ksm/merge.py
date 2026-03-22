# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Implementation pieces for command cros-kms-merge."""

import enum
import os.path
import subprocess
from dataclasses import dataclass
from pathlib import Path
from tempfile import NamedTemporaryFile
from textwrap import dedent
from typing import NoReturn

import cros_ksm.cli
from cros_ksm.git import (
    KernelVersion,
    Repo,
    make_gerrit_change_id,
)
from cros_ksm.util import (
    args_add_if,
    die,
    eprint,
    shquote_args,
)

PATCH_DIR_CONFIG_KEY = "cros-ksm.patch-dir"
PATCH_DIR_DEFAULT = "patches"

def get_patch_dir(repo: Repo) -> Path:
    """Directory that contains patches to apply during merge.

    Return absolute path.
    """

    try:
        config_value = repo.pygit2.config[PATCH_DIR_CONFIG_KEY]
    except KeyError:
        config_value = PATCH_DIR_DEFAULT

    try:
        path = Path(config_value)
    except ValueError:
        die(f"config key {PATCH_DIR_CONFIG_KEY!r} has invalid path: "
            f"{config_value!r}")

    if path.is_absolute():
        return path

    work_dir = repo.work_dir
    if work_dir is None:
        die("in a bare repo, cannot resolve cros-ksm.patch-dir to a path")

    return work_dir / path

class PatchStatus(enum.Enum):
    PENDING = enum.auto()
    CURRENT = enum.auto()
    DONE = enum.auto()

@dataclass
class Patch:
    path: Path
    status: PatchStatus = PatchStatus.PENDING

class Driver:
    """Driver for command cros-ksm-merge.

    The merged git rev must be the name of a kernel release tag. This
    restriction simplifies the implementation, especially the search for patches
    to be applied during the merge.
    """

    repo: Repo
    tag_name: str
    tag_kver: KernelVersion
    message: str
    opt_edit_message: bool
    opt_make_commit: bool
    opt_rerere_autoupdate: bool

    patches: list[Patch]

    def __init__(
        self,
        repo: Repo,
        tag_name: str,
        *,
        message: str | None = None,
        edit_message: bool,
        make_commit: bool,
        rerere_autoupdate: bool,
    ):
        self.repo = repo
        self.tag_name = tag_name
        self.opt_make_commit = make_commit
        self.opt_edit_message = edit_message
        self.opt_rerere_autoupdate = rerere_autoupdate
        self.patches = []

        # Verify that the revision resolves to a kernel version tag.
        _ = repo.resolve_tag_name(tag_name)

        kver = KernelVersion.parse_tag_name(tag_name)
        if kver is None:
            die(f"tag {tag_name!r} does not look like a kernel release tag")
        self.tag_kver = kver

        if message is None:
            message = self._format_default_message()
        self.message = message

    def _find_patches(self) -> None:
        patch_dir = get_patch_dir(self.repo)

        # Simplify the message output by using relative paths when reasonable.
        if patch_dir.is_relative_to(Path.cwd()):
            patch_dir = patch_dir.relative_to(Path.cwd())

        eprint(f"search for patches for {self.tag_name}")

        for path in sorted(patch_dir.glob(f"{self.tag_kver}-*.patch")):
            if path.is_file():
                eprint(f"found patch: {path}")
                self.patches.append(Patch(path))

        if len(self.patches) == 0:
            eprint("no patches found")

    def _format_default_message(self) -> str:
        change_id = make_gerrit_change_id()
        bug = self.repo.get_cros_ksm_bug()
        committer = self.repo.get_committer()
        dst_branch = os.path.basename(self.repo.cros_branch)

        return dedent(f"""\
            CHROMIUM: Merge tag '{self.tag_name}' into {dst_branch}

            BUG=b:{bug}
            TEST=cq

            Change-Id: {change_id}
            Disallow-Recycled-Builds: test-failures
            Signed-off-by: {committer}
            """)

    def _run_command(self, args, *, fail_ok: bool = False) -> int:
        # For consistency on stderr, use the same verb that git-rebase uses in
        # its status output.
        eprint("Executing: " + shquote_args(args))

        status = subprocess.call(args)
        if status != 0 and not fail_ok:
            self._fail_command(args)

        return status

    def _fail_command(self, args) -> NoReturn:
        """Explain how to continue the merge, then die.

        Only use this function if the merge has begun and it is in progress.
        """

        # Before our output, git-rebase will have already printed a LOT of
        # status messages. Add a separator between that and our help text.
        eprint("")

        # Do not embed the unsanitized command into the dedent string below.
        qargs = shquote_args(args)
        eprint(f"command failed: {qargs}")

        eprint(dedent("""\

            To continue the merge:
              - Resolve any merge conflicts.
              - Apply any patches marked 'todo' in the patch list below.
              - Use `git add` to add all changes to the index.
              - Run `git merge --continue`.

            Patches:"""))

        if self.patches:
            for patch in self.patches:
                match patch.status:
                    case PatchStatus.DONE:
                        status = "done"
                    case PatchStatus.CURRENT:
                        status = "failed"
                    case PatchStatus.PENDING:
                        status = "todo"
                    case _:
                        assert False

                status = f"[{status}]"
                eprint(f"    {status:<12} {patch.path}")
        else:
            eprint("    none")

        # After our output, git-rebase will also print help. Add a separator
        # between our help and the git-rebase help.
        eprint("")

        die()

    def run(self) -> None:
        # Git creates the MERGE_HEAD file when it starts a merge. If it already
        # exists, then `git merge` will refuse to start a new merge.
        #
        # We refuse to call `git merge` if the file already exists because, if
        # `git merge` fails, we will partially infer its failure mode using the
        # file's post-failure existence. If the file does not exist
        # post-failure, then git never started the merge, and therefore we
        # should not try to continue. If the file exists post-failure, then git
        # failed _after_ the merge started, and in special cases we should
        # ignore the failure and continue the merge.
        merge_head_file = self.repo.git_path("MERGE_HEAD")
        if merge_head_file.exists():
            eprint(f"error: file already exists: {merge_head_file}")
            eprint("maybe a merge is already in progress?")
            die()

        eprint(f"prepare to merge tag {self.tag_name}")

        self._find_patches()

        # Does the merge need to be completed with `git merge --continue`?
        need_continue = False

        with (NamedTemporaryFile(mode="wt+", prefix="cros-ksm-merge-")
              as message_file):
            message_file.write(self.message)
            message_file.flush()

            cmd = ["git", "merge", "--no-ff"]

            if not self.opt_make_commit:
                cmd += ["--no-commit"]
                need_continue = False
            elif self.patches:
                # The user requested --commit, but we will create the commit
                # after applying the patches.
                cmd += ["--no-commit"]
                need_continue = True
            else:
                # For consistency with the other codepaths, we could always use
                # --no-commit and always defer creation of the commit object
                # until after we apply the (possibly empty) list of patches.
                # However, most merges will lack patches, and we can speed up
                # this common case and simplify the user-visible output (and
                # possible error message) by using a single git command to
                # perform the merge and create the commit object.
                cmd += ["--commit"]
                need_continue = False

            args_add_if(cmd, self.opt_edit_message,
                        "--edit", "--no-edit")
            args_add_if(cmd, self.opt_rerere_autoupdate,
                        "--rerere-autoupdate", "--no-rerere-autoupdate")
            cmd += [f"-F{message_file.name}", "--", self.tag_name]

            status = self._run_command(cmd, fail_ok=True)

        if status != 0:
            if not merge_head_file.exists():
                # Git failed to start the merge and we do not know why. Do not
                # print our help text and failure messages, which are relevant
                # only when the merge fails mid-progress.
                die()
            elif self.opt_rerere_autoupdate and not self.repo.has_status_conflicts():
                # Do not embed the unsanitized command into the dedent string
                # below.
                qcmd = shquote_args(cmd)
                eprint(f"command failed: {qcmd}")

                eprint(cros_ksm.cli.wrap_text(
                    dedent("""\
                        cros-ksm: The merge failed but the tree status has no
                        unmerged files. This often happens when
                        --rerere-autoupdate successfully resolves all merge
                        conflicts. Ignore the failure and continue the merge.
                        """),
                    initial_indent = "",
                    subsequent_indent = "  ",
                ))

                if self.opt_make_commit:
                    need_continue = True
            else:
                self._fail_command(cmd)

        for patch in self.patches:
            assert patch.status == PatchStatus.PENDING
            patch.status = PatchStatus.CURRENT
            self._run_command(["git", "apply", "--index", "--", patch.path])
            patch.status = PatchStatus.DONE

        if need_continue:
            # Make the commit object now after all the patches have been
            # applied.
            cmd = ["git", "merge", "--continue"]

            # If the user requested --no-edit, then temporarily disable the
            # GIT_EDITOR when making the commit. This is required because `git
            # merge --continue` always invokes GIT_EDITOR, even when the merge
            # was begun with `git merge --no-edit`.
            #
            # This detail is required to make our custom rebase command behave
            # as expected. Without it, the user would be forced to edit the
            # commit message of any rebased merge commit that reaches this
            # codepath.
            if not self.opt_edit_message:
                cmd = ["env", "GIT_EDITOR=:"] + cmd

            self._run_command(cmd)
