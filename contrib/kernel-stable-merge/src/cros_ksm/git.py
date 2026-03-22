# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from __future__ import annotations

import os
import re
import secrets
import subprocess
from collections.abc import Iterator
from dataclasses import KW_ONLY, dataclass, field
from datetime import UTC, datetime, timedelta
from functools import cached_property, total_ordering
from io import StringIO
from operator import itemgetter
from os import PathLike, fspath
from pathlib import Path
from textwrap import dedent
from typing import NewType, Self

import parsy
import pygit2
from multidict import MultiDict, MultiDictProxy
from pygit2 import Oid
from pygit2.enums import FileStatus, ObjectType

import cros_ksm.cli
import cros_ksm.util
from cros_ksm import parse
from cros_ksm.util import (
    NOW,
    die,
    eprint,
    log,
    shquote_args,
)

UPSTREAM_REMOTE_NAME = "torvalds"
STABLE_REMOTE_CANON_URL = "https://git.kernel.org/pub/scm/linux/kernel/git/stable/linux.git"

DEFAULT_REMOTE_URLS = {
    "cros": "https://chromium.googlesource.com/chromiumos/third_party/kernel",
    "stable": "https://kernel.googlesource.com/pub/scm/linux/kernel/git/stable/linux.git",
    UPSTREAM_REMOTE_NAME: "https://kernel.googlesource.com/pub/scm/linux/kernel/git/torvalds/linux.git",
}

# Regex that matches a git object id. We do not need to worry about false
# matches because this partial regex is always embedded inside a larger regex
# that provides additional context.
_OID_RE = re.compile(r"(?:\b[0-9A-Za-z]{8,40}\b)")

# Type-guard for module-private constructors.
_Private = NewType("_Private", object)
_PRIVATE = _Private(object())

HexOid = NewType("HexOid", str)

def _git_subprocess_args(
    args: list[str],
    *,
    git_dir: PathLike | None = None,
    git_work_dir: PathLike | None = None,
) -> list[str]:
    full_args = ["git"]
    if git_dir is not None:
        full_args.append(f"--git-dir={git_dir}")
    if git_work_dir is not None:
        full_args.append(f"--work-tree={git_work_dir}")
    full_args += args
    return full_args

def git_output(
    args: list[str],
    *,
    git_dir: PathLike | None = None,
    git_work_dir: PathLike | None = None,
    chomp: bool = True,
    input: str | bytes | None = None,
    fail_ok: bool = False,
) -> str:
    """Wraps `subprocess.check_output`.

    Exit on failure by default. If `fail_ok` is set, then return "".
    """
    args = _git_subprocess_args(args, git_dir=git_dir,
                                git_work_dir=git_work_dir)
    try:
        out = subprocess.check_output(args, text=True, input=input)
    except subprocess.CalledProcessError as e:
        log.debug("%s", e)
        if fail_ok:
            return ""
        die(e)

    if chomp:
        out = cros_ksm.util.chomp(out)

    return out

@dataclass(eq=True, frozen=True)
class Commit:
    """Wraps `pygit2.Commit`.

    Each Commit instance is unique for a given Oid. If two Commit objects have
    the same Oid, then they are the same object.
    """

    pygit2: pygit2.Commit

    def __init__(self, priv: _Private, inner: pygit2.Commit, /):
        assert priv is _PRIVATE
        assert inner.type == ObjectType.COMMIT
        object.__setattr__(self, "pygit2", inner)

    def __str__(self):
        return str(self.id)

    @cached_property
    def id(self) -> Oid:
        return self.pygit2.id

    @cached_property
    def hex_oid(self) -> HexOid:
        return HexOid(str(self.id))

    @cached_property
    def short_id(self) -> str:
        return self.pygit2.short_id

    @cached_property
    def committer_date(self) -> datetime:
        return datetime.fromtimestamp(self.pygit2.committer.time, tz=UTC)

    @cached_property
    def message(self) -> Message:
        return Message(self.pygit2.message)

    @cached_property
    def subject(self) -> Subject:
        return self.message.subject

    @property
    def is_revert(self) -> bool:
        return (self.subject.reverted_subject is not None or
                len(self.message.reverted_oids) != 0)

