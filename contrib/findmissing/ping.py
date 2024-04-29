# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import re
import sys
import time
import urllib.parse

import common
import gerrit_interface


# Thresholds, unit: day(s).
NO_UPDATE_THRESHOLD = 7
CQ_PLUS1_RETRY_THRESHOLD = 1
# Limits
CQ_PLUS1_LIMIT_PER_DAY = 2
CQ_PLUS1_LIMIT = 4

RE_GWSQ_REVIEWER = re.compile(rf"(\w+) is from group\({common.GWSQ_EMAIL}\)")


def parse(obj):
    ret = {}

    ret["cl_num"] = obj["number"]
    ret["change_id"] = obj["id"]
    ret["subject"] = obj["subject"]
    ret["url"] = obj["url"]
    # Example to parse: chromeos-6.6
    ret["branch"] = obj["branch"].split("-")[1]
    ret["messages"] = gerrit_interface.get_messages(
        ret["change_id"], ret["branch"]
    )[::-1]

    for m in ret["messages"]:
        if m["author"]["name"] == "gwsq":
            ret["gwsq_message"] = m
            break
    else:
        ret["gwsq_message"] = None
    ret["gwsq_in_progress"] = False

    max_cr = max_cq = max_v = min_v = 0
    reviewers = set()
    for approval in obj["currentPatchSet"]["approvals"]:
        if approval["type"] == "CRVW":
            max_cr = max(max_cr, int(approval["value"]))
            email = approval["by"]["email"]
            if (
                email.endswith("@chromium.org") or email.endswith("@google.com")
            ) and email != common.GWSQ_EMAIL:
                reviewers.add(email)

            if email == common.GWSQ_EMAIL:
                ret["gwsq_in_progress"] = True

        if approval["type"] == "COMR":
            max_cq = max(max_cq, int(approval["value"]))

        if approval["type"] == "VRIF":
            max_v = max(max_v, int(approval["value"]))
            min_v = min(min_v, int(approval["value"]))

    ret["cr"] = max_cr
    ret["cq"] = max_cq
    if min_v < 0:
        ret["v"] = min_v
    else:
        ret["v"] = max_v
    ret["reviewers"] = list(reviewers)

    return ret


def has_elapsed(from_time, threshold):
    """Return True if `threshold` has elapsed since `from_time`."""
    return (int(time.time()) - from_time) >= threshold


def send_ping_msg(obj):
    """Return True on success."""

    q = [
        "component=167278",
        "format=PLAIN",
        "type=BUG",
        "priority=P2",
        "severity=S2",
        "hotlistIds=5435871",
    ]

    title = f'Kernel: fix build failure for ("{obj["subject"]}")'
    q.append("title=%s" % urllib.parse.quote(title))

    description = f"""Fix build failure for ("{obj["subject"]}").

Link: {obj["url"]}
Branch: {obj["branch"]}
"""
    q.append("description=%s" % urllib.parse.quote(description))

    q = "&".join(q)
    url = f"http://b.corp.google.com/createIssue?{q}"

    msg = f"""
Dear Reviewer,

You have been selected because this bot believes you are the best owner for this CL.

- If you think you are not the right person for handling the CL:
    - Please add chromeos-kernel-findmissing-reviewers@google.com as a reviewer.
    - Please remove yourself from the reviewer list.
- If the CL is irrelevant to ChromeOS, feel free to abandon it.
- If the CL makes sense to you:
    - If the CQ tests passed and the robot has marked V+1 and AS+1, please provide CR+2 and CQ+2.
    - If the CQ tests failed due to build failures, please provide V-1 and file a bug via [the template]({url}).
- Otherwise, please provide CR-1 and your comments.

Thank you for contributing to ChromeOS kernel stability.
"""

    return gerrit_interface.send_message(obj["cl_num"], msg)


def to_mtimestamp(msg):
    """Convert a message date to timestamp."""
    mtime = time.strptime(msg["date"], "%Y-%m-%d %H:%M:%S.000000000")
    return int(time.mktime(mtime))


