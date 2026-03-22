# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import json
import os
import re
import subprocess
import sys
from collections.abc import Callable
from dataclasses import KW_ONLY, dataclass
from io import StringIO
from pathlib import Path
from textwrap import dedent

import click
from click_aliases import ClickAliasedGroup

if not __package__:
    # Make the cli runnable from the source tree without installing it as
    # a package.
    #
    # This hack is needed only by projects that use the "src" layout, as opposed
    # to the "flat" layout. See
    # <https://packaging.python.org/en/latest/discussions/src-layout-vs-flat-layout>.
    sys.path.insert(0, os.path.dirname(os.path.dirname(__file__)))

# Import for side-effects
import cros_ksm.cli  # ruff: ignore[unsorted-imports,unused-import]

import cros_ksm.merge
import cros_ksm.wrap_git_editor
from cros_ksm.find_cherry_picks import find_cherry_picks
from cros_ksm.git import (
    STABLE_REMOTE_CANON_URL,
    UPSTREAM_REMOTE_NAME,
    KernelVersion,
    Repo,
    make_gerrit_change_id,
)
from cros_ksm.util import (
    args_add_if,
    assert_unreachable,
    chomp,
    die,
    eprint,
    format_utc,
    log,
    shquote_args,
)


@dataclass
class CliContextObject:
    """The type of click's `ctx.obj`.

    In click, the "context object" is a custom object for app authors to use
    during command line parsing and command invocation.
    """

    _: KW_ONLY
    kmajor: KernelVersion | None = None

def cli_open_repo(ctx: click.Context) -> Repo:
    """Open the kernel repo, respecting cli options and env vars."""
    return Repo(kernel_major_version=ctx.obj.kmajor)

def option_kernel_major_version(func: Callable):
    """(decorator) Add option 'kernel_major_version'.

    The option value is not passed to the command invocation function. Instead,
    the value is assigned to the `ctx.obj`.
    """

    def param_callback(ctx, _param, value):
        ctx.obj.kmajor = None
        if not value:
            return

        kmajor = KernelVersion.parse_major_exact(value)
        if kmajor is not None:
            ctx.obj.kmajor = kmajor
            return

        raise click.BadParameter(f"invalid kernel major version: {value}")

    return click.option(
        "kernel_major_version",
        "--kmajor", "--kernel-major-version",
        metavar = "<kmajor>",
        envvar = "CROS_KSM_KMAJOR",

        # Do not pass this param to the command invocation function.
        expose_value = False,

        help = dedent("""\
            Override the inferred kernel major version of the git repo.
            This setting can also be overridden by environment variable
            CROS_KSM_KMAJOR.
        """),
        callback = param_callback,
    )(func)

option_rerere_autoupdate = click.option(
    "--rerere-autoupdate / --no-rerere-autoupdate",
    default = False,
    help = dedent("""\
        Passed to the underlying git command. Default is 'no', to allow
        inspection of the automated conflict resolution before committing.
        (This command intentionally does not respect git config option
        'rerere.autoUpdate').
    """),
)

option_edit_message = click.option(
    "edit_message",
    "-e", "--edit / --no-edit",
    default = True,
    help = dedent("""\
        Like the same flag of command git-commit. Edit the commit message
        before creating the commit.
    """),
)

# The toplevel command. When no subcommand is given, then it prints a summary of
# available subcommands, similar to the git cli.
@click.group(
    # This class override allows us to define aliases for subcommands. It adds
    # the keyword param 'aliases' to the group's `@command` decorator.
    cls = ClickAliasedGroup,

    context_settings = {
        # Click adds '--help' by default. Also add '-h'.
        "help_option_names": ["-h", "--help"],
    },
)
@click.pass_context
def main(ctx):
    # Ensure that `ctx.obj` exists and it has the given type.
    ctx.ensure_object(CliContextObject)

@main.command(
    name = "kernel-major-version",
    aliases = ["kmajor"],
    help = dedent("""\
        Print the kernel major version of the git repo

        The version is inferred from the git repo's filesystem path. The printed
        format omits the 'v' prefix. For example, the command prints "6.1" not
        "v6.1".

        This tool is not useful by itself. It is intended to be used as a helper
        in shell scripts.
    """),
)
@option_kernel_major_version
@click.pass_context
def cmd_kernel_major_version(ctx: click.Context, /):
    repo = cli_open_repo(ctx)
    print(repo.kernel_major_version)

