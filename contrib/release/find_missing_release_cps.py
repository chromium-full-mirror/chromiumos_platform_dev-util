#!/usr/bin/env python3
# Copyright 2025 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""A small script to report possible missing cherry picks on release branches.

Identifies possible missing cherry picks using Gerrit CLs reported by
go/gitwatcher. That is, any CL subjects that do not appear on older milestones
(down to the minimum specified milestone, or down to the smallest milestone for
which the bug has a CL).

The user provides the milestone associated with main using --tot-milestone.

Sample usage:
    ./find_missing_release_cps.py --bug-id 987654321 --tot-milestone 136
"""

import argparse
import dataclasses
import functools
import json
import logging
import re
import subprocess
import sys


# Requirements
# - go/gcert
# - go/bugged#installation

# Relies on the following values:
GIT_WATCHER_USERS = (
    "apphosting-stubby-api--s-7egoogle-2ecom-3agitwatcher@prod.google.com",
    "dx-workflow-gitwatcher@prod.google.com",
)
CL_PATTERN = r"(https://.*review.*/[0-9]{7})"
MSTONE_BRANCH_PATTERN = r"release-R([0-9]+)-[0-9]+.B"

# ANSI escape codes used for output formatting.
GREEN = "\033[92m"
RED = "\033[31m"
CYAN = "\033[36m"
RESET = "\033[0m"


@dataclasses.dataclass
class ClInfo:
    """Details about a CL including branch, subject, milestone, etc."""

    # Gerrit Change number.
    change_number: str
    # Full URI.
    uri: str
    # Gerrit subject.
    subject: str
    # Gerrit branch.
    branch: str
    # Gerrit project.
    project: str
    # Associated milestone.
    mstone: int


@functools.lru_cache
def branch_to_mstone(branch, tot_mstone):
    if branch == "main":
        return int(tot_mstone)
    mstone_match = re.match(MSTONE_BRANCH_PATTERN, branch)
    if not mstone_match:
        logging.info(
            "Skipping non-release branch: %s. Expected branch to match pattern:"
            " '%s'.",
            branch,
            MSTONE_BRANCH_PATTERN,
        )
        return -1
    return int(mstone_match.group(1))


def set_up_args():
    argparser = argparse.ArgumentParser(
        description="Report Gerrit CLs and their branches for a given bug."
    )
    argparser.add_argument("--bug-id", help="ID of bug, e.g. 987654321")
    argparser.add_argument("--tot-milestone", help="Milestone on ToT, e.g. 136")
    argparser.add_argument(
        "--min-milestone",
        help="(Optional) Minimum milestone to look for cherry-picks, e.g. 134",
    )
    return argparser.parse_args(sys.argv[1:])


def check_required_args(args):
    if not args.bug_id:
        sys.exit("Exiting. Must provide a bug id.")

    if not args.tot_milestone:
        sys.exit(
            "Exiting. Must provide the current ToT milestone to identify "
            "possible missing cherry-picks."
        )


def check_gcert():
    cmd = ["gcertstatus"]
    result = subprocess.run(
        cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    if result.returncode:
        logging.debug(result.stderr)
        sys.exit('Exiting. Must run "gcert" first.')


def find_bug_comments(bug_id: str):
    cmd = [
        "bugged",
        "show",
        bug_id,
    ]
    result = subprocess.run(
        cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    if result.returncode:
        sys.exit(
            f"Exiting. Failed to find b/{bug_id}: {result.stderr}. Have you "
            f"checked b/{bug_id} exists and followed go/bugged#installation?"
        )
    return [
        comment
        for comment in result.stdout.decode("utf-8").split("From: ")
        if comment
    ]


def find_cls_from_gitwatcher_comments(comments, bug_id, gitwatcher_users):
    git_watcher_comments = []
    for comment in comments:
        if comment.startswith(gitwatcher_users):
            git_watcher_comments.append(comment)
    cls = set()
    for gw_comment in git_watcher_comments:
        # May also pull in CLs reference from a commit message.
        matches = re.findall(CL_PATTERN, gw_comment)
        for match in matches:
            cls.add(match)
    if not cls:
        sys.exit(f"Exiting. No CLs found for b/{bug_id}.")
    return cls


def find_cl_details(cls, bug_id, tot_milestone):
    summary_by_mstone = {}
    change_numbers_seen = set()
    for cl in sorted(cls):
        change_number = cl.split("/")[-1]
        # Dedupe URL variants.
        if change_number in change_numbers_seen:
            continue
        change_numbers_seen.add(change_number)
        cl_base = cl.replace(change_number, "")
        # Remove project atoms, if present.
        cl_base = cl_base.split("/c/")[0]
        change_endpoint = cl_base + "a/changes/" + change_number
        cmd = ["gob-curl", change_endpoint]
        result = subprocess.run(
            cmd,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
        )
        if result.returncode:
            logging.debug(result.stderr)
            cmd_str = " ".join(cmd)
            sys.exit(
                f'Exiting. Failed to fetch info for cl: "{cl}" using cmd: '
                f'"{cmd_str}".'
            )
        result_stdout = result.stdout.decode("utf-8")
        # Workaround to exclude first line non-JSON: )]}'
        bad_str = ")]}'\n"
        if result_stdout.startswith(bad_str):
            result_stdout = result_stdout.splitlines()[1]
        try:
            cl_details = json.loads(result_stdout)
        except json.JSONDecodeError:
            cmd_str = " ".join(cmd)
            sys.exit(
                f'Exiting. Failed to find cl details for cmd "{cmd_str}":\n'
                f'{result.stdout.decode("utf-8")}'
            )
        mstone = branch_to_mstone(cl_details["branch"], tot_milestone)
        if mstone == -1:
            continue
        if mstone not in summary_by_mstone:
            summary_by_mstone[mstone] = []
        cl_info = ClInfo(
            branch=cl_details["branch"],
            subject=cl_details["subject"],
            project=cl_details["project"],
            uri=cl,
            change_number=change_number,
            mstone=mstone,
        )
        summary_by_mstone[mstone].append(cl_info)

    if not summary_by_mstone:
        sys.exit(
            f"Exiting. Could not find Gerrit details for any CLs on b/{bug_id}."
        )
    return summary_by_mstone


def identify_possible_missing_cps(
    summary_by_mstone, bug_id, tot_milestone, min_milestone
):
    mstones_descending = list(summary_by_mstone.keys())
    mstones_descending.sort(reverse=True)

    min_mstone = min_milestone or min(mstones_descending)
    min_mstone = int(min_mstone)
    logging.info(
        "Checking for missing cherry picks on older milestones between "
        "M%s and M%s.",
        min_mstone,
        tot_milestone,
    )
    if min_milestone:
        logging.info(
            "\t(Using specified minimum milestone: M%s)", min_milestone
        )
    else:
        logging.info(
            "\t(Using oldest seen milestone from CLs on b/%s: M%s)",
            bug_id,
            min_mstone,
        )

    missing_cps = {}
    for mstone in mstones_descending:
        logging.debug(
            "Found %s cls for milestone %s.",
            len(summary_by_mstone[mstone]),
            mstone,
        )
        # Ensure CL exists on older milestones.
        for mstone_to_check in range(min_mstone, mstone):
            if mstone_to_check not in summary_by_mstone:
                missing_cps[mstone_to_check] = summary_by_mstone[mstone]
                continue
            for cl in summary_by_mstone[mstone]:
                if cl.subject not in [
                    cl.subject for cl in summary_by_mstone[mstone_to_check]
                ]:
                    if mstone_to_check not in missing_cps:
                        missing_cps[mstone_to_check] = []
                    missing_cps[mstone_to_check].append(cl)
    return missing_cps


def report_merged_cls(summary_by_mstone, bug_id):
    logging.info(
        "\nFor b/%s the following changes were landed on release branches:",
        bug_id,
    )
    for mstone in sorted(summary_by_mstone.keys(), reverse=True):
        cls = summary_by_mstone[mstone]
        logging.info(
            "%s%s (M%s)%s - %s CL(s)",
            CYAN,
            cls[0].branch,
            mstone,
            RESET,
            len(cls),
        )
        for cl in cls:
            logging.info("\t '%s' in '%s' - %s", cl.subject, cl.project, cl.uri)


def report_missing_cps(missing_cps):
    if not missing_cps:
        logging.info("\n%sNo missing cherry picks detected!%s", GREEN, RESET)
    else:
        logging.info("\n%sDetected missing cherry picks:%s", RED, RESET)
        for mstone in sorted(missing_cps.keys(), reverse=True):
            logging.info("M%s", mstone)
            for missing_cp in missing_cps[mstone]:
                logging.info(
                    "\t%s - seen on M%s (%s)",
                    missing_cp.subject,
                    missing_cp.mstone,
                    missing_cp.uri,
                )
        logging.info(
            "%sMay be waiting for cherry picks to be created or merged!%s",
            RED,
            RESET,
        )


def main():
    args = set_up_args()
    check_required_args(args)

    # Must run gcert.
    check_gcert()

    # Find comments on the given bug.
    bug_comments = find_bug_comments(args.bug_id)

    # Extract CLs from go/gitwatcher comments.
    cls = find_cls_from_gitwatcher_comments(
        bug_comments, args.bug_id, GIT_WATCHER_USERS
    )

    # Find Gerrit information for each CL (branch, subject, etc).
    summary_by_mstone = find_cl_details(cls, args.bug_id, args.tot_milestone)

    # Identify possible missing cherry picks to older milestones.
    missing_cps = identify_possible_missing_cps(
        summary_by_mstone, args.bug_id, args.tot_milestone, args.min_milestone
    )

    # Report CLs for bug.
    report_merged_cls(summary_by_mstone, args.bug_id)

    # Report on missing cherry picks.
    report_missing_cps(missing_cps)


if __name__ == "__main__":
    logging.basicConfig(stream=sys.stdout, level=logging.INFO)
    main()
