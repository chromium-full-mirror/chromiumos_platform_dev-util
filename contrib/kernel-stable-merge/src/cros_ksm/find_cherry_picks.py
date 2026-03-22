# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Implementation pieces for command cros-ksm-find-cherry-picks."""

from cros_ksm.disjoint_set import Node
from cros_ksm.git import Commit, HexOid, Repo, Trailer
from cros_ksm.util import log

# This dictionary is the data structure where we incrementally build equivalence
# classes of commit ids.  We consider two oids to be equivalent if (a) one is an
# abbreviation of the other, or (b) one is a cherry-pick or backport of the
# other.
#
# The dictionary is a one-to-one mapping between HexOid and Node. Every time we
# create a node, we add the node and its oid to this dict.
#
# When we create a new node, it begins its life as a singleton set. As we dig
# through the commits in the git repo and discover new equivalences, we join the
# sets together.
type HexOidNodes = dict[HexOid, Node[HexOid]]

def _warn_resolve_ported_oid(
    oid: HexOid,
    src_commit: Commit,
):
    log.warning("failed to resolve id to commit: "
                "ported id %s found in message of commit %s",
                oid, src_commit.id)

def _warn_resolve_reverted_oid(
    oid: HexOid,
    src_commit: Commit,
):
    log.warning("failed to resolve id to commit: "
                "reverted id %s found in message of commit %s",
                oid, src_commit.id)

def _warn_resolve_trailer_oid(
    oid: HexOid,
    src_commit: Commit,
    src_trailer: Trailer,
):
    log.warning("failed to resolve id to commit: "
                "id %s in trailer %r of commit %s",
                oid, src_trailer.text, src_commit.id)

def _warn_parse_trailer_oid(
    commit: Commit,
    trailer: Trailer,
):
    log.warning("failed to parse commit id from trailer %r of commit %s",
                trailer.text, commit.id)

def _collect_commits(repo: Repo, start: Commit, stop: Commit) -> list[Commit]:
    """Collect commits in revision range `start..stop`."""
    walk = repo.pygit2.walk(stop.id)
    walk.hide(start.id)
    return [
        repo.register_pygit2_commit(commit)
        for commit in walk
    ]

def _get_node(nodes: HexOidNodes, oid: HexOid) -> Node[HexOid]:
    """Return the node for oid. If it does not exist yet, then create it."""
    node = nodes.get(oid, None)
    if node is None:
        node = Node(oid)
        nodes[oid] = node
    return node

def _dig_commit(
    repo: Repo,
    nodes: HexOidNodes,
    top_commit: Commit,
    top_commit_short_oid: HexOid | None = None,
    /,
) -> Node[HexOid]:
    """Grow the commit's equivalency set by searching its commit message for
    equivalent ported oids, and recursing into the found ported commits.

    Return a node in the commit's equivalency set.

    If the caller discovered the commit by resolving a short oid (for example,
    an oid found in a Fixes trailer), the short oid should be passed to this
    function to ensure that it is added to the commit's equivalency set.
    """

    log.debug("commit %s: dig: start", top_commit.id)

    # If we have already dug into this commit, then skip digging into its commit
    # message. We must do this check _before_ creating the commit's node.
    skip_message = top_commit.hex_oid in nodes

    top_node = _get_node(nodes, top_commit.hex_oid)

    if top_commit_short_oid is not None:
        # Add short oid to the top commit's equivalency set
        log.debug("commit %s: dig: short id: %s",
                  top_commit.id, top_commit_short_oid)
        short_node = _get_node(nodes, top_commit_short_oid)
        top_node.merge(short_node)

    if not skip_message:
        # Process any ported commit ids found in the top commit message
        for port_oid in top_commit.message.ported_oids:
            log.debug("commit %s: found ported id %s", top_commit.id, port_oid)

            # Add the ported commit id to the top commit's equivalency set
            port_node = _get_node(nodes, port_oid)
            top_node.merge(port_node)

            # If the ported commit id resolves to a commit,
            # then recursively dig into its commit message.
            port_commit = repo.try_resolve_refish_to_commit(port_oid)
            if port_commit is None:
                _warn_resolve_ported_oid(port_oid, top_commit)
                continue
            top_node.merge(_dig_commit(repo, nodes, port_commit))

    log.debug("commit %s: dig: done", top_commit.id)
    return top_node

