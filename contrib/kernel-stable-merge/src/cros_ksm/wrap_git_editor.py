# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Implementation pieces for command cros-ksm-wrap-git-editor."""

import os
import subprocess
from pathlib import Path

import cros_ksm.rebase
from cros_ksm.git import Repo


def configure_caller_env(
    env: dict,
    repo: Repo,
    *,
    mode: str,
    no_edit: bool,
    override_text: str | None = None,
) -> None:
    """Configure the environment to make GIT_EDITOR or GIT_SEQUENCE_EDITOR
    dispatch to cros-ksm-wrap-git-editor.
    """

    match mode:
        case "cherry-pick":
            editor_name = "GIT_EDITOR"
        case "rebase":
            editor_name = "GIT_SEQUENCE_EDITOR"
        case _:
            assert False

    env[editor_name] = "cros-ksm wrap-git-editor"
    env["__CROS_KSM_GIT_EDITOR"] = repo.git_var(editor_name)
    env["__CROS_KSM_GIT_EDITOR_MODE"] = mode
    env["__CROS_KSM_GIT_EDITOR_NO_EDIT"] = str(int(no_edit)) # 0 or 1

    if mode == "cherry-pick":
        assert override_text is not None
        env["__CROS_KSM_GIT_EDITOR_TEXT"] = override_text

def edit(repo: Repo, path: Path) -> int:
    """Edit the path.

    Return the exit code of the original, wrapped git editor.
    """

    editor = os.environ["__CROS_KSM_GIT_EDITOR"]
    mode = os.environ["__CROS_KSM_GIT_EDITOR_MODE"]
    no_edit = bool(int(os.environ["__CROS_KSM_GIT_EDITOR_NO_EDIT"])) # 0 or 1

    match mode:
        case "cherry-pick":
            text = os.environ["__CROS_KSM_GIT_EDITOR_TEXT"]
        case "rebase":
            text = cros_ksm.rebase.transform_todo_list(repo, path.read_text())
        case _:
            assert False

    # Overwrite the file with our custom text
    path.write_text(text)

    if no_edit:
        return 0

    # The git vars GIT_EDITOR and GIT_SEQUENCE_EDITOR (see man:git-var) are
    # allowed to be shell commands.
    #
    # When invoking the editor, try to match the same behavior as the git cli.
    # Mimic the implementation in the git source code. Our implementation here
    # is a little simpler than git's implementation because we skip an
    # optimization, but the behavior should be the same. (Git invokes the editor
    # string directly without starting a shell if the string does not look like
    # a shell command, but we always use a shell).
    #
    # For details, see the following in the git-2.54.0 source:
    #   - editor.c:launch_specified_editor()
    #   - run-command.c:prepare_shell_cmd()
    #
    shell = repo.git_var("GIT_SHELL_PATH")

    # When a POSIX shell is invoked as ```sh -c <command_string> <arg0> [<arg1>
    # ...]```, the shell executes the command_string as a shell command with $0
    # set to arg0 and $@ set to the remaining args.
    cmd = [shell, "-c", f'{editor} "$@"', editor, path]

    return subprocess.call(cmd)
