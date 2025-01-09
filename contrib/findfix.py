#!/usr/bin/env python3
# -*- coding: utf-8 -*-
#
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Report branches including the provided fix(es)
"""

import argparse
import re
import subprocess
import sys


STABLE_URLS = [
    'git://git.kernel.org/pub/scm/linux/kernel/git/stable/linux-stable.git',
]

UPSTREAM_URLS = [
    'git://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git',
    'https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git',
    'https://kernel.googlesource.com/pub/scm/linux/kernel/git/torvalds/linux.git',
]

STABLE_QUEUE_URLS = [
    'git://git.kernel.org/pub/scm/linux/kernel/git/stable/linux-stable-rc.git'
]

CHROMEOS_URLS = [
    'https://chromium.googlesource.com/chromiumos/third_party/kernel'
]

BRANCHES = ('6.12', '6.6', '6.1', '5.15', '5.10', '5.4', '4.19')

MIN_RELEASE = 120

def _git(args, stdin=None, encoding='utf-8'):
    """Calls a git subcommand.

    Similar to subprocess.check_output.

    Args:
        args: subcommand + args passed to 'git'.
        stdin: a string or bytes (depending on encoding) that will be passed
            to the git subcommand.
        encoding: either 'utf-8' (default) or None. Override it to None if
            you want both stdin and stdout to be raw bytes.

    Returns:
        the stdout of the git subcommand, same type as stdin. The output is
        also run through strip to make sure there's no extra whitespace.

    Raises:
        subprocess.CalledProcessError: when return code is not zero.
            The exception has a .returncode attribute.
    """

    try:
        # print(['git'] + args)
        return subprocess.run(
            ['git'] + args,
            encoding=encoding,
            input=stdin,
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            check=True,
        ).stdout.strip()
    except subprocess.CalledProcessError:
        return None


def release_branches(chromeos_remote, baseline):
    """Return existing release branches."""

    command = ['branch', '-r', '--list', '%s/release-R*-chromeos-%s' % (chromeos_remote, baseline)]

    releases = []

    branches = _git(command).splitlines()

    for branch in branches:
        release = re.match(r' *%s/release-(R([0-9]+)-[0-9]+\.B)-.*' % chromeos_remote, branch)
        if release and int(release[2]) >= MIN_RELEASE:
            releases += [release[1]]

    return releases


def _find_remote(urls):
    """Find a remote pointing to a given repository."""

    # https remotes may end with or without '/'. Cover both variants.
    _urls = []
    for url in urls:
        _urls.append(url)
        if url.startswith('https://'):
            if url.endswith('/'):
                _urls.append(url.rstrip('/'))
            else:
                _urls.append(url + '/')

    for remote in _git(['remote']).splitlines():
        try:
            if _git(['remote', 'get-url', remote]) in _urls:
                return remote
        except subprocess.CalledProcessError:
            # Kinda weird, get-url failing on an item that git just gave us.
            continue
    return None


def integrated(sha):
    """Find tag at which a given SHA was integrated"""
    fixed = _git(['describe', '--match', 'v*', '--contains', sha])
    if fixed:
        fixed = fixed.split('~')[0]
    else:
        # This patch may not be upstream.
        upstream = _find_remote(UPSTREAM_URLS)
        gitcommand = ['merge-base', '--is-ancestor', sha, f'{upstream}/master']
        if _git(gitcommand) is not None:
            fixed = "ToT"
        else:
            fixed = None

    return fixed


def compare_releases(r1, r2):
    """Compare two releases.

    Return 0 if equal, <0 if r1 is older than r2, >0 if r1 is newer than r2
    """

    s1 = 0
    s2 = 0

    if r1 == r2:
        return 0
    if r1 == 'ToT':
        return 1
    if r2 == 'ToT':
        return -1

    m = re.match(r'(?:v)(\d+).(\d+)(?:\.(\d+))?.*', r1)
    if m:
        s1 = int(m.group(1)) * 1000000
        s1 += int(m.group(2)) * 1000
        if m.group(3) is not None:
            s1 += int(m.group(3))

    m = re.match(r'(?:v)?(\d+).(\d+)(?:\.(\d+))?.*', r2)
    if m:
        s2 = int(m.group(1)) * 1000000
        s2 += int(m.group(2)) * 1000
        if m.group(3) is not None:
            s2 += int(m.group(3))

    if s1 == s2:
        return 0
    if s1 < s2:
        return -1
    return 1


def getcommit(sha):
    """Get commit message for given SHA"""

    gitcommand = ['show', sha]
    return _git(gitcommand)


def get_patch_id(sha):
    """Get patch ID for given SHA"""

    commit = getcommit(sha)
    if not commit:
        return None

    gitcommand = ['patch-id']
    spid = _git(gitcommand, stdin=commit)
    if not spid:
        return None
    return spid.split(" ", 1)[0]


def get_fixes(sha):

    commit = getcommit(sha)
    if not commit:
        return None

    return re.findall(r'Fixes: (.*)', commit)


def findpatch(remote, branch, baseline, sha, subject):
    """ Return True if a patch is found in a branch, False otherwise"""

    release_branch = f'release-{branch}-chromeos-{baseline}'
    gitcommand = ['merge-base', '--is-ancestor', sha, f'{remote}/{release_branch}']
    result = _git(gitcommand)
    if result:
        return True

    # SHA is not in release branch. Search for it based on subject and patch ID.

    patch_id = get_patch_id(sha)
    if not patch_id:
        return False

    gitcommand = ['log', '--oneline', f'v{baseline}..{remote}/{release_branch}']
    commits = _git(gitcommand)
    if not commits:
        return False

    for commit in commits.splitlines():
        if subject in commit:
            ssha = commit.split(' ')[0]
            match_id = get_patch_id(ssha)
            if match_id == patch_id:
                return True

    return False


def check_release_branches(baseline, sha, subject):
    """Check if a patch is present in a release branch"""

    chromeos_remote = _find_remote(CHROMEOS_URLS)
    if chromeos_remote:
        contained = []
        not_contained = []
        for b in release_branches(chromeos_remote, baseline):
            release = b.split('-', maxsplit=1)[0]
            if findpatch(chromeos_remote, b, baseline, sha, subject):
                contained += [release]
            else:
                not_contained += [release]
        if not_contained:
            print('    Not in ' + ', '.join(not_contained))
        if contained:
            print('    In ' + ', '.join(contained))


def checkbranch(remote, baseline, subject, queued=False):
    """Check if a commit is present in a branch.

    Check if a commit described by its subject line is present
    in the provided remote and branch (identified by its baseline version
    number)
    """

    gitcommand = ['log', '--oneline', f'v{baseline}..{remote}/linux-{baseline}.y']
    commits = _git(gitcommand)

    # If the patch in not in an upstream stable release, try to find it in
    # a ChromeOS branch.
    if subject not in commits:
        chromeos_remote = _find_remote(CHROMEOS_URLS)
        if chromeos_remote:
            gitcommand = ['log', '--oneline', f'v{baseline}..{chromeos_remote}/chromeos-{baseline}']
            commits = _git(gitcommand)

    if not commits:
        return False

    for commit in commits.splitlines():
        if subject in commit:
            ssha = commit.split(' ')[0]
            isha = integrated(ssha)
            if not isha or isha == 'ToT':
                if queued:
                    print((f'  Expected to be fixed in chromeos-{baseline} '
                           f'with next stable release merge (sha {ssha})'))
                else:
                    print(f'  Fixed in chromeos-{baseline} (sha {ssha})')
                    check_release_branches(baseline, ssha, subject)
            elif not queued:
                print(f'  Fixed in chromeos-{baseline} with merge of {isha} (sha {ssha})')
                check_release_branches(baseline, ssha, subject)
            return True

    return False


def main(args):
    """Main entrypoint.

    Args:
        args: sys.argv[1:]

    Returns:
        An int return code.
    """

    parser = argparse.ArgumentParser()

    parser.add_argument('shas', nargs='+',
                        help='A valid SHA')

    args = vars(parser.parse_args(args))

    remote = _find_remote(STABLE_URLS)
    if remote:
        _git(['fetch', remote])
    else:
        print('Stable remote not found, results may be incomplete')

    queue_remote = _find_remote(STABLE_QUEUE_URLS)
    if queue_remote:
        _git(['fetch', queue_remote])
    else:
        print('Stable queue remote not found, results may be incomplete')

    chromeos_remote = _find_remote(CHROMEOS_URLS)
    if chromeos_remote:
        _git(['fetch', chromeos_remote])
    else:
        print('ChromeOS remote not found, results may be incomplete')

    # Try to find and fetch upstream
    upstream = _find_remote(UPSTREAM_URLS)
    if upstream:
        _git(['fetch', upstream])

    for sha in args['shas']:
        subject = _git(['show', '--pretty=format:%s', '-s', sha])
        if not subject:
            print(f'Subject not found for SHA {sha}')
            sys.exit(1)

        full_sha = _git(['show', '--pretty=format:%H', '-s', sha])
        if not full_sha:
            print(f'Failed to extract full SHA for SHA {sha}')
            sys.exit(1)

        ssha = full_sha[0:11]

        integrated_branch = integrated(ssha)
        if integrated_branch:
            print(f'Upstream commit {ssha} ("{subject}")')
            print(f'  Integrated in {integrated_branch}')
            fixes = get_fixes(ssha)
            for fix in fixes:
                print(f'  Fixes: {fix}')
                isha = fix.split(' ')[0]
                if isha:
                    introduced = integrated(isha)
                    if introduced:
                        print(f'    Introduced in {introduced}')
        else:
            # If the patch is not upstream, do not bother trying to find
            # ChromeOS branches.
            # FIXME: We should try to find non-upstream patches as well.
            print(f'Commit {ssha} ("{subject}")')
            print('  Not found in upstream kernel')
            continue

        for branch in BRANCHES:
            if compare_releases(integrated_branch, branch) > 0:
                if (not checkbranch(remote, branch, subject) and
                    (not queue_remote
                     or not checkbranch(queue_remote, branch, subject, queued=True))):
                    print(f'  Not in chromeos-{branch}')


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
