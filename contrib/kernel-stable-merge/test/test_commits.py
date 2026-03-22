# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from __future__ import annotations

from dataclasses import KW_ONLY, dataclass, field

import pytest

from cros_ksm.git import Commit, Repo


@dataclass
class Expect:

    commit_id: str
    _: KW_ONLY
    subject: str
    reverted_subject: str | None = None
    ported_subject: str | None = None
    parsed_trailers: list[str] = field(default_factory=list)
    ported_commit_ids_from_message: set[str] = field(default_factory=set)
    reverted_commit_ids_from_message: set[str] = field(default_factory=set)
    is_revert: bool = False

    # list of tests that are expected to fail due to bugs
    xfail: list = field(default_factory=list)

def test_commit_id(commit: Commit, expect: Expect):
    assert commit.id == expect.commit_id

def test_commit_is_revert(commit: Commit, expect: Expect):
    assert commit.is_revert == expect.is_revert

def test_subject_text(commit: Commit, expect: Expect):
    assert commit.subject.text == expect.subject

def test_reverted_subject(commit: Commit, expect: Expect):
    if expect.reverted_subject is None:
        assert commit.subject.reverted_subject is None
    else:
        assert commit.subject.reverted_subject is not None
        assert commit.subject.reverted_subject.text == expect.reverted_subject

def test_ported_subject(commit: Commit, expect: Expect):
    if expect.ported_subject is None:
        assert commit.subject.ported_subject == commit.subject
    else:
        assert commit.subject.ported_subject is not None
        assert commit.subject.ported_subject.text == expect.ported_subject

def test_parsed_trailers(commit: Commit, expect: Expect):
    parsed_trailers = list(map(str, commit.message.trailers.values()))
    assert parsed_trailers == expect.parsed_trailers

def test_ported_commit_ids_from_message(commit: Commit, expect: Expect):
    actual = commit.message.ported_oids
    assert actual == expect.ported_commit_ids_from_message

def test_reverted_commit_ids_from_message(commit: Commit, expect: Expect):
    actual = commit.message.reverted_oids
    assert actual == expect.reverted_commit_ids_from_message