@main.command(
    name = "fetch",
    help = dedent(f"""
        Fetch from relevant remotes

        Fetch the remotes 'cros', 'stable', '{UPSTREAM_REMOTE_NAME}'. If
        a remote does not exist, it will be created.

        It's often necessary to run this command before starting a stable merge
        because many cros-ksm commands inspect only the locally fetched remotes
        and tags.
    """),
)
@option_kernel_major_version
@click.pass_context
def cmd_fetch(ctx: click.Context, /):
    repo = cli_open_repo(ctx)
    remotes = ("cros", "stable", UPSTREAM_REMOTE_NAME)

    # Ensure each remote exists
    for name in remotes:
        _remote = repo.get_remote(name)

    repo.git_call(["fetch", "--multi", *remotes])

@main.command(
    name = "last-merged-release",
    help = dedent("""\
        Print name of the last merged kernel release

        Search for last merged kernel release in the history of <rev>
        and print its name without the 'v' prefix. For example, the command
        prints "6.1.17" and not "v6.1.17".

        If <rev> is omitted, it defaults to "refs/remotes/cros/chromeos-W.X",
        where W.X is the repo's kernel major version.
    """),
)
@click.argument(
    "rev",
    metavar = "[<rev>]",
    required = False,
)
@option_kernel_major_version
@click.pass_context
def cmd_last_merged_release(
    ctx: click.Context,
    /, *,
    rev: str | None,
):
    repo = cli_open_repo(ctx)
    if rev is None:
        rev = repo.cros_branch
    kver: KernelVersion = repo.get_last_merged_release(rev)
    print(kver)

@main.command(
    name = "choose-release",
    help = dedent("""\
        Choose release tag for stable merge

        Choose a release tag and print the tag name. Only locally fetched
        release tags are searched. The heuristic for choosing the tag is an
        implementation detail.

        If the chosen tag has already been merged, then print nothing to stdout
        and still exit with success.

        [BUG] This command never chooses the newest tag in a release
        series. Normally, this is the desired behavior. However, after a stable
        release branch has reached end of life, then this command will never
        choose the stable branch's final tag.
        """),
)
@click.option(
    "-v", "--verbose",
    is_flag = True,
)
@option_kernel_major_version
@click.pass_context
def cmd_choose_release(
    ctx: click.Context,
    /, *,
    verbose: bool,
):
    repo = cli_open_repo(ctx)
    info = repo.get_target_release_info()

    if verbose:
        def eprint_tag(pretty_name, tag):
            if tag is None:
                eprint(f"{pretty_name}: none")
            else:
                eprint(f"{pretty_name}: {tag.name} {format_utc(tag.tagger_date)}")

        eprint(f"kernel major version: {info.kernel_major_version}")
        eprint(f"now: {format_utc(info.now)}")
        eprint(f"search min age: {info.search_min_age}")
        eprint(f"search max date: {format_utc(info.search_max_date)}")
        eprint_tag("last merged release tag", info.last_merged_release_tag)
        eprint_tag("target release tag", info.target_release_tag)
        eprint_tag("newest release tag", info.newest_release_tag)

    if info.target_release_tag is None:
        eprint("no target release was found")
        return

    print(info.target_release_tag.name)

@main.command(
    name = "prepare-remote",
    help = dedent("""\
        Prepare remote 'cros' for receiving stable merge CLs

        Push all release tags for the kernel major series that are missing on
        the remote, up to the stable merge's target release tag. And update the
        remote ref for the stable branch.
    """),
)
@click.option(
    "-n", "--dry-run",
    is_flag = True,
)
@option_kernel_major_version
@click.pass_context
def cmd_prepare_remote(
    ctx: click.Context,
    /, *,
    dry_run: bool,
):
    repo = cli_open_repo(ctx)
    cros_remote = repo.get_remote("cros")
    target_tag = repo.get_target_release_info().target_release_tag
    if target_tag is None:
        eprint("no target release was found")
        return
    target_kver = KernelVersion.parse_tag_name_checked(target_tag.name)

    # Tags that will be pushed
    push_tags: set[str] = set()

    # Tentatively add to the push set all tags in the release series, up to the
    # target release. We generate the tag names explicitly, rather than discover
    # them by listing the tags in the repo, because we want git-push to fail if
    # any of the tags are missing locally. After the push completes, we want the
    # remote to contain all the desired tags.
    for y in range(1 + target_kver.y):
        w = repo.kernel_major_version.w
        x = repo.kernel_major_version.x
        kver = KernelVersion(w, x, y)
        if kver > target_kver:
            break
        push_tags.add(kver.tag_name)

    # Do not push tags that already exist on the remote. The ChromeOS Gerrit
    # server is slow when receiving many refs, so avoid pushing unnecessary
    # ones.
    cros_remote_tags: set[str] = set(cros_remote.list_tags())
    push_tags -= cros_remote_tags

    # Before pushing the stable branch, check that it is up-to-date, that it
    # contains the target tag.
    stable_commit = repo.resolve_refish_to_commit(str(repo.stable_branch))
    if not repo.is_ancestor(target_tag.commit.id, stable_commit.id):
        die(f"error: branch {repo.stable_branch} does not contain "
            f"target release tag {target_tag.name}")

    git_args = [
        "push",

        # Push exactly the requested refspecs and nothing else. Do not
        # accidentally push additional tags.
        "--no-follow-tags",

        # The ChromeOS kernel repo is configured to prevent pushing directly to
        # branches and tags. Override the safeguard with the justification code
        # for the stable merge workflow.
        "-opush-justification=b/338297531",

        # The Gerrit server will reject commits that contain banned words.
        # Override the banned word check because we are pushing only upstream
        # commits.
        "-obanned-words~skip",
    ]

    if dry_run:
        git_args += ["-n"]

    stable_branch_name = os.path.basename(repo.stable_branch)

    git_args += [
        cros_remote.name,
        f"{repo.stable_branch}:refs/heads/stable-merge/{stable_branch_name}",
        *sorted(push_tags),
    ]

    repo.git_call(git_args)