def add_backup_reviewer(obj):
    """Add a backup reviewer from CLs (with the same Change-Id) in other branches or GWSQ.

    Return True if a backup reviewer (excluding GWSQ) has added.
    """
    if obj["gwsq_in_progress"]:
        print("Deferred, gwsq is still in progress")
        return False
    if obj["gwsq_message"]:
        print("gwsq has involved")
        return False

    cl_nums = gerrit_interface.get_cl_nums(obj["change_id"])
    cl_nums.pop(obj["branch"], None)

    for cl_num in cl_nums.values():
        o = gerrit_interface.inspect(cl_num)[0]
        o = parse(o)

        # Only take chromium accounts into consideration.
        chromium_reviewers = [r for r in o["reviewers"] if "@chromium.org" in r]
        if chromium_reviewers:
            for r in chromium_reviewers:
                gerrit_interface.add_reviewer(obj["cl_num"], r)
                print(f"Add {r} as a reviewer from {cl_num}")
            return True

        if o["gwsq_in_progress"]:
            print(f"Deferred, gwsq is still in progress in {cl_num}")
            return False

        if o["gwsq_message"]:
            match = RE_GWSQ_REVIEWER.search(o["gwsq_message"]["message"])
            if match:
                gwsq_reviewer = match.group(1)
                gerrit_interface.add_reviewer(
                    obj["cl_num"], f"{gwsq_reviewer}@chromium.org"
                )
                print(f"Add {gwsq_reviewer} as a reviewer from {cl_num}")
                return True

    print("Deferred, loop in gwsq")
    gerrit_interface.add_reviewer(obj["cl_num"], common.GWSQ_EMAIL)
    return False


def check_no_update(obj):
    for last, m in enumerate(obj["messages"]):
        if m["author"]["name"] != "gwsq":
            break
    else:
        print("Failed to find the last non-gwsq message")
        return

    if not has_elapsed(
        to_mtimestamp(obj["messages"][last]), 86400 * NO_UPDATE_THRESHOLD
    ):
        return

    print(f"Hasn't been changed for more than {NO_UPDATE_THRESHOLD} day(s)")

    # Don't try to loop in gwsq if a CL has V-1.
    if obj["v"] != -1:
        if obj["gwsq_message"]:
            # gwsq has involved.
            pass
        elif not add_backup_reviewer(obj):
            # gwsq is in progress.
            return

    if not send_ping_msg(obj):
        print("Failed to send ping message")


def check_cq_plus1(obj):
    if obj["cq"] != 0 or obj["v"] == 1:
        print("Already handled")
        return

    if obj["v"] == -1:
        print("Already handled (V-1)")
        return

    cl_num = obj["cl_num"]
    has_failed = 0
    for m in obj["messages"]:
        msg = m["message"]

        # If a new PS has uploaded, retry CQ+1 anyway.
        if "Uploaded patch set" in msg or "was rebased" in msg:
            print("Mark CQ+1 because of a new patchset")
            gerrit_interface.label_cq(cl_num, "1")
            return

        if "This CL has passed the run" in msg:
            # Make sure the latest result passes CQ.
            if has_failed == 0:
                print("CQ+1 has passed")

                if obj["reviewers"] or add_backup_reviewer(obj):
                    print("Send ping message, V+1, and AS+1")
                    send_ping_msg(obj)
                    gerrit_interface.label_v(cl_num, "1")
                    gerrit_interface.label_as(cl_num, "1")
                return

        if "its dependencies weren't CQ-ed" in msg:
            print("Dependencies weren't CQ-ed")

            gerrit_interface.send_message(
                cl_num,
                "Won't CQ+1 again because dependencies weren't CQ-ed",
            )
            gerrit_interface.label_v(cl_num, "-1")
            return

        if "This CL has failed the run" in msg:
            if "buildtest-cq" in msg:
                print("CQ failed on buildtest")

                if obj["reviewers"] or add_backup_reviewer(obj):
                    print("Send ping message, and V-1")
                    send_ping_msg(obj)
                    gerrit_interface.label_v(cl_num, "-1")
                return

            has_failed += 1
            if has_failed >= CQ_PLUS1_LIMIT:
                print("Do nothing because the retries are over the limit")

                if obj["reviewers"] or add_backup_reviewer(obj):
                    print("Send ping message")
                    send_ping_msg(obj)
                    gerrit_interface.send_message(
                        cl_num,
                        "Won't CQ+1 again because the retries are over the limit",
                    )
                    gerrit_interface.label_v(cl_num, "-1")
                return

            if not has_elapsed(
                to_mtimestamp(m), 86400 * CQ_PLUS1_RETRY_THRESHOLD
            ):
                if has_failed >= CQ_PLUS1_LIMIT_PER_DAY:
                    print(
                        "Do nothing because the retries are over the limit today"
                    )
                    return

    print("Mark CQ+1")
    gerrit_interface.label_cq(cl_num, "1")


def ping_cl(cl_num):
    obj = gerrit_interface.inspect(cl_num)[0]
    obj = parse(obj)
    send_ping_msg(obj)


def ping_pending_cls():
    for obj in gerrit_interface.inspect():
        obj = parse(obj)

        print(f"Checking CL {obj['cl_num']}")
        check_no_update(obj)
        check_cq_plus1(obj)


if __name__ == "__main__":
    if len(sys.argv) == 2:
        ping_cl(sys.argv[1])
    else:
        ping_pending_cls()