@dataclass(eq=True, frozen=True)
class Tag:
    """Wraps `pygit2.Tag`.

    Each Tag instance is unique for a given Oid. If two Tag objects have the
    same Oid, then they are the same object.
    """

    repo: Repo = field(compare=False) # exclude from hash
    pygit2: pygit2.Tag

    def __init__(self, priv: _Private, repo: Repo, inner: pygit2.Tag, /):
        assert priv is _PRIVATE
        assert inner.type == ObjectType.TAG
        object.__setattr__(self, "repo", repo)
        object.__setattr__(self, "pygit2", inner)

    def __str__(self):
        return str(self.id)

    @cached_property
    def id(self) -> Oid:
        return self.pygit2.id

    @cached_property
    def name(self) -> str:
        return self.pygit2.name

    @cached_property
    def tagger_date(self) -> datetime:
        return datetime.fromtimestamp(self.pygit2.tagger.time, tz=UTC)

    @cached_property
    def commit(self) -> Commit:
        commit = self.pygit2.peel(pygit2.Commit)
        return self.repo.register_pygit2_commit(commit)

@dataclass(eq=True, frozen=True)
class Message:
    """A commit message."""

    text: str

    def __str__(self):
        return self.text

    @cached_property
    def subject(self) -> Subject:
        i = self.text.find("\n")
        if i == -1:
            i = len(self.text)
        return Subject.from_parse(self.text[:i])

    @cached_property
    def trailers(self) -> MultiDictProxy[Trailer]:
        """Trailer lines, but only the important ones.

        This contains a trailer if and only if its key is found in
        `PARSED_TRAILER_KEYS`.

        This is a read-only multi-dict. The dict keys are from
        `PARSED_TRAILER_KEYS`.  The dict value is the full, parsed trailer,
        including its original key.

        See Also:
            - man:git-interpret-trailers(1)
        """

        # We cannot use method `pygit2.Commit.message_trailers` because it has
        # a bug that is critical to our use case (as of pygit2-1.19.1). If
        # a message contains multiple trailers with the same key, then the
        # pygit2 method returns only one of the trailers because its return type
        # is `dict[str, str]`.
        #
        # The underlying libgit2 function `git_message_trailers()` does not
        # suffer from the bug. It returns a flat list of key/value pairs.
        #
        # As a workaround, parse the trailers with the git cli. This workaround
        # can be slow when searching for stable merge cherry picks, which may
        # need to parsing thousands of commits depending on the size of the
        # merge series.
        #
        # TODO: Fix performance by using libgit2 `git_message_trailers()`.

        trailers = MultiDict()
        out = git_output(
            ["interpret-trailers", "--parse"],
            input = self.text,
            chomp = True,
        )

        for line in out.split("\n"):
            trailer = Trailer.from_parse(line)
            if trailer is None:
                continue

            normal_key = trailer.key.lower()
            assert normal_key in parse.PARSED_TRAILER_KEYS

            trailers.add(normal_key, trailer)

        # return a read-only proxy
        return MultiDictProxy(trailers)

    # Regex that finds ported oids in a commit message.
    _PORTED_OID_RE = re.compile(
        fr"""^(?: # Line that frequently appears near top of message in upstream
                  # stable branches.
                  \[\s*upstream\s+commit\s+({_OID_RE.pattern})\s*\]

                  # Line near top of message. Must be bracketed with empty line
                  # above and below to avoid false capture where the line is
                  # continuation of a sentence, as in fa361565a7275cc4.
                  | \n
                    commit\s+({_OID_RE.pattern})\s+upstream\.?\s*
                    \n

                  # Standard cherry-pick line. Omit the closing paren because this
                  # may contain trailing custom info that linewraps.
                  | \(\s*cherry\s+picked\s+from\s+commit\s+({_OID_RE.pattern})\s*

                  # Trailer that identifies the non-upstream source of the commit,
                  # as in 1e207880c46da6103d7f on branch chromeos-6.12.
                  | [\w-]+-tree:\s+({_OID_RE.pattern})
              )\s*$
        """,
        re.IGNORECASE | re.MULTILINE | re.VERBOSE)

    @cached_property
    def ported_oids(self) -> set[HexOid]:
        """Parse the message for any commit ids that this commit may have been
        ported from.
        """
        cls = self.__class__
        return {
            HexOid(g)
            for m in cls._PORTED_OID_RE.finditer(self.text)
            for g in m.groups()
            if g is not None
        }

    # Regex that finds reverted oids in a commit message.
    _REVERTED_OID_RE = re.compile(
        fr"""(?:# Line that frequently appears near top of message in upstream stable
                # branches. The reverted oid may be from upstream or from the stable
                # branch.
                (?:\bthis\s+)reverts?\s+(?:commit\s+)?({_OID_RE.pattern})\s*

                # Sometimes the above line continues to specify the upstream commit
                # oid too.
                (?:,?\s+which\s+is\s+
                    (?: ({_OID_RE.pattern})\s+upstream
                        | commit\s+({_OID_RE.pattern})\s+upstream
                        | upstream\s+commit\s+({_OID_RE.pattern})
                        | upstream\s+({_OID_RE.pattern})
                    )
                )?
              )

              # A pattern that was common in older stable commits, but still appears
              # in some recent commits.
              | (?:^Reverts?\s+'commit\s+({_OID_RE.pattern}))
        """,
        re.IGNORECASE | re.MULTILINE | re.VERBOSE)

    @cached_property
    def reverted_oids(self) -> set[HexOid]:
        """Parse the message for any commit ids that this commit may be a revert
        of.
        """
        cls = self.__class__
        return {
            HexOid(g)
            for m in cls._REVERTED_OID_RE.finditer(self.text)
            for g in m.groups()
            if g is not None
        }