@main.command(
    name = "tig-split",
    help = dedent("""\
        Explore divergent history of files with tig and tmux

        Create a new tmux window with two panes. The left pane contains the
        "local-only" history of <file>. The right pane contains the
        "remote-only" history of <file>.

        If <file> is omitted, then the command chooses the first unmerged file
        in the index.  This behavior complements the git-mergetool command.  If
        no unmerged files exist, then <file> is required.

        The local-only history is the revision range `LOCAL_REV ^REMOTE_REV`,
        and the remote-only history is the inverse, `REMOTE_REV ^LOCAL_REV`. The
        definitions of LOCAL_REV and REMOTE_REV depend on the state of the git
        repo. When resolving a merge conflict, then they are ORIG_HEAD and
        MERGE_HEAD. When resolving a cherry-pick conflict, they are ORIG_HEAD
        and CHERRY_PICK_HEAD. When no conflict resolution is current but HEAD is
        a merge commit, then they are HEAD^1 and HEAD^2.  Otherwise, the revs
        must be specified with options --local-rev and --remote-rev.

        Note: In git revision syntax, "^A" means "exclude commits reachable from
        A". See man:git-revisions for details.

        [BUG] This command must be run inside a tmux session. When run outside
        a tmux session, the command fails when it should instead create
        a temporary session in the current terminal.
    """),
)
@click.argument(
    "files",
    metavar = "[<file>...]",
    nargs = -1,
    default = [],
)
@click.option(
    "-r", "--remote-rev",
    metavar = "REMOTE_REV",
    help = "Inferred if omitted.",
)
@click.option(
    "-l", "--local-rev",
    metavar = "LOCAL_REV",
    help = "Inferred if omitted.",
)
@click.option(
    "-R", "--remote-full-history",
    is_flag = True,
    help = dedent("""\
        In the pane that inspects remote history, do not exclude local
        history. That is, do not add ^LOCAL_REV to the revision set.
    """),
)
@click.option(
    "--no-merges",
    is_flag = True,
    help = dedent("""\
        Do not show merge commits. (See `git log --no-merges`).
    """),
)
@option_kernel_major_version
@click.pass_context
# ruff: ignore[too-many-branches]
# ruff: ignore[too-many-statements]
def cmd_tig_split(
    ctx: click.Context,
    /, *,
    local_rev: str | None,
    remote_rev: str | None,
    remote_full_history: bool,
    no_merges: bool,
    files: list[str],
):
    repo = cli_open_repo(ctx)

    if (remote_rev is None) != (local_rev is None):
        ctx.fail("if either --remote-rev or --local-rev are given, "
                 "then both are required")

    # Choose remote rev
    if remote_rev is not None:
        # Verify the user-provided rev to ensure the tig args are sanitary.
        if not repo.rev_parse_verify(remote_rev):
            die(f"failed to resolve remote rev {remote_rev!r} to a commit")
    else:
        for rev in ("MERGE_HEAD", "CHERRY_PICK_HEAD", "HEAD^2"):
            log.debug("try fallback to %s as remote rev", rev)
            if repo.rev_parse_verify(rev):
                remote_rev = rev
                break

    if remote_rev is None:
        die("error: failed to infer <remote-rev>")

    # Choose local rev
    if local_rev is not None:
        # Verify the user-provided rev to ensure the tig args are sanitary.
        if not repo.rev_parse_verify(local_rev):
            die(f"failed to resolve local rev {local_rev!r} to a commit")
    else:
        # ORIG_HEAD may exist as a leftover file that hasn't yet been cleaned
        # up.  Only use it if the repo has an in-progress merge or cherry-pick.
        if remote_rev in {"MERGE_HEAD", "CHERRY_PICK_HEAD"}:
            rev = "ORIG_HEAD"
        elif remote_rev == "HEAD^2":
            rev = "HEAD^1"
        else:
            assert False

        log.debug("try fallback to %s as local rev", local_rev)
        if repo.rev_parse_verify(rev):
            local_rev = rev

    if local_rev is None:
        die("error: failed to infer <local-rev>")

    if not files:
        # Choose the first unmerged file.
        #
        # Take care to choose the same file as git-mergetool. The user expects
        # cros-ksm-tig-split and git-mergetool to inspect the same file by
        # default.
        #
        # The git-mergetool manpage says it respects config value
        # `diff.orderFile`. Therefore we choose the first file listed by
        # git-diff.
        out = repo.git_output(["diff", "-z", "--name-only", "--diff-filter=U"])

        # Drop the terminal null
        out = out.removesuffix("\0")

        # Take the first file, if it exists
        files = out.split("\0")[0:1]
        if not files:
            die("error: argument <file> is required because no unmerged "
                "files exist")

    log.debug("remote rev: %s", remote_rev)
    log.debug("local rev: %s", local_rev)
    log.debug("files: %s", files)

    def wrap_tig_args(tig_args) -> str:
        tig_qargs: str = shquote_args(tig_args)

        # Prevent the tmux pane from immediately closing when tig fails to find
        # any matching revisions to show.
        sh_args = ["sh", "-c", tig_qargs + "|| cros-ksm press-q"]
        sh_qargs: str = shquote_args(sh_args)

        return sh_qargs

    # Create a new tmux window. Make the window current.  Spawn tig with local
    # history.

    tig_args = ["tig"]
    if no_merges:
        tig_args += ["--no-merges"]
    tig_args += [local_rev, f"^{remote_rev}", "--", *files]

    # -P: print id of new window
    tmux_args = ["tmux", "new-window", "-P", "-n", "tig", "--",
                 wrap_tig_args(tig_args)]
    tmux_win = subprocess.check_output(tmux_args, text=True)
    tmux_win = chomp(tmux_win)

    # Create a new tmux pane in the current window. Spawn tig with remote
    # history.

    tig_args = ["tig"]
    if no_merges:
        tig_args += ["--no-merges"]
    tig_args += [remote_rev]
    if not remote_full_history:
        tig_args += [f"^{local_rev}"]
    tig_args += ["--", *files]

    # -h: horizontal split
    # -d: Do not make current the new pane.
    tmux_args = ["tmux", "split-window", "-d", "-t", tmux_win, "-h",
                 wrap_tig_args(tig_args)]
    subprocess.check_call(tmux_args)

