# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Implementation pieces for command cros-ksm-rebase."""

import re

import pygit2
from pygit2.enums import ObjectType

from cros_ksm.git import Commit, Repo, Tag
from cros_ksm.util import die, eprint, log


def transform_todo_list(repo: Repo, text: str, /) -> str:
    """Transform a todo list created by `git rebase -r`."""
    lines = text.splitlines()
    lines = _discard_dead_lines(lines)
    lines = _tranform_merge_lines(repo, lines)
    return "\n".join(lines)

def _discard_dead_lines(old_lines: list[str], /) -> list[str]:
    """Discard all lines preceding the final 'reset' command."""

    new_lines = []

    for line in reversed(old_lines):
        if re.match(r"^(t|reset)\s", line):
            break
        new_lines.append(line)

    return list(reversed(new_lines))

def _tranform_merge_lines(repo: Repo, old_lines: list[str], /) -> list[str]:
    """Rewrite all lines that contain a 'merge' command."""

    new_lines = []

    for line in old_lines:
        # ruff: disable[redefined-loop-name]
        m = re.match(r"^(m|merge)\b", line)
        if m is not None:
            line = _transform_merge_line(repo, line)
        new_lines.append(line)

    return new_lines

def _transform_merge_line(repo: Repo, old_line: str, /) -> str:
    """Rewrite a single line that contains a 'merge' command."""

    log.debug("rebase: rewrite merge line: %s", old_line)

    def fail(message: str):
        eprint(f"error: failed to rewrite rebase line: {old_line}")
        eprint(message)
        die()

    m = re.match(r"^(?P<command>m|merge)\s+-C\s*(?P<oid>[a-fA-F0-9]+)\b", old_line)
    if m is None:
        fail("error: merge command has unrecognized syntax")

    # If the old line used an abbreviated rebase command, then continue to use
    # an abbreviated command in the new line.
    if len(m["command"]) == 1:
        exec_command = "x"
    else:
        exec_command = "exec"

    commit_id = m["oid"]
    commit = repo.resolve_refish_to_commit(commit_id)

    # ruff: ignore[magic-value-comparison]
    if len(commit.pygit2.parent_ids) > 2:
        fail(f"error: commit {commit.short_id} has too many parents")

    tag = _get_tag_for_commit_id(repo, commit)
    if tag is None:
        fail(f"error: failed to find a kernel release tag that points "
             f"to commit {commit_id}^2")

    new_line = [
        exec_command,
        "cros-ksm", "merge",
        "-C", commit_id,

        # Add these flags even though they are the default behavior. This allows
        # the user to quickly and easily changed the behavior of the merge
        # command when editing the todo list.
        "--commit",
        "--no-edit",

        "--rerere-autoupdate",
        tag.name,
        "#",
        commit.subject.text,
    ]

    new_line = " ".join(new_line)
    log.debug("rebase: rewrote merge line: %s", new_line)
    return new_line

def _get_tag_for_commit_id(repo: Repo, commit: Commit) -> Tag | None:
    """Find the kernel release tag that points to the commit id, if any.

    Return None on failure.
    """

    log.debug("rebase: find release tag for commit %s (%s)",
              commit.short_id, commit.subject.text)

    parent_ids = commit.pygit2.parent_ids

    try:
        parent_obj = repo.pygit2[parent_ids[1]]
    except KeyError:
        return None

    try:
        parent_commit = parent_obj.peel(ObjectType.COMMIT)
    except pygit2.InvalidSpecError:
        return None

    parent_commit = repo.register_pygit2_commit(parent_commit)

    log.debug("rebase: commit %s^2 is %s (%s)",
              commit.short_id, parent_commit.short_id, parent_commit.subject)

    m = re.match(r"^Linux\s+(\S+)\s*$", parent_commit.subject.text)
    if m is None:
        log.debug("rebase: unknown syntax in subject: %s",
                  parent_commit.subject)
        return None

    log.debug("rebase: parsed version from subject: %s", m[1])

    tag_name = "v" + m[1]
    tag = repo.try_resolve_tag_name(tag_name)
    if tag is None:
        log.debug("rebase: failed to resolve tag name %r", tag_name)
        return None

    log.debug("rebase: found tag %s", tag.name)

    if tag.commit != parent_commit:
        log.debug("rebase: tag %s does not point to commit %s",
                  tag.name, parent_commit.id)
        return None

    return tag
