#!/usr/bin/env python3
# Copyright 2025 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Unit tests for find_missing_release_cps.py"""

import copy
import json
import subprocess
import unittest
from unittest import mock

import find_missing_release_cps


### Set up test data.
TEST_DATA_M101_CL = "https://chromium-review.googlesource.com/c/path/+/1234567"
TEST_DATA_M100_CL = "https://chrome-internal-review.googlesource.com/7654321"
TEST_DATA_M100_BRANCH = "release-R100-456.B"

TEST_DATA_BUG_ID = "987654321"
TEST_DATA_BUG_COMMENTS_STR = """From: git@watcher
> https://chromium-review.googlesource.com/c/path/+/1234567

> From: git@watcher
> https://chrome-internal-review.googlesource.com/7654321

> From: git@watcher
> https://some.other.url.google.com/1234567"""
TEST_DATA_BUG_COMMENTS_LIST = [
    f"git@watcher\n> {TEST_DATA_M101_CL}\n\n> ",
    f"git@watcher\n> {TEST_DATA_M100_CL}\n\n> ",
    "git@watcher\n> https://some.other.url.google.com/1234567",
]
TEST_DATA_GITWATCHER_CL_M101 = set([TEST_DATA_M101_CL])
TEST_DATA_GITWATCHER_CL_M100 = set([TEST_DATA_M100_CL])
TEST_DATA_GITWATCHER_CLS = set([TEST_DATA_M101_CL, TEST_DATA_M100_CL])
TEST_DATA_GITWATCHER_USER = "git@watcher"
TEST_DATA_GERRIT_CL_MAIN_DETAILS = {
    "branch": "main",
    "subject": "lib: fix the bug",
    "project": "chromeos/project",
}
TEST_DATA_GERRIT_CL_M100_DETAILS = {
    "branch": TEST_DATA_M100_BRANCH,
    "subject": "lib: fix the bug",
    "project": "chromeos/project",
}
TEST_DATA_M100 = "100"
TEST_DATA_M101 = "101"

TEST_DATA_SUMMARY_BY_MSTONE = {
    101: [
        find_missing_release_cps.ClInfo(
            change_number="1234567",
            uri=TEST_DATA_M101_CL,
            subject="lib: fix the bug",
            branch="main",
            project="chromeos/project",
            mstone=101,
        ),
    ],
    100: [
        find_missing_release_cps.ClInfo(
            change_number="7654321",
            uri=TEST_DATA_M100_CL,
            subject="lib: fix the bug",
            branch=TEST_DATA_M100_BRANCH,
            project="chromeos/project",
            mstone=100,
        ),
    ],
}

TEST_DATA_SUMMARY_BY_MSTONE_WITH_MISSING = copy.deepcopy(
    TEST_DATA_SUMMARY_BY_MSTONE
)
del TEST_DATA_SUMMARY_BY_MSTONE_WITH_MISSING[100]


class TestCheckArgs(unittest.TestCase):
    """Test find_missing_release_cps.check_args"""

    def test_args_success(self):
        args = find_missing_release_cps.set_up_args()
        args.bug_id = TEST_DATA_BUG_ID
        args.tot_milestone = TEST_DATA_M101
        find_missing_release_cps.check_required_args(args)

    def test_check_args_bug_id(self):
        args = find_missing_release_cps.set_up_args()
        with self.assertRaises(SystemExit):
            find_missing_release_cps.check_required_args(args)

    def test_check_args_tot_milestone(self):
        args = find_missing_release_cps.set_up_args()
        args.bug_id = TEST_DATA_BUG_ID
        with self.assertRaises(SystemExit):
            find_missing_release_cps.check_required_args(args)


class TestCheckGcert(unittest.TestCase):
    """Test find_missing_release_cps.check_gcert"""

    @mock.patch("subprocess.run")
    def test_gcert_success(self, mock_run):
        mock_run.return_value.returncode = 0
        find_missing_release_cps.check_gcert()

    @mock.patch("subprocess.run")
    def test_gcert_missing(self, mock_run):
        mock_run.return_value.returncode = 1
        with self.assertRaises(SystemExit):
            find_missing_release_cps.check_gcert()


class TestBranchToMilestone(unittest.TestCase):
    """Test find_missing_release_cps.branch_to_mstone"""

    def test_main_branch(self):
        milestone = find_missing_release_cps.branch_to_mstone(
            "main", TEST_DATA_M101
        )
        self.assertEqual(101, milestone)

    def test_release_branch(self):
        milestone = find_missing_release_cps.branch_to_mstone(
            TEST_DATA_M100_BRANCH, TEST_DATA_M101
        )
        self.assertEqual(100, milestone)

    def test_non_release_branch(self):
        milestone = find_missing_release_cps.branch_to_mstone(
            "branch-12345.B", TEST_DATA_M101
        )
        self.assertEqual(-1, milestone)


class TestFindBugComments(unittest.TestCase):
    """Test find_missing_release_cps.find_bug_comments"""

    @mock.patch("subprocess.run")
    def test_bug_comments_success(self, mock_run):
        mock_run.return_value.returncode = 0
        mock_run.return_value.stdout = TEST_DATA_BUG_COMMENTS_STR.encode(
            "utf-8"
        )
        comments = find_missing_release_cps.find_bug_comments(TEST_DATA_BUG_ID)
        self.assertEqual(3, len(comments))
        self.assertEqual(TEST_DATA_BUG_COMMENTS_LIST, comments)

    @mock.patch("subprocess.run")
    def test_bug_comments_missing(self, mock_run):
        mock_run.return_value.returncode = 1
        mock_run.return_value.stdout = "".encode("utf-8")
        with self.assertRaises(SystemExit):
            find_missing_release_cps.find_bug_comments(TEST_DATA_BUG_ID)


