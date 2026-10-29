# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from datetime import UTC, datetime

import pytest

from .git import Commit, KernelVersion, Message, Repo, Subject, Tag


def test_subject_hash():
    subject1 = Subject.from_parse("xyz")
    subject2 = Subject.from_parse("xyz")
    assert hash(subject1) == hash(subject2)

def test_message_hash():
    message1 = Message("xyz")
    message2 = Message("xyz")
    assert hash(message1) == hash(message2)

def test_commit_unique(repo: Repo):
    commit1: Commit = repo.resolve_refish_to_commit("v3.14")
    commit2: Commit = repo.resolve_refish_to_commit("v3.14")
    assert commit1 is commit2

def test_tag_unique(repo: Repo):
    tag1: Tag = repo.resolve_tag_name("v3.14")
    tag2: Tag = repo.resolve_tag_name("v3.14")
    assert tag1 is tag2

def test_kernel_version_hash():
    kver1 = KernelVersion(6, 12)
    kver2 = KernelVersion(6, 12)
    assert hash(kver1) == hash(kver2)

def test_get_target_release_info_eval_date(repo: Repo, monkeypatch: pytest.MonkeyPatch):
    monkeypatch.setattr(
        repo,
        "get_last_merged_release",
        lambda _rev: KernelVersion(6, 12, 91),
    )

    eval_date = datetime(2026, 10, 26, 0, 0, 0, tzinfo=UTC)
    info = repo.get_target_release_info(eval_date=eval_date)
    assert info.eval_date == eval_date
    assert info.search_max_date == datetime(2026, 9, 28, 0, 0, 0, tzinfo=UTC)
    assert info.last_merged_release_tag.name == "v6.12.91"
    assert info.target_release_tag is not None
    assert info.target_release_tag.name == "v6.12.111"

    # When the search window is older than the last merged release,
    # target_release_tag should be None.
    early_date = datetime(2026, 6, 1, 0, 0, 0, tzinfo=UTC)
    early_info = repo.get_target_release_info(eval_date=early_date)
    assert early_info.eval_date == early_date
    assert early_info.target_release_tag is None