@dataclass
class Trailer:

    text: str
    key: str
    value: str
    oid: HexOid | None
    subject: Subject | None

    @classmethod
    def from_parse(cls, text: str) -> Self | None:
        try:
            p = parse.trailer.parse(text)
        except parsy.ParseError:
            return None
        else:
            return cls(p)

    def __init__(self, p: parse.Trailer):
        self.text = p.source.text
        self.key = p.key
        self.value = p.value

        self.oid = None
        if p.oid is not None:
            self.oid = HexOid(p.oid)

        self.subject = None
        if p.subject is not None:
            self.subject = Subject(p.subject)

    def __str__(self):
        return self.text

@dataclass(eq=True, frozen=True)
class Subject:
    """Subject of a commit message."""

    _parsed: parse.Subject

    @classmethod
    def from_parse(cls, text: str) -> Self:
        subject = cls(parse.subject.parse(text))
        assert subject.text == text
        return subject

    def __str__(self):
        return self.text

    @cached_property
    def text(self) -> str:
        return self._parsed.source.text

    @cached_property
    def reverted_subject(self) -> Self | None:
        # Strip any leading port prefixes
        p = self._parsed

        while p.inner_type == parse.SubjectInnerType.PORT:
            assert p.inner_subject is not None
            p = p.inner_subject

        if p.inner_type == parse.SubjectInnerType.REVERT:
            assert p.inner_subject is not None
            return Subject(p.inner_subject)

        return None

    @cached_property
    def ported_subject(self) -> Self:
        """The subject without modifications often found in backported and
        cherry picked commits.
        """
        p = self._parsed
        if p.inner_type == parse.SubjectInnerType.PORT:
            assert p.inner_subject is not None
            return Subject(p.inner_subject)
        else:
            return self

