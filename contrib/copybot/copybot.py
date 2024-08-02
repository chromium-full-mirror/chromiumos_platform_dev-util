#!/usr/bin/env vpython3
# Copyright 2022 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""CopyBot script.

This script copies commits from one repo (the "upstream") to another
(the "downstream").

Usage: copybot.py [options...] upstream_repo:branch downstream_repo:branch
"""

# [VPYTHON:BEGIN]
# python_version: "3.8"
# wheel: <
#   name: "infra/python/wheels/requests-py3"
#   version: "version:2.31.0"
# >
# wheel: <
#   name: "infra/python/wheels/certifi-py2_py3"
#   version: "version:2020.11.8"
# >
# wheel: <
#   name: "infra/python/wheels/idna-py2_py3"
#   version: "version:2.8"
# >
# wheel: <
#   name: "infra/python/wheels/charset_normalizer-py3"
#   version: "version:2.0.4"
# >
# wheel: <
#   name: "infra/python/wheels/urllib3-py2_py3"
#   version: "version:1.26.6"
# >
# [VPYTHON:END]

from __future__ import annotations

import argparse
import dataclasses
import enum
import json
import logging
import os
import pathlib
import re
import shlex
import subprocess
import tempfile
import time
from typing import Dict, Optional, Union
import urllib

import requests  # pylint: disable=import-error


logger = logging.getLogger(__name__)

# Matches a full 40-character commit hash.
_COMMIT_HASH_PATTERN = re.compile(r"\b[0-9a-f]{40}\b")


class MergeConflictBehavior(enum.Enum):
    """How to behave on merge conflicts.

    FAIL: Stop immediately. Don't upload anything.
    SKIP: Skip the commit that failed to merge. Summarize the failed
        commits at the end of the execution, and exit failure status.
    STOP: Stop immediately. Upload staged changes prior to conflict.
    """

    FAIL = enum.auto()
    SKIP = enum.auto()
    STOP = enum.auto()


class MergeConflictError(Exception):
    """A commit cannot be cherry-picked due to a conflict."""


class EmptyCommitError(Exception):
    """A commit cannot be cherry-picked as it results in an empty commit."""


class CopybotFatalError(Exception):
    """Copybot fatal error."""

    enum_name = "FAILURE_UNKNOWN"

    def __init__(self, *args, commits=(), **kwargs):
        self.commits = commits
        super().__init__(*args, **kwargs)


class UpstreamFetchError(CopybotFatalError):
    """Copybot died as the upstream failed to fetch."""

    enum_name = "FAILURE_UPSTREAM_FETCH_ERROR"


class DownstreamFetchError(CopybotFatalError):
    """Copybot died as the downstream failed to fetch."""

    enum_name = "FAILURE_DOWNSTREAM_FETCH_ERROR"


class PushError(CopybotFatalError):
    """Copybot died as it failed to push to the downstream GoB host."""

    enum_name = "FAILURE_DOWNSTREAM_PUSH_ERROR"


class MergeConflictsError(CopybotFatalError):
    """Copybot ran, but encountered merge conflicts."""

    enum_name = "FAILURE_MERGE_CONFLICTS"


@dataclasses.dataclass
class GerritClInfo:
    """Stores information for a Gerrit CL."""

    def __init__(self, change_id: str, hashtags: str, ref: str) -> None:
        """Initialize the Gerrit CL Info.

        Args:
            change_id: Change ID on Gerrit for this change.
            hashtags: Hashtags associated with this change
            ref: Current ref for this changes to be able to form a refspec
        """
        self.change_id = change_id
        self.hashtags = hashtags
        self.current_ref = ref


class GitRepo:
    """Class wrapping common Git repository actions."""

    def __init__(self, git_dir):
        self.git_dir = git_dir

    def _run_git(self, *args, **kwargs):
        """Wrapper to run git with the provided arguments."""
        argv = ["git", "-C", self.git_dir, "--no-pager", *args]
        logger.info("Run `%s`", " ".join(shlex.quote(str(arg)) for arg in argv))
        kwargs.setdefault("encoding", "utf-8")
        kwargs.setdefault("errors", "replace")
        try:
            return subprocess.run(
                argv,
                check=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                **kwargs,
            )
        except subprocess.CalledProcessError as e:
            logger.error("Git command failed!")
            logger.error("  STDOUT:")
            for line in e.stdout.splitlines():
                logger.error("    %s", line)
            logger.error("  STDERR:")
            for line in e.stderr.splitlines():
                logger.error("    %s", line)
            raise

    @classmethod
    def init(cls, git_dir: Union[str, "os.PathLike[str]"]):
        """Do a `git init` to create a new repository."""
        git_dir = pathlib.Path(git_dir)
        repo = cls(git_dir)
        if not (git_dir / ".git").exists():
            repo._run_git("init")  # pylint: disable=protected-access
        return repo

    def rev_parse(self, rev="HEAD"):
        """Do a `git rev-parse`."""
        result = self._run_git("rev-parse", rev)
        return result.stdout.rstrip()

    def fetch(self, remote, ref=None):
        """Do a `git fetch`.

        Returns:
            The full commit hash corresponding to FETCH_HEAD.
        """
        extra_args = []
        if ref:
            extra_args.append(ref)
        self._run_git("fetch", remote, *extra_args)
        return self.rev_parse("FETCH_HEAD")

    def checkout(self, ref):
        """Do a `git checkout`."""
        return self._run_git("checkout", ref)

    def log(
        self,
        revision_range="HEAD",
        fmt=None,
        num=0,
        subtree=None,
        exclude_file_patterns=(),
    ):
        """Do a `git log`."""
        extra_args = []
        if fmt:
            extra_args.append(f"--format={fmt}")
        if num:
            extra_args.append(f"-n{num}")
        extra_args.append("--")
        extra_args.extend(
            [f":!{path}" for path in (exclude_file_patterns or [])]
        )
        if subtree:
            extra_args.append(subtree)
        return self._run_git("log", revision_range, *extra_args)

    def log_hashes(
        self,
        revision_range="HEAD",
        num=0,
        subtree=None,
        exclude_file_patterns=(),
    ):
        """Get the commit log as a list of commit hashes."""
        result = self.log(
            revision_range=revision_range,
            fmt="%H",
            num=num,
            subtree=subtree,
            exclude_file_patterns=exclude_file_patterns,
        )
        return result.stdout.splitlines()

    def get_commit_message(self, rev="HEAD"):
        """Get a commit message of a commit."""
        result = self.log(revision_range=rev, num=1, fmt="%B")
        return result.stdout

    def commit_file_list(self, rev="HEAD"):
        """Get the files modified by a commit."""
        result = self._run_git("show", "--pretty=", "--name-only", rev)
        return result.stdout.splitlines()

    def show(self, rev="HEAD", files=()):
        """Do a `git show`."""
        result = self._run_git("show", rev, "--", *files)
        return result.stdout

    def apply(self, patch, path=None, include_paths=None, exclude_paths=None):
        """Apply a patch to the staging area."""
        extra_args = []
        if path:
            extra_args.append(f"--directory={path}")
        if include_paths:
            extra_args.extend(f"--include={path}" for path in include_paths)
        if exclude_paths:
            extra_args.extend(f"--exclude={path}" for path in exclude_paths)

        extra_args.append(patch)
        return self._run_git("apply", *extra_args)

    def commit(self, message, amend=False, sign_off=False, stage=False):
        """Create a commit.

        Returns:
            The commit hash.
        """
        extra_args = []
        if stage:
            extra_args.append("--all")
        if amend:
            extra_args.append("--amend")
        if sign_off:
            extra_args.append("--signoff")
        self._run_git("commit", *extra_args, "-m", message)
        return self.rev_parse()

    def reword(self, new_message, sign_off=False):
        """Reword the commit at HEAD.

        Returns:
            The new commit hash.
        """
        return self.commit(new_message, amend=True, sign_off=sign_off)

    def format_patch(self, rev, output_path, num=1, relative_path=None):
        """Generate patch from revision with an optional relative path.

        Returns:
            Path of the output patch.
        """
        extra_args = []
        if relative_path:
            extra_args.append(f"--relative={relative_path}")
        extra_args.extend([f"-{num}", rev, f"--output-directory={output_path}"])

        result = self._run_git("format-patch", *extra_args)
        return result.stdout.rstrip()

    def add(self, path, stage=False, force=False):
        """Add unstaged files."""
        extra_args = []
        if stage:
            extra_args.append("--all")
        if force:
            extra_args.append("--force")
        if path:
            extra_args.append(path)
        return self._run_git("add", *extra_args)

    def get_subtree_lowest_working_dir(
        self, path: Union[pathlib.Path, "os.PathLike[str]"]
    ) -> pathlib.Path:
        """Get the lowest working directory path for subtree."""
        patch_dir = pathlib.Path(self.git_dir) / path
        subtree = path
        if not patch_dir.is_dir():
            subtree = pathlib.Path(path).parents[0]
        return subtree

    def cherry_pick(
        self,
        rev,
        patch_dir=None,
        upstream_subtree=None,
        downstream_subtree=None,
        include_paths=None,
        exclude_paths=None,
    ):
        """Do a `git cherry-pick`.

        This will first try without any merge options, and if that fails,
        try again with -Xpatience, which is slower, but may be more likely
        to resolve a merge conflict.

        Raises:
            EmptyCommitError: The resultant commit was empty and should be
                skipped.
            MergeConflictError: There was a merge conflict that could not
                be resolved automatically with -Xpatience.
        """

        def _try_cherry_pick(extra_flags):
            try:
                self._run_git("cherry-pick", "-x", rev, *extra_flags)
            except subprocess.CalledProcessError as e:
                try:
                    self._run_git("rev-parse", "--verify", "CHERRY_PICK_HEAD")
                    logger.warning("Could not cherry-pick")
                except subprocess.CalledProcessError as err:
                    if "is a merge but no -m option was given" in e.stderr:
                        logger.warning("Merge commit detected")
                    raise MergeConflictError() from err
                self._run_git("cherry-pick", "--abort")
                if "The previous cherry-pick is now empty" in e.stderr:
                    raise EmptyCommitError() from e
                raise MergeConflictError() from e

        cherry_pick_flag_list = ([], ["-Xpatience"], ["-m", "2"])
        if downstream_subtree or upstream_subtree or include_paths:
            cherry_pick_flag_list = ()
        for flags in cherry_pick_flag_list:
            try:
                _try_cherry_pick(flags)
            except MergeConflictError:
                continue
            else:
                return
        patch = self.format_patch(
            rev,
            patch_dir,
            1,
            self.get_subtree_lowest_working_dir(upstream_subtree),
        )
        try:
            self.apply(
                patch=patch,
                path=self.get_subtree_lowest_working_dir(downstream_subtree),
                include_paths=include_paths,
                exclude_paths=exclude_paths,
            )
        except subprocess.CalledProcessError as e:
            raise MergeConflictError() from e
        self.add(downstream_subtree, stage=True, force=True)
        self.commit(
            self.get_commit_message(rev),
            amend=False,
            sign_off=False,
            stage=True,
        )
        return

    def push(self, url, refspec, options=()):
        """Do a `git push`."""
        args = []

        for option in options:
            args.extend(["-o", option])

        args.append(url)
        args.append(refspec)
        self._run_git("push", *args)


class Pseudoheaders:
    """Dictionary-like object for the pseudoheaders from a commit message.

    The pseudoheaders are the header-like lines often found in the
    bottom of a commit message.  Header names are case-insensitive.

    Pseudoheaders are parsed the same way that the "git footers"
    command parses them.
    """

    # Matches lines that look like a "header" (the conventional footer
    # lines in a commit message).
    _PSEUDOHEADER_PATTERN = re.compile(r"^(?:[A-Za-z0-9]+-)*[A-Za-z0-9]+:\s+")

    def __init__(self, header_list=()):
        self._header_list = list(header_list)

    @classmethod
    def from_commit_message(cls, commit_message, offset=1):
        """Parse pseudoheaders from a commit message.

        Args:
            commit_message: commit message from git log.
            offset: which line to start processing the message from.
                Lines less than the offset will not be altered.

        Returns:
            Two values, a Pseudoheaders dictionary, and the commit
            message without any pseudoheaders.
        """
        message_lines = commit_message.splitlines()
        rewritten_message = []

        header_list = []
        for i, line in enumerate(message_lines):
            if i < offset or not cls._PSEUDOHEADER_PATTERN.match(line):
                rewritten_message.append(line)
            else:
                name, _, value = line.partition(":")
                header_list.append((name, value.strip()))

        return cls(header_list), "".join(
            f"{line}\n" for line in rewritten_message
        )

    def prefix(self, prefix="Original-", keep=()):
        """Prefix all header keys with a string.

        Args:
            prefix: The prefix to use.
            keep: Headers which should not be modified.

        Returns:
            A new Pseudoheaders dictionary.
        """
        new_header_list = []

        # Constructing a new pseudoheaders dictionary ensures we
        # consider the keep list to be case insensitive.
        keep_dict = self.__class__([(key, True) for key in keep])

        for key, value in self._header_list:
            if keep_dict.get(key):
                new_header_list.append((key, value))
            else:
                new_header_list.append((f"{prefix}{key}", value))
        return self.__class__(new_header_list)

    def __getitem__(self, item):
        """Get a header value by name."""
        for key, value in self._header_list:
            if key.lower() == item.lower():
                return value
        raise KeyError(item)

    def get(self, item, default=None):
        """Get a header value by name, or return a default value."""
        try:
            return self[item]
        except KeyError:
            return default

    def as_dict(self) -> Dict[str, str]:
        """Get the dict of stored values."""
        return dict(self._header_list)

    def __setitem__(self, key, value):
        """Add a header."""
        self._header_list.append((key, value))

    def add_to_commit_message(self, commit_message):
        """Add our pseudoheaders to a commit message.

        Returns:
            The new commit message.
        """
        message_lines = commit_message.splitlines()

        if not message_lines:
            message_lines = ["NO COMMIT MESSAGE"]

        # Ensure exactly one blank line separating body and pseudoheaders.
        while not message_lines[-1].strip():
            message_lines.pop()

        message_lines.append("")

        for key, value in self._header_list:
            message_lines.append(f"{key}: {value}")
        return "".join(f"{line}\n" for line in message_lines)

    def __str__(self):
        return "\n".join(f"{key}:{value}" for key, value in self._header_list)

    def update(self, other: Union[dict, Pseudoheaders]) -> None:
        if isinstance(other, type(self)):
            for key, value in other.as_dict().items():
                self[key] = value
        elif isinstance(other, dict):
            for key, value in other:
                self[key] = value
        else:
            raise TypeError(f"Other class has conflicting type({type(other)})")


class Gerrit:
    """Wrapper for actions on a Gerrit host."""

    def __init__(self, hostname):
        self.hostname = hostname

    def search(self, query):
        """Do a query on Gerrit."""
        url = f"https://{self.hostname}/changes/"
        params = [
            ("q", query),
            ("o", "CURRENT_REVISION"),
            ("o", "CURRENT_COMMIT"),
            ("o", "COMMIT_FOOTERS"),
        ]
        while True:
            r = requests.get(url, params=params)
            if r.ok:
                break
            if r.status_code == requests.codes.too_many:
                time.sleep(1)
                continue
            r.raise_for_status()
            assert False

        if r.text[:5] != ")]}'\n":
            logger.error("Bad response from Gerrit: %r", r.text)
            raise ValueError("Unexpected JSON payload from gerrit")

        result = json.loads(r.text[5:])
        return result

    def find_pending_changes(
        self, project, branch, hashtags=(), subtree=None, exclude_paths=()
    ):
        """Find pending changes previously opened by CopyBot on Gerrit.

        Returns:
            A dictionary mapping upstream commit hashes to their
            current Change-Id on Gerrit and their associated hashtags.
        """
        query = [
            "status:open",
            f"project:{project}",
            f"branch:{branch}",
        ]
        query.extend(f"hashtag:{hashtag}" for hashtag in hashtags)
        if subtree:
            query.append(f"directory:{subtree}")
        for path in exclude_paths:
            query.append(f"-directory:{path}")
        query_result = self.search(" ".join(query))
        change_ids = {}
        for cl in query_result:
            change_id = cl["change_id"]
            current_revision_hash = cl["current_revision"]
            current_revision_data = cl["revisions"][current_revision_hash]
            commit_message = current_revision_data["commit"]["message"]
            origin_rev_id = get_origin_rev_id(commit_message)
            rev_id = []
            if origin_rev_id:
                rev_id.append(origin_rev_id)
            else:
                for commit_hash in _COMMIT_HASH_PATTERN.finditer(
                    commit_message
                ):
                    rev_id.append(commit_hash.group(0))
            for rev in rev_id:
                change_ids[rev] = GerritClInfo(
                    change_id,
                    cl["hashtags"].copy(),
                    current_revision_data["ref"],
                )
        return change_ids


def generate_change_id():
    """Generate a Unique Change-Id."""
    return f"I{os.urandom(20).hex()}"


def get_origin_rev_id(commit_message):
    """Get the origin revision hash from a commit message.

    Returns:
        The revision hash if one was found, or None otherwise.
    """
    pseudoheaders, _ = Pseudoheaders.from_commit_message(commit_message)
    origin_revid = pseudoheaders.get("GitOrigin-RevId")
    if not origin_revid:
        origin_revid = pseudoheaders.get("Original-Commit-Id")
    return origin_revid


def find_last_merged_rev(
    repo,
    upstream_rev,
    downstream_rev,
    upstream_subtree=None,
    downstream_subtree=None,
    exclude_file_patterns=(),
):
    """Find the last merged revision in a Git repo.

    Args:
        repo: The GitRepo.
        upstream_rev: The commit hash of the upstream HEAD.
        downstream_rev: The commit hash of the downstream HEAD.
        upstream_subtree: The subtree of interest of the upstream repo.
        downstream_subtree: The subtree of interest of the downstream repo.
        exclude_file_patterns: List of paths to be excluded.

    Returns:
        Two values,
            1. A commit hash of the last merged revision by CopyBot, or the
                first common commit hash in both logs.
            2. The number of CLs which are eligible to be downstreamed.

    Raises:
        ValueError: No common history could be found.
    """
    upstream_hashes = repo.log_hashes(
        revision_range=upstream_rev,
        subtree=upstream_subtree,
        exclude_file_patterns=exclude_file_patterns,
    )
    downstream_hashes = repo.log_hashes(
        revision_range=downstream_rev,
        subtree=downstream_subtree,
        exclude_file_patterns=exclude_file_patterns,
    )

    for rev in downstream_hashes:
        commit_message = repo.get_commit_message(rev)
        origin_revid = get_origin_rev_id(commit_message)
        if rev in upstream_hashes or origin_revid:
            if origin_revid in upstream_hashes:
                counter = upstream_hashes.index(origin_revid or rev)
            else:
                continue
            return origin_revid or rev, counter

    raise ValueError(
        "Downstream has no GitOrigin-RevId commits, and upstream and "
        "downstream share no common history."
    )


def get_downstreamed_list(
    repo,
    downstream_rev,
    downstream_subtree=None,
    exclude_file_patterns=(),
    limit=0,
):
    """Find the last merged revision in a Git repo.

    Args:
        repo: The GitRepo.
        downstream_rev: The commit hash of the downstream HEAD.
        downstream_subtree: The subtree of interest of the downstream repo.
        exclude_file_patterns: List of paths to be excluded.
        limit: The maxinum number of CLs in the history to check.

    Returns:
        The set of upstream commit hashes that have already been downstreamed.
    """
    downstream_hashes = repo.log_hashes(
        revision_range=downstream_rev,
        subtree=downstream_subtree,
        exclude_file_patterns=exclude_file_patterns,
    )
    downstreamed_revs = downstream_hashes[:]

    for counter, rev in enumerate(downstream_hashes):
        if counter > limit and limit != 0:
            break

        commit_message = repo.get_commit_message(rev)
        origin_revid = get_origin_rev_id(commit_message)

        if origin_revid:
            downstreamed_revs.append(origin_revid)
    return downstreamed_revs


def find_commits_to_copy(
    repo,
    upstream_rev,
    upstream_subtree,
    downstream_rev,
    downstream_subtree,
    include_paths,
    upstream_limit=0,
    downstream_limit=0,
    exclude_file_patterns=(),
    pending_changes=(),
    skip_copybot_job_names=(),
):
    """Find the commits to copy to downstream.

    Args:
        repo: The GitRepo.
        upstream_rev: The commit hash of the upstream HEAD.
        upstream_subtree: The subtree of interest of the upstream repo.
        downstream_rev: The commit hash of the downstream HEAD.
        downstream_subtree: The subtree of interest of the downstream repo.
        include_paths: The paths to include from the upstream relative to
            the downstream subtree(Only valid with downstream subtree)
        upstream_limit: The maxinum number of CLs in the upstream history to
            check.
        downstream_limit: The maxinum number of CLs in the downstream history
            to check.
        exclude_file_patterns: File paths that should not be copied.
        pending_changes: Changes pending in downstream repo.
        skip_copybot_job_names: Names of copybot jobs to not copy CLs from

    Returns:
        A list of the commit hashes to copy.

    Raises:
        ValueError: If the provided last merged commit hash does not
           exist in upstream commit history.
    """
    commits_to_copy = []
    downstreamed_revs = get_downstreamed_list(
        repo,
        downstream_rev,
        downstream_subtree,
        exclude_file_patterns=exclude_file_patterns,
        limit=downstream_limit,
    )

    for counter, rev in enumerate(
        repo.log_hashes(
            revision_range=upstream_rev,
            subtree=upstream_subtree,
            exclude_file_patterns=exclude_file_patterns,
        )
    ):
        # Early exit if limit reached to avoid inadvertent continuation
        if counter > upstream_limit and upstream_limit != 0:
            break

        # Check if this is a filtered commit.
        commit_message = repo.get_commit_message(rev)
        pseudoheaders, commit_message = Pseudoheaders.from_commit_message(
            commit_message
        )
        job_name = pseudoheaders.get("Copybot-Job-Name")
        if job_name in skip_copybot_job_names:
            logger.info(
                "Skip %s due to Copybot-Job-Name: %s",
                rev,
                job_name,
            )
            continue

        if rev in pending_changes:
            if "copybot-skip" in pending_changes[rev].hashtags:
                logger.info("Skip %s due to copybot-skip hashtag", rev)
                continue

        # If change is in pending list, allow relands
        if rev in downstreamed_revs:
            if rev in pending_changes:
                logger.info(
                    "Found pending change that has already merged: %s", rev
                )
            else:
                continue

        if downstream_subtree and include_paths:
            commit_files = repo.commit_file_list(rev)
            filtered_commit_files = []
            for path in commit_files:
                filtered_path = path.relative_to(upstream_subtree)
                if filtered_path in include_paths:
                    filtered_commit_files.append(path)
                    break

            if not filtered_commit_files:
                logger.info(
                    "Skip commit %s due to empty file list after filtering "
                    "(before filtering was %r)",
                    rev,
                    commit_files,
                )
                continue
        commits_to_copy.append(rev)

    return commits_to_copy


def rewrite_commit_message(
    repo,
    upstream_rev,
    change_id,
    prepend_subject="",
    sign_off=False,
    keep_pseudoheaders=(),
    additional_pseudoheaders=(),
):
    """Reword the commit at HEAD with appropriate metadata.

    Args:
        repo: The GitRepo to operate on.
        upstream_rev: The upstream commit hash corresponding to this commit.
        change_id: The Change-Id to add to the commit.
        prepend_subject: A string to prepend the subject line with.
        sign_off: True if Signed-off-by should be added to the commit message.
        keep_pseudoheaders: Pseudoheaders which should not be prefixed.
        additional_pseudoheaders: Psuedoheaders to be added to the commit
            message.
    """
    commit_message = repo.get_commit_message()
    if prepend_subject:
        commit_message = prepend_subject + commit_message
    pseudoheaders, commit_message = Pseudoheaders.from_commit_message(
        commit_message
    )
    pseudoheaders = pseudoheaders.prefix(keep=keep_pseudoheaders)

    pseudoheaders["GitOrigin-RevId"] = upstream_rev
    for additional_header in additional_pseudoheaders:
        parsed, _ = Pseudoheaders.from_commit_message(
            additional_header, offset=0
        )
        pseudoheaders.update(parsed)
    if "Change-Id" not in keep_pseudoheaders:
        pseudoheaders["Change-Id"] = change_id

    commit_message = pseudoheaders.add_to_commit_message(commit_message)
    repo.reword(commit_message, sign_off=sign_off)


def get_push_refspec(args, downstream_branch):
    """Generate a push refspec for Gerrit.

    Args:
        args: The parsed command line arguments.
        downstream_branch: The branch to push to.

    Returns:
        A push refspec as a string.
    """
    push_options = []

    def _add_push_option(key, value):
        for option in value.split(","):
            push_options.append(f"{key}={option}")

    for label in args.labels:
        _add_push_option("l", label)

    for cc in args.ccs:
        _add_push_option("cc", cc)

    for reviewer in args.reviewers:
        _add_push_option("r", reviewer)

    for hashtag in [args.topic, *args.hashtags]:
        _add_push_option("t", hashtag)

    return f"HEAD:refs/for/{downstream_branch}%{','.join(push_options)}"


def is_server_gob(url):
    return re.fullmatch(
        r"https://(chromium|chrome-internal)"
        r"(?:-review)?\.googlesource\.com/(.*)",
        url,
    )


def parse_repo_info(repo_string):
    is_local = True
    max_fields = 3
    repo_info = []

    base_string = repo_string
    url_result = urllib.parse.urlparse(repo_string)
    if url_result.netloc and url_result.scheme:
        is_local = False
    for _ in range(max_fields + 1):
        base_string, sep, current_field = base_string.rpartition(":")
        if not sep:
            if is_local:
                repo_info.insert(0, pathlib.Path(current_field))
            else:
                repo_info[0] = url_result.scheme + ":" + repo_info[0]
            break
        else:
            repo_info.insert(0, current_field)

    for _ in range(len(repo_info), max_fields):
        repo_info.append(None)

    repo_url = repo_info[0]
    repo_branch = repo_info[1]
    repo_subtree = repo_info[2]
    if not repo_branch:
        repo_branch = "main"

    return is_local, repo_url, repo_branch, repo_subtree


def run_copybot(args, git_dir, patch_dir):
    """Run copybot.

    Args:
        args: The parsed command line arguments.
        git_dir: A temporary or local directory to use for Git operations.
        patch_dir: A temporary directory to use for storing patch files.
    """
    (
        _,
        upstream_url,
        upstream_branch,
        upstream_subtree,
    ) = parse_repo_info(args.upstream)
    (
        local_downstream,
        downstream_url,
        downstream_branch,
        downstream_subtree,
    ) = parse_repo_info(args.downstream)

    if local_downstream:
        git_dir = downstream_url

    keep_pseudoheaders = list(args.keep_pseudoheaders)
    if upstream_url == downstream_url or (
        is_server_gob(str(downstream_url)) and is_server_gob(str(upstream_url))
    ):
        if "Change-Id" not in keep_pseudoheaders:
            keep_pseudoheaders.append("Change-Id")
    merge_conflict_behavior = MergeConflictBehavior[
        args.merge_conflict_behavior
    ]
    pending_changes = {}
    gerrit: Optional[Gerrit] = None
    if (m := is_server_gob(str(downstream_url))) is not None:
        downstream_gob_host = m.group(1)
        downstream_project = m.group(2)

        gerrit = Gerrit(f"{downstream_gob_host}-review.googlesource.com")
        pending_changes = gerrit.find_pending_changes(
            project=downstream_project,
            branch=downstream_branch,
            hashtags=[args.topic],
            subtree=downstream_subtree,
            exclude_paths=args.exclude_file_patterns,
        )
        logger.info(
            "Found %s pending changes already on Gerrit", len(pending_changes)
        )

    repo = GitRepo.init(git_dir)
    try:
        upstream_rev = repo.fetch(upstream_url, upstream_branch)
    except subprocess.CalledProcessError as e:
        raise UpstreamFetchError(
            f"Failed to fetch branch {upstream_branch} from {upstream_url}"
        ) from e

    try:
        downstream_rev = repo.fetch(downstream_url, downstream_branch)
    except subprocess.CalledProcessError as e:
        raise DownstreamFetchError(
            f"Failed to fetch branch {downstream_branch} from {downstream_url}"
        ) from e

    # Verify that the two repositories share a history
    try:
        last_related_rev, num_cls_to_downstream = find_last_merged_rev(
            repo,
            upstream_rev,
            downstream_rev,
            upstream_subtree,
            downstream_subtree,
            args.exclude_file_patterns,
        )
    except ValueError as e:
        if not pending_changes:
            raise e
        else:
            logger.info("Last related revision from pending changes!")
    else:
        logger.info("Last related revision: %s", last_related_rev)

    logger.info("Found: %s changes to downstream", num_cls_to_downstream)

    if num_cls_to_downstream > args.upstream_history_limit:
        logger.warning(
            "There are %s CLs between HEAD and %s but the history limit is"
            " set to %s. Raising the history limit to accommodate this.",
            num_cls_to_downstream,
            last_related_rev,
            args.upstream_history_limit,
        )
        args.upstream_history_limit = num_cls_to_downstream

    commits_to_copy = find_commits_to_copy(
        repo,
        upstream_rev,
        upstream_subtree,
        downstream_rev,
        downstream_subtree,
        include_paths=args.include_downstream,
        upstream_limit=args.upstream_history_limit,
        downstream_limit=args.downstream_history_limit,
        exclude_file_patterns=args.exclude_file_patterns,
        pending_changes=pending_changes,
        skip_copybot_job_names=args.skip_job_name,
    )

    if not commits_to_copy:
        logger.info("Nothing to do!")
        return

    skipped_revs = []
    empty_revs = []

    if args.limit > 0 and len(commits_to_copy) > args.limit:
        logger.warning(
            "Limiting commits to copy from %s to %s",
            len(commits_to_copy),
            args.limit,
        )
        commits_to_copy = commits_to_copy[-args.limit :]

    repo.checkout(downstream_rev)
    for i, rev in enumerate(reversed(commits_to_copy)):
        logger.info("(%s/%s) Cherry-pick %s", i + 1, len(commits_to_copy), rev)
        pending_change = False
        if rev in pending_changes:
            if "copybot-preserve" in pending_changes[rev].hashtags:
                pending_change = True
        try:
            if pending_change:
                repo.fetch(downstream_url, pending_changes[rev].current_ref)
                repo.cherry_pick("FETCH_HEAD")
            else:
                repo.cherry_pick(
                    rev,
                    patch_dir=patch_dir,
                    upstream_subtree=upstream_subtree,
                    downstream_subtree=downstream_subtree,
                    include_paths=args.include_downstream,
                    exclude_paths=args.exclude_file_patterns,
                )
        except EmptyCommitError:
            logger.warning("Skip cherry-pick due to empty commit")
            empty_revs.append(rev)
            continue
        except MergeConflictError as e:
            logger.error("Merge conflict cherry-picking %s!", rev)
            if merge_conflict_behavior is MergeConflictBehavior.SKIP:
                logger.warning("Skipping %s", rev)
                skipped_revs.append(rev)
                continue
            elif merge_conflict_behavior is MergeConflictBehavior.STOP:
                logger.warning("Stopping at revision %s", rev)
                skipped_revs.extend(list(reversed(commits_to_copy))[i:])
                break
            raise MergeConflictsError(commits=[rev]) from e
        if not pending_change:
            change_id = None
            pending_overwrite = pending_changes.get(rev)
            if pending_overwrite is not None:
                change_id = pending_changes.get(rev).change_id
            rewrite_commit_message(
                repo,
                upstream_rev=rev,
                change_id=change_id or generate_change_id(),
                prepend_subject=args.prepend_subject,
                sign_off=args.add_signed_off_by,
                keep_pseudoheaders=keep_pseudoheaders,
                additional_pseudoheaders=args.add_pseudoheader,
            )

    if repo.rev_parse() == downstream_rev:
        logger.info("Nothing to push!")
    else:
        push_refspec = get_push_refspec(args, downstream_branch)
        if not args.dry_run and not local_downstream:
            try:
                repo.push(
                    downstream_url, push_refspec, options=args.push_options
                )
            except subprocess.CalledProcessError as e:
                raise PushError(f"Failed to push to {downstream_url}") from e
        else:
            logger.info("Skip push due to dry/local run")

    emptylist = [
        repo.log(rev, fmt="%H %s", num=1).stdout.strip() for rev in empty_revs
    ]
    if emptylist:
        logger.warning(
            "The following commits were not applied because they were empty:"
        )
        for rev in emptylist:
            logger.warning("- %s", rev)

    revlist = [
        repo.log(rev, fmt="%H %s", num=1).stdout.strip() for rev in skipped_revs
    ]
    if revlist:
        logger.error(
            "The following commits were not applied due to merge conflict:"
        )
        for rev in revlist:
            logger.error("- %s", rev)
        raise MergeConflictsError(commits=skipped_revs)


def write_json_error(path: pathlib.Path, err: Exception):
    """Write out the JSON-serialized protobuf from an exception.

    Args:
        path: The Path to write to.
        err: The exception to serialize.
    """
    err_json = {}
    if err:
        if isinstance(err, CopybotFatalError):
            err_json["failure_reason"] = err.enum_name
            if err.commits:
                err_json["merge_conflicts"] = [{"hash": x} for x in err.commits]
        else:
            err_json["failure_reason"] = CopybotFatalError.enum_name
    logger.debug("JSON response: %s", err_json)
    path.write_text(json.dumps(err_json))


def main(argv=None):
    """The entry point to the program."""
    parser = argparse.ArgumentParser(description="CopyBot")
    parser.add_argument(
        "--topic",
        help="Topic to set and search in Gerrit",
        default="copybot",
    )
    parser.add_argument(
        "--label",
        help="Label to set in Gerrit (can be passed multiple times)",
        action="append",
        dest="labels",
        default=[],
    )
    parser.add_argument(
        "--re",
        help="Reviewer to set in Gerrit (can be passed multiple times)",
        action="append",
        dest="reviewers",
        default=[],
    )
    parser.add_argument(
        "--cc",
        help="CC to set in Gerrit (can be passed multiple times)",
        action="append",
        dest="ccs",
        default=[],
    )
    parser.add_argument(
        "--push-option",
        help="Add downstream push option (can be passed multiple times)",
        action="append",
        dest="push_options",
        default=[],
    )
    parser.add_argument(
        "--ht",
        help="Hashtag to set in Gerrit (can be passed multiple times)",
        action="append",
        dest="hashtags",
        default=[],
    )
    parser.add_argument(
        "--json-out",
        type=pathlib.Path,
        help="Write JSON result to this file.",
    )
    parser.add_argument(
        "--dry-run",
        help="Don't push",
        action="store_true",
    )
    parser.add_argument(
        "--prepend-subject",
        help="Prepend the subject of commits made with this string",
        default="",
    )
    parser.add_argument(
        "--exclude-file-pattern",
        help="Exclude changes to files matched by these path regexes",
        action="append",
        dest="exclude_file_patterns",
        default=[],
    )
    parser.add_argument(
        "--merge-conflict-behavior",
        help="How to handle merge conflicts",
        default="SKIP",
        choices=[behavior.name for behavior in MergeConflictBehavior],
    )
    parser.add_argument(
        "--add-signed-off-by",
        help="Add Signed-off-by pseudoheader to commit messages",
        action="store_true",
    )
    parser.add_argument(
        "--keep-pseudoheader",
        help="Keep a pseudoheader from being prefixed",
        action="append",
        dest="keep_pseudoheaders",
        default=[],
    )
    parser.add_argument(
        "--limit",
        type=int,
        default=500,
        help="Maximum number of CLs to downstream at once.  0 for infinite.",
    )
    parser.add_argument(
        "--upstream-history-limit",
        type=int,
        default=250,
        help="Maximum number of CLs in upstream history to check.  0 for"
        " infinite.",
    )
    parser.add_argument(
        "--downstream-history-limit",
        type=int,
        default=0,
        help="Maximum number of CLs in upstream history to check.  0 for"
        " infinite.",
    )
    parser.add_argument(
        "--include-downstream",
        action="append",
        default=[],
        help="Downstream include paths (relative to the subtree) separated by"
        " colons.  Note: Only supported with subtrees",
    )
    parser.add_argument(
        "--add-pseudoheader",
        action="append",
        default=[],
        help="Pseudoheaders to be added to the commit message",
    )
    parser.add_argument(
        "--skip-job-name",
        action="append",
        default=[],
        help="Skip CLs in upstream copied from the specified job name",
    )
    parser.add_argument(
        "upstream",
        help="Upstream Git URL, optionally with a branch and subtree separated"
        " by colons",
    )
    parser.add_argument(
        "downstream",
        help="Downstream Git URL, optionally with a branch and subtree"
        "separated by colons",
    )
    args = parser.parse_args(argv)

    logging.basicConfig(
        format="%(asctime)s %(levelname)s: %(message)s",
        level=logging.INFO,
    )

    if 0 < args.downstream_history_limit < args.upstream_history_limit:
        logger.warning(
            "Using a lower downstream limit than upstream limit may cause"
            " previously downstreamed changes to be chosen again."
        )

    err = None
    try:
        with tempfile.TemporaryDirectory(
            ".copybot"
        ) as git_dir, tempfile.TemporaryDirectory("_patches") as patch_dir:
            run_copybot(args, git_dir, patch_dir)
    except Exception as e:
        err = e
        raise
    finally:
        if args.json_out:
            write_json_error(args.json_out, err)


if __name__ == "__main__":
    main()