@main.command(
    name = "patch-dir",
    help = dedent(f"""
        Print the patch dir for merges.

        The patch dir is set with the config variable
        {cros_ksm.merge.PATCH_DIR_CONFIG_KEY}. The default value is
        {cros_ksm.merge.PATCH_DIR_DEFAULT!r}.  It is a path relative to the git
        repo's working tree or an absolute path.

        For an explanation of how it is used, see `cros-ksm merge --help`.
    """),
)
@click.pass_context
def cmd_patch_dir(ctx):
    repo = cli_open_repo(ctx)
    print(cros_ksm.merge.get_patch_dir(repo))

@main.command(
    name = "merge",
    help = dedent("""\
        Merge a release tag, including patches.

        This command wraps the git-merge command.

        \b
        Differences with git-merge:
          - Default commit message is customized for ChromeOS stable merges,
            including trailers and subject prefix.
          - Enables --rerere-autoupdate by default.
          - Applies patches found in the cros-ksm patch dir.

        If <tag> is omitted, then option --next-release is required and will
        select the tag.

        This command searches the cros-ksm patch dir for patches to be applied
        during the merge. (See `cros-ksm patch-dir --help`). When merging tag
        "vW.X.Y", it will apply all patches whose filename matches the glob
        "W.X.Y-*.patch". The patches are applied after rerere replays any
        resolutions and before the commit object is created.

        [BUG] The -C flag preserves the commit message of its target commit
        but not its author info (name, email, timestamp). This behavior
        differs from the -C flag of git-commit and the -C flag of the builtin
        `merge` command used in git rebase todo lists. When recreating merge
        commits during a rebase, the correct behavior is to preserve author
        info. This bug impacts `cros-ksm rebase`, which internally uses
        `cros-ksm merge`.
    """),
)
@click.argument(
    "tag_name",
    metavar = "[<tag>]",
    required = False,
)
@option_kernel_major_version
@option_rerere_autoupdate
@option_edit_message
@click.option(
    "choose_next_release",
    "-N", "--next-release",
    is_flag = True,
    help = dedent("""\
        Merge the next release tag. For example, if the most recently merged tag
        is v5.4.100, then merge v5.4.101.
    """),
)
@click.option(
    "message_from_file",
    "-F",
    metavar = "<file>",
    type = click.Path(
        exists = True,
        file_okay = True,
        dir_okay = False,
        readable = True,
        allow_dash = True,
    ),
    help = dedent("""\
        Like same flag of command git-commit. Use the file's content as the
        commit message. Use - to read from standard input.
    """),
)
@click.option(
    "message_from_commit",
    "-C",
    metavar = "<commit>",
    help = dedent("""\
        Like the same flag of command git-commit. When creating the merge
        commit, reuse the message from an existing commit.
    """),
)
@click.option(
    "make_commit",
    "--commit / --no-commit",
    default = True,
    help = "Like the same flag of command git-merge.",
)
@click.pass_context
def cmd_merge(
    ctx: click.Context,
    /, *,
    tag_name: str | None,
    choose_next_release: bool,
    make_commit: bool,
    edit_message: bool,
    message_from_file: str | None,
    message_from_commit: str | None,
    rerere_autoupdate: bool,
):
    repo = cli_open_repo(ctx)

    # Choose the revision to be merged
    if choose_next_release:
        if tag_name is not None:
            ctx.fail("argument <tag> and flag --next-release are mutually exclusive")
        kver: KernelVersion = repo.get_last_merged_release("HEAD")
        tag_name = f"v{kver.w}.{kver.x}.{kver.y + 1}"
        eprint(f"merge next release tag {tag_name}")
    elif tag_name is None:
        ctx.fail("either <tag> or --next-release is required")
    else:
        # Proceed with the user-provided tag name. Validation of the tag is
        # deferred to the the `merge.Driver`.
        pass

    if (message_from_file is not None and
        message_from_commit is not None
    ):
        ctx.fail("flags -F and -C are mutually exclusive")
        assert_unreachable()
    elif message_from_file is not None:
        with click.open_file(message_from_file) as f:
            message_text = f.read()
    elif message_from_commit is not None:
        commit = repo.resolve_refish_to_commit(message_from_commit)
        message_text = str(commit.message)
    else:
        message_text = None

    cros_ksm.merge.Driver(
        repo, tag_name,
        message = message_text,
        edit_message = edit_message,
        make_commit = make_commit,
        rerere_autoupdate = rerere_autoupdate,
    ).run()