def _build_equivalency_sets(
    repo: Repo,
    incoming_commits: list[Commit],
    future_commits: list [Commit],
) -> HexOidNodes:
    """Build equivalency sets of commit ids of the given commits, and of any
    relevant commits discovered by recursively digging into their commit
    messages.
    """

    nodes: HexOidNodes = {}

    # We process incoming and future commits differently.  For each future
    # commit, we must also process any fixed or reverted commits found in its
    # commit message. For incoming commits, this is unnecessary and does not
    # change the final list of chosen cherry picks.

    for incoming_commit in incoming_commits:
        _dig_commit(repo, nodes, incoming_commit)

    for future_commit in future_commits:
        _dig_commit(repo, nodes, future_commit)

        # Dig into any oids that the commit reverts.
        for reverted_oid in future_commit.message.reverted_oids:
            log.debug("commit %s: found reverted id %s",
                      future_commit.id, reverted_oid)

            reverted_commit = repo.try_resolve_refish_to_commit(reverted_oid)
            if reverted_commit is None:
                _warn_resolve_reverted_oid(reverted_oid, future_commit)
                continue

            _dig_commit(repo, nodes, reverted_commit, reverted_oid)

        # Dig into any oids that the commit fixes or is a stable dep of.
        for trailer in future_commit.message.trailers.values():
            if trailer.oid is None:
                _warn_parse_trailer_oid(future_commit, trailer)
                continue

            log.debug("commit %s: trailer %r: parsed id %s",
                      future_commit.id, trailer.text, trailer.oid)

            trailer_commit = repo.try_resolve_refish_to_commit(trailer.oid)
            if trailer_commit is None:
                _warn_resolve_trailer_oid(trailer.oid, future_commit, trailer)
                continue

            _dig_commit(repo, nodes, trailer_commit, trailer.oid)

    return nodes

def _choose_cherry_picks(
    repo: Repo,
    incoming_commits: list[Commit],
    future_commits: list [Commit],
    nodes: HexOidNodes,
) -> list[Commit]:
    """Choose the future commits that should be cherry picked.

    Candidates for cherry picking are the future commits that fix or revert an
    incoming commit, and also the dependencies of any chosen cherry picks.
    """

    # Commits that need cherry picking
    cherry_commits: list[Commit] = []

    # Set of all oids (full and abbreviated) of all commits in `cherry_commits`,
    # as well as any known ports of those commits.
    #
    # We use this set to avoid adding duplicates to `cherry_commits`.
    #
    # A realworld corner-case that must be avoided is when the future commits on
    # the stable branch contain a cherry pick candidate, it is later reverted on
    # the stable branch, and then the stable branch cherry picks it again. In
    # this case, we must add it to `cherry_commits` only once not twice.
    cherry_oid_ports: set[HexOid] = set()

    # Set of all oids (full and abbreviated) of all commits in
    # `incoming_commits`, as well as any known ports of those commits.
    incoming_oid_ports: set[HexOid] = {
        node.data
        for commit in incoming_commits
        for node in nodes[commit.hex_oid].iter_nodes()
    }

    def want_cherry_pick(commit: Commit) -> bool:
        if commit.hex_oid in cherry_oid_ports:
            # Commit was already chosen earlier
            return False

        target_oids: set[HexOid] = commit.message.reverted_oids.copy()
        target_oids.update(
            trailer.oid
            for trailer in commit.message.trailers.values()
            if trailer.oid is not None
        )

        for oid in target_oids:
            if oid in incoming_oid_ports:
                # The commit targets an incoming commit
                return True

            if oid in cherry_oid_ports:
                # The commit targets a previously chosen cherry pick
                return True

        return False

    # Collect the cherry picks with repeated passes until the collected set
    # becomes stable. This is necessary because among the future commits (a)
    # a cherry pick candidate might itself be buggy and be fixed by a later
    # commit, which transitively also becomes a cherry pick candidate, and (b)
    # stable deps may have been applied out of order on the stable branch.
    while True:
        old_len = len(cherry_commits)

        for commit in future_commits:
            if not want_cherry_pick(commit):
                continue
            log.debug("found cherry pick %s", commit.id)
            cherry_commits.append(commit)
            cherry_oid_ports.update(
                node.data for node in nodes[commit.hex_oid].iter_nodes()
            )

        if len(cherry_commits) == old_len:
            # The search has become stable. No new cherry picks were found.
            break

    return cherry_commits

def _sorted_commits(
    repo: Repo,
    head: Commit,
    commits: list[Commit],
) -> list[Commit]:
    """Return a new list of commits where each commit appears before its
    descendants.

    If the commit history is linear, then the returned order is chronological by
    committer date, oldest to newest.

    All commits must be in the history of the `head` commit. Duplicate commits are removed.
    """

    new_list: list[Commit] = []

    oids = {
        commit.id: commit
        for commit in commits
    }

    for pygit2_commit in repo.pygit2.walk(head.id):
        if not oids:
            break
        oid = pygit2_commit.id
        commit = oids.get(oid)
        if commit is None:
            continue
        new_list.append(commit)
        del oids[oid]

    if oids:
        raise ValueError("head does not contain history of all commits")

    new_list.reverse()
    return new_list

def find_cherry_picks(
    repo: Repo,
    *,
    incoming_commit: Commit,
    downstream_commit: Commit,
    future_commit: Commit,
) -> list[Commit]:
    log.debug("find cherry picks: start")
    log.debug("find cherry picks: incoming commit %s", incoming_commit.id)
    log.debug("find cherry picks: downstream commit %s", downstream_commit.id)
    log.debug("find cherry picks: future commit %s", future_commit.id)

    incoming_commits = _collect_commits(repo, downstream_commit, incoming_commit)
    future_commits = _collect_commits(repo, incoming_commit, future_commit)
    nodes = _build_equivalency_sets(repo, incoming_commits, future_commits)
    cherry_commits = _choose_cherry_picks(repo, incoming_commits, future_commits, nodes)
    return _sorted_commits(repo, future_commit, cherry_commits)
