# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import sys
import time
import urllib.parse

import gerrit_interface


# Thresholds, unit: day(s).
NO_UPDATE_THRESHOLD = 7
CQ_PLUS1_RETRY_THRESHOLD = 1
# Limits
CQ_PLUS1_LIMIT_PER_DAY = 2
CQ_PLUS1_LIMIT = 4


def parse(obj):
    ret = {}

    ret["cl_num"] = obj["number"]
    ret["last_updated"] = obj["lastUpdated"]
    ret["change_id"] = obj["id"]
    ret["subject"] = obj["subject"]
    ret["url"] = obj["url"]
    # Example to parse: chromeos-6.6
    ret["branch"] = obj["branch"].split("-")[1]

    max_cr = max_cq = max_v = min_v = 0
    for approval in obj["currentPatchSet"]["approvals"]:
        if approval["type"] == "CRVW":
            max_cr = max(max_cr, int(approval["value"]))

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


def check_no_update(obj):
    if has_elapsed(int(obj["last_updated"]), 86400 * NO_UPDATE_THRESHOLD):
        print(f"Hasn't been changed for more than {NO_UPDATE_THRESHOLD} day(s)")

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
    for m in reversed(
        gerrit_interface.get_messages(obj["change_id"], obj["branch"])
    ):
        msg = m["message"]

        # If a new PS has uploaded, retry CQ+1 anyway.
        if "Uploaded patch set" in msg:
            print("Mark CQ+1 because of a new patchset")
            gerrit_interface.label_cq(cl_num, "1")
            return

        if "This CL has passed the run" in msg:
            # Make sure the latest result passes CQ.
            if has_failed == 0:
                print("CQ+1 has passed")
                print("Mark V+1 and AS+1")
                gerrit_interface.label_cq(cl_num, "1")
                gerrit_interface.label_as(cl_num, "1")
                return

        if "This CL has failed the run" in msg:
            if "buildtest-cq" in msg:
                print("CQ failed on buildtest")
                print("Mark V-1")
                gerrit_interface.label_v(cl_num, "-1")
                return

            has_failed += 1
            if has_failed >= CQ_PLUS1_LIMIT:
                print("Do nothing because the retries are over the limit")
                return

            mtime = time.strptime(m["date"], "%Y-%m-%d %H:%M:%S.000000000")
            if not has_elapsed(
                int(time.mktime(mtime)), 86400 * CQ_PLUS1_RETRY_THRESHOLD
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