@main.command(
    name = "cherry-pick",
    help = dedent("""\
        Cherry pick a commit.

        This command wraps the git-cherry-pick command. The default flags that
        it passes to git-cherry-pick and the default commit message that it
        generates, they are customized for ChromeOS stable merges.
    """),
)
@option_kernel_major_version
@option_edit_message
@option_rerere_autoupdate
@click.option(
    "-o", "--origin",
    metavar = "<origin>",
    help = dedent("""\
        Specify the origin of the cherry picked commit. This changes the
        generated commit message. If <origin> is omitted then an attempt is made
        to infer it.

        \b
        Valid choices are:
            "stable"
                The <commit> comes from this repo's stable branch.
            "upstream"
                The <commit> comes from the upstream mainline branch.
            "linux-{w}.{x}.y"
                The name of any stable branch, such as "linux-6.6.y".
            "other"
                Omit from the generated commit message any information that is
                specific to the commit's origin.
    """),
)
@click.option(
    "add_cherry_pick_line",
    "-x / -r",
    default = True,
    help = "Same as options -x and -r of git-cherry-pick. Default is -x.",
)
@click.argument(
    "rev", # use 'rev' here to avoid a variable name collision later
    metavar = "[<commit>]",
)
@click.pass_context
# ruff: ignore[too-many-branches]
# ruff: ignore[too-many-statements]
def cmd_cherry_pick(
    ctx: click.Context,
    /, *,
    rev: str,
    add_cherry_pick_line: bool,
    edit_message: bool,
    rerere_autoupdate: bool,
    origin: str | None,
):
    repo = cli_open_repo(ctx)
    commit = repo.resolve_refish_to_commit(rev)
    merge_msg_file = repo.git_path("MERGE_MSG")
    cherry_pick_head_file = repo.git_path("CHERRY_PICK_HEAD")

    def match_linux_w_x_y(branch: str) -> bool:
        m = re.match(r"^linux-\d+\.\d+\.y$", branch)
        return m is not None

    # Test if the cherry pick commit is in a revision range
    def test_range(start_rev, stop_rev, *, required=False):
        range_desc = f"revision range {start_rev}..{stop_rev}"
        log.debug("cherry pick: test if commit %s is in %s",
                  commit.id, range_desc)

        start_commit = repo.resolve_refish_to_commit(start_rev)
        stop_commit = repo.resolve_refish_to_commit(stop_rev)

        if repo.in_revision_range(
                commit.id,
                include = [stop_commit.id],
                exclude = [start_commit.id]):
            return True

        if required:
            die(f"failed to find commit {rev} in {range_desc}")

        return False

    # Guess the origin if it was omitted
    if origin is None:
        log.debug("cherry pick: guess origin")

        if test_range("HEAD", repo.stable_branch):
            origin = "stable"
        elif test_range("HEAD", repo.upstream_branch):
            origin = "upstream"
        else:
            eprint(f"failed to find origin branch of commit {rev}")
            eprint("hint: try option --origin")
            die()

    # Rewrite origin 'stable' so that all stable branch origins match the
    # same pattern.
    if origin == "stable":
        origin = os.path.basename(repo.stable_branch)
        assert match_linux_w_x_y(origin)

    # Here we could validate that the cherry pick commit is contained in the
    # user-specified origin branch. But instead we trust the user and skip the
    # validation.

    # Choose the prefix for the commit subject
    if origin == "upstream":
        subject_prefix = "UPSTREAM: "
    elif match_linux_w_x_y(origin):
        subject_prefix = "FROMGIT: "
    else:
        subject_prefix = ""

    change_id = make_gerrit_change_id()
    bug = repo.get_cros_ksm_bug()
    committer = repo.get_committer()

    # Create the commit message
    message = StringIO()

    if not commit.subject.text.startswith(subject_prefix):
        message.write(subject_prefix)

    message.write(chomp(commit.message.text))

    if add_cherry_pick_line:
        message.write(f"\n(cherry picked from commit {commit.id}")

        if match_linux_w_x_y(origin):
            message.write(f"\n {STABLE_REMOTE_CANON_URL}")
            message.write(f"\n {origin}")

        message.write(")\n")

    message.write(dedent(f"""\

        BUG=b:{bug}
        TEST=cq

        Change-Id: {change_id}
        Disallow-Recycled-Builds: test-failures
        Signed-off-by: {committer}
        """))

    message = message.getvalue()

    # Prepare the environment for the custom GIT_EDITOR
    env = os.environ.copy()
    cros_ksm.wrap_git_editor.configure_caller_env(
        env, repo,
        mode = "cherry-pick",
        no_edit = not edit_message,
        override_text = message,
    )

    # Build the git-cherry-pick command
    cmd = [
        "git", "cherry-pick",

        # Always invoke our editor wrapper. Without it, we cannot insert the
        # custom commit message.
        "--edit",
    ]
    args_add_if(cmd, rerere_autoupdate,
                "--rerere-autoupdate", "--no-rerere-autoupdate")
    cmd += ["--", rev]

    # Do not call git-cherry-pick if file CHERRY_PICK_HEAD exists. If the file
    # exists now and git-cherry-pick fails, then the file's existence disrupts
    # the heuristic we use to assess the post-failure state of the repo.
    #
    # Anyway, the file should exist only if a cherry pick is in progress
    # (according to the git-prompt bash completion code). And if a cherry pick
    # is in progress, we should avoid starting a new cherry pick.
    if cherry_pick_head_file.exists():
        eprint(f"file already exists: {cherry_pick_head_file}")
        eprint("fail: is a cherry pick already in progress?")
        die()

    status = subprocess.call(cmd, env=env)
    if status == 0:
        return

    if cherry_pick_head_file.exists():
        # git-cherry-pick failed but left the repo in an in-progress state that
        # is ready for `git cherry-pick --continue`.
        #
        # We must handle the following failure cases:
        #
        #   1. git-cherry-pick failed before invoking the editor. This is the
        #      most common case. It occurs, for example, when the cherry pick
        #      encounters file conflicts.
        #
        #   2. git-cherry-pick invoked the editor and the editor exited with
        #      a failure status. For all builtin git commands that invoke an
        #      editor, if the editor fails then git will discard any text that
        #      the editor wrote to disk and and instead use the default text
        #      that was originally provided to the editor.
        #
        # In each failure case, git-cherry-pick will write the default commit
        # message to file MERGE_MSG before exiting.  Future git commands that
        # create the commit (such as `git cherry-pick --continue` or `git
        # commit`) will retrieve the commit message from MERGE_MSG, clean it,
        # write it to COMMIT_EDITMSG, and then invoke GIT_EDITOR on
        # COMMIT_EDITMSG.
        #
        # Therefore, our custom cherry-pick command must rewrite MERGE_MSG here
        # in order to override the default commit message.
        merge_msg_file.write_text(message)

    sys.exit(status)