class TestFindClsFromGitwatcherComments(unittest.TestCase):
    """Test find_missing_release_cps.find_cls_from_gitwatcher_comments"""

    def test_gitwatcher_cls_success(self):
        cls = find_missing_release_cps.find_cls_from_gitwatcher_comments(
            TEST_DATA_BUG_COMMENTS_LIST,
            TEST_DATA_BUG_ID,
            TEST_DATA_GITWATCHER_USER,
        )
        self.assertEqual(TEST_DATA_GITWATCHER_CLS, cls)

    def test_no_gitwatcher_cls(self):
        with self.assertRaises(SystemExit):
            find_missing_release_cps.find_cls_from_gitwatcher_comments(
                [], TEST_DATA_BUG_ID, TEST_DATA_GITWATCHER_USER
            )


class TestFindClDetails(unittest.TestCase):
    """Test find_missing_release_cps.find_cl_details"""

    @mock.patch("subprocess.run")
    def test_find_cl_details_success(self, mock_run):
        mock_run.return_value.returncode = 0
        mock_run.return_value.stdout = json.dumps(
            TEST_DATA_GERRIT_CL_MAIN_DETAILS
        ).encode("utf-8")
        summary_by_mstone = find_missing_release_cps.find_cl_details(
            TEST_DATA_GITWATCHER_CL_M101, TEST_DATA_BUG_ID, TEST_DATA_M101
        )
        self.assertEqual(
            {101: TEST_DATA_SUMMARY_BY_MSTONE[101]}, summary_by_mstone
        )

    @mock.patch("subprocess.run")
    def test_find_cl_details_with_leading_chars_success(self, mock_run):
        # Test workaround for gob-curl returning ")]}'\n".
        mock_run.return_value.returncode = 0
        cl_details_test_data = json.dumps(TEST_DATA_GERRIT_CL_M100_DETAILS)
        mock_run.return_value.stdout = f")]}}'\n{cl_details_test_data}".encode(
            "utf-8"
        )
        summary_by_mstone = find_missing_release_cps.find_cl_details(
            TEST_DATA_GITWATCHER_CL_M100, TEST_DATA_BUG_ID, TEST_DATA_M100
        )
        self.assertEqual(
            {100: TEST_DATA_SUMMARY_BY_MSTONE[100]}, summary_by_mstone
        )

    @mock.patch("subprocess.run")
    def test_find_cl_details_multiple_cls(self, mock_run):
        mock_run.side_effect = [
            subprocess.CompletedProcess(
                args=[],
                returncode=0,
                stdout=json.dumps(TEST_DATA_GERRIT_CL_M100_DETAILS).encode(
                    "utf-8"
                ),
            ),
            subprocess.CompletedProcess(
                args=[],
                returncode=0,
                stdout=json.dumps(TEST_DATA_GERRIT_CL_MAIN_DETAILS).encode(
                    "utf-8"
                ),
            ),
        ]
        summary_by_mstone = find_missing_release_cps.find_cl_details(
            TEST_DATA_GITWATCHER_CLS, TEST_DATA_BUG_ID, TEST_DATA_M101
        )
        self.assertEqual(TEST_DATA_SUMMARY_BY_MSTONE, summary_by_mstone)

    @mock.patch("subprocess.run")
    def test_find_cl_details_duplicates_removed(self, mock_run):
        mock_run.return_value.returncode = 0
        mock_run.return_value.stdout = json.dumps(
            TEST_DATA_GERRIT_CL_MAIN_DETAILS
        ).encode("utf-8")
        summary_by_mstone = find_missing_release_cps.find_cl_details(
            set([TEST_DATA_M101_CL, TEST_DATA_M101_CL]),
            TEST_DATA_BUG_ID,
            TEST_DATA_M101,
        )
        self.assertEqual(
            {101: TEST_DATA_SUMMARY_BY_MSTONE[101]}, summary_by_mstone
        )

    @mock.patch("subprocess.run")
    def test_find_cl_details_invalid_json(self, mock_run):
        mock_run.return_value.returncode = 0
        mock_run.return_value.stdout = "{invalid json ]]]".encode("utf-8")
        with self.assertRaises(SystemExit):
            find_missing_release_cps.find_cl_details(
                TEST_DATA_GITWATCHER_CLS,
                TEST_DATA_BUG_ID,
                TEST_DATA_M101,
            )


class TestIdentifyPossibleMissingCps(unittest.TestCase):
    """Test find_missing_release_cps.identify_possible_missing_cps"""

    def test_find_cl_details_success(self):
        missing_cps = find_missing_release_cps.identify_possible_missing_cps(
            TEST_DATA_SUMMARY_BY_MSTONE,
            TEST_DATA_BUG_ID,
            TEST_DATA_M101,
            TEST_DATA_M100,
        )
        self.assertEqual({}, missing_cps)

    def test_find_cl_details_missing_cp(self):
        missing_cps = find_missing_release_cps.identify_possible_missing_cps(
            TEST_DATA_SUMMARY_BY_MSTONE_WITH_MISSING,
            TEST_DATA_BUG_ID,
            TEST_DATA_M101,
            TEST_DATA_M100,
        )
        # Expect the M101 CL to show as missing on M100.
        self.assertEqual(
            {100: TEST_DATA_SUMMARY_BY_MSTONE_WITH_MISSING[101]}, missing_cps
        )


if __name__ == "__main__":
    unittest.main()