@total_ordering
@dataclass(frozen=True, eq=True)
class KernelVersion:
    """Version of an upstream kernel release tag.

    This supports only version formats "w.x" and "w.x.y". It does not support
    "rc" versions.

    Using this class rather than a raw version string provides several benefits:
        - Correct version sorting.
        - Type safety and informative type annotations.
        - Validation when ingesting version strings.

    Why not use the popular `semver` package rather than implementing a version
    class from scratch?  The kernel's version scheme differs from the semver
    scheme. Using `semver` would require a wrapper class, which requires as much
    code as a direct implementation.
    """

    w: int
    x: int
    y: int = 0

    _TAG_RE = re.compile(
        r"""v
            (?P<w>[1-9]\d*)
            \.(?P<x>0|[1-9]\d*)
            (?:
                \.(?P<y>0|[1-9]\d*)
            )?
        """,
        re.VERBOSE)

    _MAJOR_RE = re.compile(
        r"""\bv?
            (?P<w>[1-9]\d*)
            \.(?P<x>0|[1-9]\d*)
            \b
        """,
        re.VERBOSE)

    @classmethod
    def _from_match(cls, m: re.Match) -> Self:
        w = int(m["w"])
        x = int(m["x"])

        y =  0
        if m["y"] is not None:
            y = int(m["y"])

        return cls(w, x, y)

    @classmethod
    def parse_tag_name(cls, text: str) -> Self | None:
        """Return None on failure."""
        m = cls._TAG_RE.fullmatch(text)
        if m is None:
            return None
        return cls._from_match(m)

    @classmethod
    def parse_tag_name_checked(cls, text: str) -> Self:
        """Like `parse_tag_name` but raise on failure."""
        kver = cls.parse_tag_name(text)
        if kver is None:
            raise ValueError(f"invalid kernel release tag name: {text!r}")
        return kver

    @classmethod
    def parse_major_exact(cls, text: str) -> Self | None:
        """Parse a major version.

        The text must exactly match a major version.

        Return None on failure.
        """
        m = cls._MAJOR_RE.fullmatch(text)
        if m is None:
            return None
        return cls(int(m["w"]), int(m["x"]))

    @classmethod
    def parse_major_fuzzy(cls, text: str) -> Self | None:
        """Fuzzy parse a major version.

        The text may include extra.

        Return None on failure.
        """
        m = cls._MAJOR_RE.search(text)
        if m is None:
            return None
        return cls(int(m["w"]), int(m["x"]))

    def __post_init__(self):
        if (self.w <= 0 or
            self.x < 0 or
            self.y < 0
        ):
            raise ValueError

    def __str__(self):
        buf = StringIO()
        buf.write(f"{self.w}.{self.x}")
        if self.y > 0:
            buf.write(f".{self.y}")
        return buf.getvalue()

    def cmp_key(self):
        return (self.w, self.x, self.y)

    def __lt__(self, other):
        if not isinstance(other, self.__class__):
            raise NotImplementedError
        return self.cmp_key() < other.cmp_key()

    def __gt__(self, other):
        if not isinstance(other, self.__class__):
            raise NotImplementedError
        return self.cmp_key() > other.cmp_key()

    @cached_property
    def tag_name(self) -> str:
        return f"v{self}"

    @cached_property
    def major(self) -> KernelVersion:
        """Return this version's major version."""
        return KernelVersion(self.w, self.x)