@main.command(
    name = "rebase",
    context_settings = {
        "ignore_unknown_options": True,
        "allow_extra_args" : True,
    },
    help = dedent("""\
        Rebase a stable merge series.

        This command takes the same arguments as the git-rebase command. The
        command calls git-rebase, but rewrites the list of commands in the
        rebase's todo list before the rebase begins. The rewritten todo list is
        tailored for ChromeOS stable merges.

        This command should only be used to rebase a work-in-progress stable
        merge branch. Do not use it for arbitrary rebases. The command will fail
        when rewriting the todo list if it encounters unexpected patterns.
    """),
)
@option_kernel_major_version
@click.option(
    "-i", "--interactive",
    is_flag = True,
    hidden = True,
)
@click.pass_context
def cmd_rebase(
    ctx: click.Context,
    /, *,
    interactive: bool,
):
    repo = cli_open_repo(ctx)

    # Prepare the environment for the custom GIT_SEQUENCE_EDITOR
    env = os.environ.copy()
    cros_ksm.wrap_git_editor.configure_caller_env(
        env, repo,
        mode = "rebase",
        no_edit = not interactive,
    )

    cmd = [
        "git", "rebase",

        # Always use --interactive because it invokes our editor wrapper.
        # Without the wrapper, we cannot insert the custom todo list.
        "--interactive",

        "--rebase-merges",
        *ctx.args,
    ]

    sys.exit(subprocess.call(cmd, env=env))