expect_list: list[Expect] = [
    # Cherry picked commit id has format:
    #
    # ```
    # [ Upstream commit 3d07b691ee707c00afaf365440975e81bb96cd9b ]
    # ```
    Expect(
        "fa361565a7275cc43c6ca1abec9ec4fcc9ec51f1",
        subject = "selftest/ptp: update ptp selftest to exercise the gettimex options",
        parsed_trailers = [
            'Stable-dep-of: 76868642e427 ("testptp: Add option to open PHC in readonly mode")',
        ],
        ported_commit_ids_from_message = {"3d07b691ee707c00afaf365440975e81bb96cd9b"},
    ),

    # The revert description contains two commit ids.
    #
    # ```
    # This reverts commit fa361565a7275cc43c6ca1abec9ec4fcc9ec51f1, which is
    # commit 3d07b691ee707c00afaf365440975e81bb96cd9b upstream.
    # ```
    Expect(
        "6b2176a5c99b33f3c4acc04faadaa9c75da7b163",
        subject = "Revert \"selftest/ptp: update ptp selftest to exercise the gettimex options\"",
        reverted_subject = "selftest/ptp: update ptp selftest to exercise the gettimex options",
        reverted_commit_ids_from_message = {
            "fa361565a7275cc43c6ca1abec9ec4fcc9ec51f1",
            "3d07b691ee707c00afaf365440975e81bb96cd9b",
        },
        is_revert = True,
    ),

    # Reverted subject is quoted with ' instead of ".
    Expect(
        "879d77d0cbe20ad1d6099a1e16f03b72c0649828",
        subject = "Revert 'perf evlist: Fix creation of cpu map'",
        reverted_subject = "perf evlist: Fix creation of cpu map",
        is_revert = True,
    ),

    # Subject is a revert of a revert.
    Expect(
        "ea7d521569a70418aa9f6309a1d1916709818b62",
        subject = "Revert 'Revert \"ACPICA: Permanently set _REV to the value '2'.\"'",
        reverted_subject = "Revert \"ACPICA: Permanently set _REV to the value '2'.\"",
        is_revert = True,
    ),

    # Subject contains a component prefix before the revert.
    Expect(
        "104c229932f18605acef2351373f8fd8f6c6f8c6",
        subject = "ASoC: Revert 'ASoC: imx-ssi: Remove mono support'",
        reverted_subject = "ASoC: imx-ssi: Remove mono support",
        is_revert = True,
    ),

    # Subject contains a quoted reverted subject that itself contains quoted
    # text.
    Expect(
        "de3f906f0af0c3f5d862b07df4d020c9322cd9c3",
        subject = "gfs2: Revert 'Fix \"truncate in progress\" hang'",
        reverted_subject = "Fix \"truncate in progress\" hang",
        is_revert = True,
    ),

    # Commit is a revert but the subject does not contain a quoted subject.
    #
    # Subject looks like it contains a reverted subject, but the quoted text is
    # merely the topic of the revert.
    #
    # Revert of multiple commits.
    #
    # Message contains a list of reverted oids and subjects.
    #
    # ```
    # This reverts the following set of commits:
    #
    # d1ed113f1669390da9898da3beddcc058d938587 ("ipv6: remove duplicate neigh_ifdown")
    # 29ba5fed1bbd09c2cba890798c8f9eaab251401d ("ipv6: don't flush routes when setting loopback down")
    # 9d82ca98f71fd686ef2f3017c5e3e6a4871b6e46 ("ipv6: fix missing in6_ifa_put in addrconf")
    # [...]
    # ```
    Expect(
        "73a8bd74e2618990dbb218c3d82f53e60acd9af0",
        xfail = [
            test_commit_is_revert,
        ],
        subject = "ipv6: Revert 'administrative down' address handling changes.",
        reverted_subject = None,
        is_revert = True,
    ),

    # Message presents the reverted oid in a blockquote from the git log.
    #
    # ```
    # Revert the following patch:
    #
    #         commit c08c6e1f54c85fc299cf9f88cf330d6dd28a9a1d
    #         Author: Steven J. Magnani <steve@digidescorp.com>
    #         Date:   Fri Mar 5 13:42:24 2010 -0800
    #
    #         nommu: get_user_pages(): pin last page on non-page-aligned start
    # ```
    Expect(
        "7561e8ca0dfaf6fca3feef982830de3b65300e5b",
        subject = "NOMMU: Revert 'nommu: get_user_pages(): pin last page on non-page-aligned start'",
        reverted_subject = "nommu: get_user_pages(): pin last page on non-page-aligned start",
        is_revert = True,
    ),

    # Message describes the revert multiple times because this is a squashed
    # commit with a squashed message. We should report the reverted oid exactly
    # once.
    #
    # Message describes a reverted commit using older syntax:
    #
    # ```
    # Revert 'commit 809216233555 ("selftests/harness: remove use of LINE_MAX")'
    # as that came in quite late in the 6.9 cycle.
    # ```
    Expect(
        "6bb955fce08cbc8495a72755130d2d220994faee",
        subject = "Revert \"selftests/harness: remove use of LINE_MAX\"",
        reverted_subject = "selftests/harness: remove use of LINE_MAX",
        reverted_commit_ids_from_message = {
            "8092162335554c8ef5e7f50eff68aa9cfbdbf865",
            "809216233555",
        },
        parsed_trailers = [
            'Fixes: 809216233555 ("selftests/harness: remove use of LINE_MAX")',
        ],
        is_revert = True,
    ),

    # Message has multiple 'Fixes' trailers.
    #
    # Cherry picked commit is has format:
    #
    # ```
    # commit 48fd7ebe00c1cdc782b42576548b25185902f64c upstream.
    # ```
    Expect(
        "43a27836ac98412dfda7cc15a053f5f229f7fc25",
        subject = "Revert \"bcache: remove heap-related macros and switch to generic min_heap\"",
        reverted_subject = "bcache: remove heap-related macros and switch to generic min_heap",
        parsed_trailers = [
            'Fixes: 866898efbb25 ("bcache: remove heap-related macros and switch to generic min_heap")',
            'Fixes: 92a8b224b833 ("lib/min_heap: introduce non-inline versions of min heap API functions")',
        ],
        ported_commit_ids_from_message = {"48fd7ebe00c1cdc782b42576548b25185902f64c"},
        reverted_commit_ids_from_message = {"866898efbb25bb44fd42848318e46db9e785973a"},
        is_revert = True,
    ),

    # Subject has port prefix "CHROMIUM".
    #
    # Subject has difficult format: `{port_prefix}: {component_prefix}: Revert "reverted_subject}"`.
    #
    # Cherry picked commit id has an unusual format in a trailer:
    #
    #   iwl7000-tree: 02bfddf388c9c4de92ca7b9a763b38ad72bd76aa
    Expect(
        "f39e824e90e852ccf5564e673ad4c2e7bf745d96",
        subject = "CHROMIUM: iwl7000: Revert \"wifi: iwlwifi: set IEEE80211_HE_MAC_CAP0_TWT_REQ in the he cap elem\"",
        reverted_subject = "wifi: iwlwifi: set IEEE80211_HE_MAC_CAP0_TWT_REQ in the he cap elem",
        ported_subject = "iwl7000: Revert \"wifi: iwlwifi: set IEEE80211_HE_MAC_CAP0_TWT_REQ in the he cap elem\"",
        is_revert = True,
        ported_commit_ids_from_message = {"02bfddf388c9c4de92ca7b9a763b38ad72bd76aa"},
        reverted_commit_ids_from_message = {"6c3c2170d60337f9c55c4158588961847b016a2a"},
    ),

    # Subject has port prefix "FROMLIST".
    Expect(
        "7528fe27ea178308b1727168c664610d0f3b97c5",
        subject = "FROMLIST: net: usb: r8152: fix resume reset deadlock",
        ported_subject = "net: usb: r8152: fix resume reset deadlock",
        is_revert = False,
    ),

    # Subject has multiple port prefixes.
    Expect(
        "28ad5668e00b3453db7df0fb671f777d0107cea6",
        subject = "BACKPORT: FROMGIT: platform/chrome: cros_ec_chardev: Introduce chardev_data",
        ported_subject = "platform/chrome: cros_ec_chardev: Introduce chardev_data",
        is_revert = False,
        ported_commit_ids_from_message = {"5ad1406881102bc3e94fd25311d1728dfaf435a2"},
    ),

    # "Subject has misspelled port prefix "FORMGIT".
    Expect(
        "601049ee152a771bcdac536ed523bef34d364108",
        subject = "FORMGIT: platform: chrome: Put docs with the code",
        ported_subject = "platform: chrome: Put docs with the code",
        ported_commit_ids_from_message = {"c402f645d36c4a4c8675ea360d92cc71592e9e0a"},
        is_revert = False,
    ),

    # Trailer contains a subject that is difficult to parse.
    Expect(
        "0f7908a016c092cfdaa16d785fa5099d867bc1a3",
        subject = "PCI/ASPM: Fix deadlock when enabling ASPM",
        ported_commit_ids_from_message = {"1e560864159d002b453da42bd2c13a1805515a20"},
        parsed_trailers = [
            'Fixes: f93e71aea6c6 ("Revert "PCI/ASPM: Remove pcie_aspm_pm_state_change()"")',
        ],
        is_revert = False,
    ),
]

@pytest.fixture(
    params = expect_list,
    ids = lambda expect: expect.commit_id[:12],
)
def expect(request: pytest.FixtureRequest) -> Expect:
    xinfo: Expect = request.param

    # Maybe mark the current test as an expected failure
    if request.function in xinfo.xfail:
        request.applymarker(pytest.mark.xfail)

    return xinfo

@pytest.fixture
def commit(repo: Repo, expect: Expect) -> Commit:
    return repo.resolve_refish_to_commit(expect.commit_id)