class Repo:
    """A ChromeOS kernel git repo. Wraps pygit2.Repository."""

    pygit2: pygit2.Repository
    kernel_major_version: KernelVersion
    upstream_branch: str
    stable_branch: str
    cros_branch: str

    # A store of every commit and tag instance created from this repo. This
    # enforces uniqueness of commit and tag objects with respect to oids, which
    # reduces overhead when exploring the commit history when searching for
    # cherry picks.
    __oid_to_commit: dict[Oid, Commit]
    __oid_to_tag: dict[Oid, Tag]

    def __init__(
        self,
        *,
        git_dir: PathLike | str | None = None,
        kernel_major_version: KernelVersion | None = None,
    ):
        """Open an existing git repo.

        When searching for and opening the repo, try to respect the same
        environment vars as cli git.

        Die if a repo is not found or it fails to open.
        """

        # Git gets confused by the kernel repos in the ChromeOS tree because of
        # the non-standard symlinks created by the gerrit repo tool.  Both cli
        # git and libgit2 get confused.
        #
        # gerrit-repo creates a symlink from
        # "$CROS_TOP/src/third_party/kernel/{version}/.git" to to
        # "$CROS_TOP/.repo/projects/src/third_party/kernel/{version}.git".  Git
        # does not expect its gitdir to be a symlink.  gerrit-repo chooses to
        # use a symlink to make it possible to delete the working tree without
        # losing the git repo. However, standard git would instead use a gitlink
        # file instead of a symlink.
        #
        # As a consequence of the gitdir symlink, cli git and libgit2 will infer
        # the incorrect path to the working tree when environment vars like
        # GIT_DIR and GIT_WORK_TREE and cmdline options like --git-dir and
        # --work-tree are in use.
        #
        # To work around the symlink problem, we query the path to the gitdir
        # and to the working tree using git-rev-parse, and then feed the paths
        # to libgit2. ([linyaa: I cannot find a workaround that uses only
        # libgit2 without the cli).

        if git_dir is None:
            git_dir = git_output(["rev-parse", "--git-dir"])
        git_dir = Path(git_dir)

        # Respect the environment variables used by the git cli.
        flags = pygit2.enums.RepositoryOpenFlag.FROM_ENV

        # We have already found the exact git-dir, therefore...

        # Do not search upwards for the git repo.
        flags |= pygit2.enums.RepositoryOpenFlag.NO_SEARCH

        # Do not append `/.git` to search path.
        flags |= pygit2.enums.RepositoryOpenFlag.NO_DOTGIT

        try:
            self.pygit2 = pygit2.Repository(git_dir, flags)
        except pygit2.GitError as e:
            die(e)

        # Override the workdir discovered by libgit2, to workaround the symlink
        # problem.
        if not self.pygit2.is_bare:
            self.pygit2.workdir = git_output(["rev-parse", "--show-toplevel"])

        if kernel_major_version is not None:
            # Overwrite cached_property slot
            self.kernel_major_version = kernel_major_version

        kmajor = self.kernel_major_version

        self.upstream_branch = f"refs/remotes/{UPSTREAM_REMOTE_NAME}/master"
        self.stable_branch = f"refs/remotes/stable/linux-{kmajor.w}.{kmajor.x}.y"
        self.cros_branch = f"refs/remotes/cros/chromeos-{kmajor.w}.{kmajor.x}"

        self.__oid_to_commit = {}
        self.__oid_to_tag = {}

        log.debug("repo: git dir: %s", self.pygit2.path)
        log.debug("repo: work dir: %s", self.pygit2.workdir)
        log.debug("repo: bare: %s", self.pygit2.is_bare)
        log.debug("repo: kernel major version: %s", self.kernel_major_version)
        log.debug("repo: stable branch: %s", self.stable_branch)
        log.debug("repo: cros branch: %s", self.cros_branch)

    @cached_property
    def git_dir(self) -> Path:
        return Path(self.pygit2.path)

    @cached_property
    def work_dir(self) -> Path | None:
        w = self.pygit2.workdir
        if w is None:
            return None
        return Path(w)

    @cached_property
    def kernel_major_version(self) -> KernelVersion:
        """The repo's inferred kernel major version, formatted as `w.x`.

        Die if version cannot be inferred.
        """

        if self.pygit2.is_bare:
            path = self.git_dir
        else:
            assert self.work_dir is not None
            path = self.work_dir

        kmajor = KernelVersion.parse_major_fuzzy(path.name)
        if kmajor is None:
            die("error: unable to infer kernel major version of git repo")

        return kmajor

    def get_cros_ksm_bug(self) -> str:
        """Get the bug id that should be inserted into commit messages.

        Exit on failure.
        """

        bug = os.environ.get("CROS_KSM_BUG", "")
        if bug:
            return bug

        try:
            bug = self.pygit2.config["cros-ksm.bug"]
        except KeyError:
            bug = ""

        if bug:
            return bug

        die("error: neither git config 'cros-ksm.bug' nor environment var "
            "CROS_KSM_BUG contain a bug id")

    def git_call(self, args: list[str]) -> None:
        """See module-level function `git_call`."""

        eprint("git " + shquote_args(args))

        args = _git_subprocess_args(args, git_dir=self.git_dir,
                                    git_work_dir=self.work_dir)
        try:
            subprocess.check_call(args)
        except subprocess.CalledProcessError as e:
            die(e)

    def git_var(self, name: str) -> str:
        value = self.git_output(["var", name])
        if not value:
            die(f"failed to query git var {name}")
        return value

    def git_output(
            self,
            args: list[str],
            *,
            chomp: bool = True,
            input: str | bytes | None = None,
            fail_ok: bool = False,
        ) -> str:
        """See module-level function `git_output`."""
        return git_output(args, git_dir=self.git_dir,
                          git_work_dir=self.work_dir,
                          chomp=chomp,
                          input=input,
                          fail_ok=fail_ok)

    def git_path(self, path: str | PathLike) -> Path:
        """Resolve the path under the repo's git dir.

        Returned path may not be "{git_dir}/{path}". The path may be affected by
        git worktrees and git config options. See command `git rev-parse
        --git-path`.
        """
        # The syntax of the git-rev-parse command has unusual corner cases.
        #   - The path argument must not be attached to --git-path with '='.
        #   - The flag --end-of-options must not be used with --git-path.
        #   - This works even if the path argument begins with '-'.
        return Path(self.git_output(["rev-parse", "--git-path", fspath(path)]))

    def register_pygit2_commit(self, pygit2_commit: pygit2.Commit, /) -> Commit:
        """Wrap a `pygit2.Commit` instance with our `Commit` class.

        All instances of `Commit` must be created by this method. This enforces
        uniqueness of Commit objects with respect to git object ids.
        """
        oid = pygit2_commit.id
        commit = self.__oid_to_commit.get(oid)
        if commit is None:
            commit = Commit(_PRIVATE, pygit2_commit)
            self.__oid_to_commit[oid] = commit
        return commit

    def register_pygit2_tag(self, pygit2_tag: pygit2.Tag, /) -> Tag:
        """Wrap a `pygit2.Tag` instance with our `Tag` class.

        All instances of `Tag` must be created by this method. This enforces
        uniqueness of Tag objects with respect to git object ids.
        """
        oid = pygit2_tag.id
        tag = self.__oid_to_tag.get(oid)
        if tag is None:
            tag = Tag(_PRIVATE, self, pygit2_tag)
            self.__oid_to_tag[oid] = tag
        return tag

    def is_ancestor(self, a: Oid, b: Oid) -> bool:
        """Return true if commit A is an ancestor of commit B.

        Simulates `git merge-base --is-ancestor A B`.
        """
        return a == self.pygit2.merge_base(a, b)

    def in_revision_range(
        self,
        needle: Oid,
        *,
        include: list[Oid],
        exclude: list[Oid] | None = None,
    ) -> bool:
        """Search for `needle` in a range of revisions.

        Warning: Counter to intuition, adding revisions to the exclude list may
        slow down the search.
        """

        walk = self.pygit2.walk(None)

        for oid in include:
            walk.push(oid)

        if exclude is not None:
            for oid in exclude:
                walk.hide(oid)

        for commit in walk:
            if needle == commit.id:
                return True

        return False

    def rev_parse_verify(self, rev: str) -> str:
        """Verify a rev with `git rev-parse --verify`.

        If rev exists, return its oid. Else, return the empty string.

        Sometimes we need to fallback to the git cli in order to resolve a rev
        due to incompleteness in libgit2. In particular, pygit2-1.19.1 cannot
        resolve CHERRY_PICK_HEAD.
        """
        return self.git_output(["rev-parse", "--verify", "--quiet",
                                "--end-of-options", rev], fail_ok=True)

    def try_resolve_refish_to_commit(self, refish: str) -> Commit | None:
        """Resolve a refish to a commit. Return None on lookup failure."""
        try:
            obj, _ = self.pygit2.resolve_refish(refish)
        except KeyError:
            return None

        # `pygit2.resolve_refish` incorrectly claims (in its docstring and type
        # signature) that it always returns a commit object. Actually it also
        # returns tag objects, which we must peel down to a commit.
        #
        # See issue <https://github.com/libgit2/pygit2/issues/1414>.
        try:
            obj = obj.peel(ObjectType.COMMIT)
        except pygit2.InvalidSpecError:
            return None

        log.debug("resolved refish %r to commit %s", refish, obj.id)
        return self.register_pygit2_commit(obj)

    def resolve_refish_to_commit(self, refish: str) -> Commit:
        """Resolve a refish to a commit. Exit on failure."""
        commit = self.try_resolve_refish_to_commit(refish)
        if commit is None:
            die(f"error: failed to resolve refish {refish!r} to a commit")
        return commit

    def try_resolve_tag_name(self, name: str) -> Tag | None:
        refname = f"refs/tags/{name}"

        try:
            ref = self.pygit2.lookup_reference(refname)
        except KeyError:
            return None

        try:
            obj = ref.peel(pygit2.Tag)
        except pygit2.InvalidSpecError:
            return None

        return self.register_pygit2_tag(obj)

    def resolve_tag_name(self, name: str) -> Tag:
        """Exit on failure."""
        tag = self.try_resolve_tag_name(name)
        if tag is None:
            die(f"error: failed to resolve {name!r} to a tag")
        return tag

    def get_last_merged_release(self, rev: str) -> KernelVersion:
        """Find the last kernel release merged in the history of `rev`."""

        head_rev = rev
        head_commit = self.resolve_refish_to_commit(head_rev)

        for commit in self.pygit2.walk(head_commit.id):
            # ruff: ignore[redefined-loop-name]
            commit = self.register_pygit2_commit(commit)

            m = re.search(r"Merge\b.*'([^']+)'", commit.subject.text)
            if m is None:
                continue

            merge_src = m[1]

            if commit.is_revert:
                eprint(f"error: found revert of merge at commit {commit} {commit.subject}")
                eprint("error: unable to continue parsing git history")
                die()

            # We do not verify that the commit's second parent is the expected
            # tag because:
            #
            #   1. The repo may lack the tag because it has not yet been
            #      fetched.
            #
            #   2. In ChromeOS, sometimes the CQ merges the tag and then merges
            #      again into the target branch. In this case, the expected tag
            #      would not be commit^2 but be commit^2^2, or something even
            #      stranger. We could still verify this case by repeatedly
            #      peeling HEAD^2 until we found the expected tag, but :shrug:.
            #      A real example is the commit:
            #
            #      6a2d352914788061 Merge "CHROMIUM: Merge 'v5.15.118' into chromeos-5.15" into chromeos-5.15

            kver = KernelVersion.parse_tag_name(merge_src)
            if kver is None:
                continue

            if kver.major != self.kernel_major_version:
                eprint(cros_ksm.cli.wrap_text(dedent(f"""\
                    error: last merged release '{merge_src}' at commit {commit}
                    does not match repo's kernel major version
                    {self.kernel_major_version}
                    """),
                    subsequent_indent = "  ",
                ))

            return kver

        die(f"failed to find a kernel release merge under git rev {head_rev!r}")

    def iter_kmajor_release_tags(self) -> Iterator[tuple[KernelVersion, Tag]]:
        """Iterate over all tags that match the repo's kernel major version."""

        for tag_ref in self.pygit2.references.iterator(
                pygit2.enums.ReferenceFilter.TAGS):
            kver = KernelVersion.parse_tag_name(tag_ref.shorthand)
            if kver is None:
                continue
            if kver.major != self.kernel_major_version:
                continue
            tag = self.register_pygit2_tag(tag_ref.peel(pygit2.Tag))
            assert tag.name == tag_ref.shorthand
            assert tag.name == kver.tag_name
            yield (kver, tag)

    def get_target_release_info(self) -> TargetReleaseSearchInfo:
        """Choose the target release for the next stable merge."""

        # Choose the most recent release whose age is at least min_age and is
        # not the newest release.

        min_age = timedelta(weeks=4)
        max_date = NOW - min_age

        merged_kver: KernelVersion = self.get_last_merged_release(self.cros_branch)
        merged_tag: Tag = self.resolve_tag_name(merged_kver.tag_name)

        # Collect tags that match the kernel major version
        kver_tags: list[tuple[KernelVersion, Tag]] = list(self.iter_kmajor_release_tags())

        # Sort by release version
        kver_tags.sort(reverse=True, key=itemgetter(0))

        # Skip the newest release
        newest_tag = kver_tags.pop(0)[1]

        target_tag = None

        for (kver, tag) in kver_tags:
            if tag.tagger_date > max_date:
                continue

            if kver > merged_kver:
                target_tag = tag

            break

        return TargetReleaseSearchInfo(
            kernel_major_version = self.kernel_major_version,
            last_merged_release_tag = merged_tag,
            newest_release_tag = newest_tag,
            now = NOW,
            search_min_age = min_age,
            search_max_date = max_date,
            target_release_tag = target_tag,
        )

    def get_remote(self, name: str, /) -> Remote:
        try:
            remote = self.pygit2.remotes[name]
        except KeyError:
            # Automatically create the remote
            url = DEFAULT_REMOTE_URLS.get(name)
            if url is None:
                die(f"error: git remote {name!r} does not exist")

            eprint(f"git remote add {name} {url}")
            remote = self.pygit2.remotes.create(name, url)

        return Remote(self, remote)

    def get_committer(self) -> str:
        """Return the "{name} {email}" component of GIT_COMMITTER_IDENT."""

        # Query git-var, not git-config, because only git-var respects the
        # relevant environment variables such as GIT_COMMITTER_NAME and
        # GIT_COMMITTER_EMAIL.
        x = self.git_var("GIT_COMMITTER_IDENT")

        m = re.match(r"^(.*)\s+\d+\s+[+-]?[0-9]+\s*$", x)
        if m is None:
            die(f"GIT_COMMITTER_IDENT has unexpected format: {x!r}")

        # Strip the trailing timestamp. Keep only the name and email.
        return m.expand(r"\1")

    def has_status_conflicts(self) -> bool:
        """Return true if any file has unmerged status."""
        status = self.pygit2.status(untracked_files="no", ignored=False)
        for flag in status.values():
            if flag == FileStatus.CONFLICTED:
                return True
        return False