@main.command(
    name = "wrap-git-editor",
    hidden = True,
    help = dedent("""\
        (internal) Wrap a git editor.

        Use this command as a temporary GIT_EDITOR that overwrites the file with
        custom text and then dispatches to the user's real GIT_EDITOR. Can also
        be used for GIT_SEQUENCE_EDITOR.
    """),
)
@click.argument(
    "path",
    required = True,
    type = click.Path(path_type=Path),
)
@click.pass_context
def cmd_wrap_git_editor(
    ctx: click.Context,
    path: Path,
):
    repo = cli_open_repo(ctx)
    status = cros_ksm.wrap_git_editor.edit(repo, path)
    sys.exit(status)

@main.command(
    name = "find-cherry-picks",
    help = dedent("""\
        Find cherry picks needed for a stable merge.

        In typical usage, when merging a stable release into a downstream
        branch, this command searches for commits in future stable releases that
        fix any of the merge's incoming commits.

        More precisely, the command searches the revision range
        `INCOMING_REV..FUTURE_REV` for any commits that revert or fix a commit
        in range `DOWNSTREAM_REV..INCOMING_REV`, and also searches for any
        dependencies of the found commits.

        The cherry pick commits are listed in the recommended order for cherry
        picking. Each commit is listed before its descendants.
    """),
)
@option_kernel_major_version
@click.option(
    "-i", "--incoming-rev",
    metavar = "INCOMING_REV",
    help = "Default is HEAD.",
)
@click.option(
    "-d", "--downstream-rev",
    help = dedent("""\
        Default is the 'cros' branch for the repo's kernel major version, such
        as 'cros/chromeos-6.12'.
        """),
)
@click.option(
    "-f", "--future-rev",
    help = dedent("""\
        Default is the 'stable' branch for the repo's kernel major version, such
        as 'stable/linux-6.12.y'.
        """),
)
@click.option(
    "-F", "--format",
    default = "flat",
    type = click.Choice(["flat", "id", "json"]),
    help = dedent("""\
        \b
        Choices:
            flat [default]
                Similar to `git log --oneline`. Each line is "id subject".
            id
                Each line is only the commit id.
            json
                The format everyone loves to hate.
        """),
)
@click.pass_context
# ruff: ignore[too-many-branches]
def cmd_find_cherry_picks(
    ctx: click.Context,
    /, *,
    incoming_rev: str | None,
    downstream_rev: str | None,
    future_rev: str | None,
    format: str,
):
    repo = cli_open_repo(ctx)

    if incoming_rev is None:
        incoming_rev = "HEAD"

    if downstream_rev is None:
        downstream_rev = repo.cros_branch

    if future_rev is None:
        future_rev = repo.stable_branch

    log.debug("incoming-rev: %r", incoming_rev)
    log.debug("downstream-rev: %r", downstream_rev)
    log.debug("future-rev: %r", future_rev)

    incoming_commit = repo.try_resolve_refish_to_commit(incoming_rev)
    if incoming_commit is None:
        die(f"failed to resolve incoming rev {incoming_rev!r} to a commit")

    downstream_commit = repo.try_resolve_refish_to_commit(downstream_rev)
    if downstream_commit is None:
        die(f"failed to resolve downstream rev {downstream_rev!r} to a commit")

    future_commit = repo.try_resolve_refish_to_commit(future_rev)
    if future_commit is None:
        die(f"failed to resolve future rev {future_rev!r} to a commit")

    cherry_commits = find_cherry_picks(
        repo,
        incoming_commit = incoming_commit,
        downstream_commit = downstream_commit,
        future_commit = future_commit,
    )

    match format:
        case "flat":
            for commit in cherry_commits:
                print(f"{commit.id} {commit.subject}")
        case "id":
            for commit in cherry_commits:
                print(commit.id)
        case "json":
            j = {
                "format_version": "1",
                "commits": [],
            }
            for commit in cherry_commits:
                j["commits"].append({
                    "id": str(commit.id),
                    "subject": commit.subject.text,
                    "committer-date-utc": format_utc(commit.committer_date),
                })
            json.dump(j, sys.stdout, indent=4)
            sys.stdout.write("\n")
        case _:
            assert False

