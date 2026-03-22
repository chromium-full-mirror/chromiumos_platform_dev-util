# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

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