@dataclass(eq=False)
class Remote:
    """Wraps `pygit2.Remote`.

    A wrapper is needed because `pygit2.Remote` is buggy. For example,
    pygit2-1.19.1 cannot handle custom url schemes that depend on git remote
    helpers, which are common inside Google, such as 'persistent-https://'. (See
    `git help remote-helpers`).
    """

    repo: Repo
    pygit2: pygit2.Remote

    @cached_property
    def name(self) -> str:
        assert self.pygit2.name is not None
        return self.pygit2.name

    def list_tags(self) -> list[str]:
        """List all remote tag names that match refspec "refs/tags/*"."""
        out = self.repo.git_output([
            "ls-remote",

            # Limit to only local tags.
            "--tags",

            # Do not show peeled tags (like `refs/tags/foo^{}`) or pseudorefs
            # (like HEAD).
            "--refs",

            "--",
            self.name,
            "refs/tags/*",
        ])
        prefix_len = len("refs/tags/")
        return [
            line.split()[1][prefix_len:]
            for line in out.splitlines()
        ]

@dataclass
class TargetReleaseSearchInfo:
    """Info collected by the search for the next target release."""

    _: KW_ONLY

    kernel_major_version: KernelVersion
    now: datetime
    last_merged_release_tag: Tag
    newest_release_tag: Tag
    search_min_age: timedelta
    search_max_date: datetime
    target_release_tag: Tag | None

def make_gerrit_change_id() -> str:
    return "I" + secrets.token_hex(nbytes=20)