@main.command(
    name = "status-unmerged",
    help = dedent("""\
        Show status of unmerged files.

        This command behaves similar to `git status --porcelain=v1`
        but only shows unmerged entries.
    """),
)
@click.argument(
    "pathspec",
    metavar = "[<pathspec>...]",
    nargs = -1,
)
@click.option(
    "-n", "--name-only",
    is_flag = True,
    help = dedent("""\
        Show only the file name of each file.
        Omit the status flags.
    """),
)
@click.option(
    "opt_z", "-z",
    is_flag = True,
    help = "Terminate entries with NUL instead of LF."
)
# ruff: ignore[too-many-branches]
def cmd_status_unmerged(
    *,
    pathspec: list[str],
    name_only: bool,
    opt_z: bool,
):
    git_args = ["git", "status", "-z", "--porcelain=v2", "--", *pathspec]
    git_out = subprocess.check_output(git_args, text=False)
    chunks = iter(git_out.split(b"\0"))

    def die_bad_chunk(chunk: bytes):
        die(f"error: bad git output: {chunk!r}\n")

    while True:
        chunk = next(chunks, None)
        if chunk is None:
            break
        elif len(chunk) == 0:
            continue

        match chunk[0:1]:
            case b"1":
                # Ordinary entry.
                pass
            case b"2":
                # Renamed or copied file. Next chunk is the destination
                # path. Discard it.
                next(chunks, None)
            case b"u":
                # Unmerged entry. See <man:git-status(1)> for explanation of
                # fields.
                try:
                    _, xy, _sub, _m1, _m2, _m3, _mw, _h1, _h2, _h3, path = chunk.split(b" ")
                except ValueError:
                    die_bad_chunk(chunk)

                if not name_only:
                    sys.stdout.buffer.write(xy)
                    sys.stdout.buffer.write(b" ")

                sys.stdout.buffer.write(path)

                if opt_z:
                    sys.stdout.buffer.write(b"\0")
                else:
                    sys.stdout.buffer.write(b"\n")
            case b"?":
                pass
            case b"#":
                pass
            case _:
                die_bad_chunk(chunk)

@main.command(
    name = "press-yes-no",
    hidden = True,
    help = dedent("""\
        Wait until 'y' or 'n' is pressed.

        Exit with 0 if 'y' is pressed. Exit with 1 if 'n'.

        This command is intended for usage in user shell scripts.
    """),
)
@click.option(
    "-q", "--quiet",
    is_flag = True,
)
def cmd_press_yes_no(*, quiet: bool):
    if not quiet:
        eprint("press 'y' to continue", flush=True)
        eprint("press 'n' to fail", flush=True)

    while True:
        match click.getchar():
            case "y" | "Y":
                sys.exit(0)
            case "n" | "N":
                sys.exit(1)

# internal helper command
@main.command(
    name = "press-q",
    hidden = True,
    help = "Wait until 'q' is pressed",
)
@click.option(
    "-q", "--quiet",
    is_flag = True,
)
def cmd_press_q(*, quiet: bool):
    if not quiet:
        eprint("press 'q' to exit", flush=True)

    while True:
        match click.getchar():
            case "q" | "Q":
                sys.exit(0)

main()
