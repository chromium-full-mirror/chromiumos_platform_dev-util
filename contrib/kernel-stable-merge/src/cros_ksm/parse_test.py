# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import itertools

import pytest
from parsy import ParseError

from . import parse as p


@pytest.mark.parametrize(
    argnames = (
        "input",
        "expect_inner",
        "expect_remainder",
    ),
    argvalues = [
        # empty string
        (
            '""',
            "",
            "",
        ),

        # with ""
        (
            '"hello"',
            'hello',
            '',
        ),
        (
            '"hello" ',
            'hello',
            ' ',
        ),
        (
            '"hello""',
            'hello"',
            '',
        ),
        (
            '"hello"" ',
            'hello"',
            ' '
        ),
        (
            '"hello" " ',
            'hello" ',
            ' '
        ),
        (
            '"hello"""',
            'hello""',
            '',
        ),
        (
            '"he"llo"',
            'he"llo',
            '',
        ),

        # with []
        (
            "[hello]",
            "hello",
            "",
        ),
        (
            "[hello] ",
            "hello",
            " ",
        ),
        (
            "[hello]]",
            "hello]",
            "",
        ),
        (
            "[hello]] ",
            "hello]",
            " "
        ),
        (
            "[he[llo] ] ",
            "he[llo] ",
            " "
        ),
        (
            "[hello[][",
            "hello[",
            "[",
        ),
    ],
    ids = itertools.count(), # pyrefly: ignore[bad-specialization]
)
def test_clumsy_string(input: str, expect_inner: str, expect_remainder: str):
    actual_cs, actual_remainder = p.clumsy_string.parse_partial(input)
    assert actual_cs.inner_source.text == expect_inner
    assert actual_remainder == expect_remainder

@pytest.mark.parametrize(
    argnames = "input",
    argvalues = [
        '"',
        ' "hello"',
        '"hello',
    ],
    ids = itertools.count(), # pyrefly: ignore[bad-specialization]
)
def test_clumsy_string_error(input: str):
    with pytest.raises(ParseError):
        p.clumsy_string.parse_partial(input)

@pytest.mark.parametrize(
    argnames = ("expect", "text"),
    argvalues = [
        ("a subject", '"a subject"'),
        ("a subject", "'a subject'"),
        ("a subject", '("a subject")'),
        ("a subject", "('a subject')"),

        ("a subject", '("a subject", extra)'),
        ("a subject", '("a subject" extra)'),
        ("a subject", '("a subject") extra'),
        ("a subject", '("a subject" ) extra'),
        ("a subject", '("a subject" extra) extra'),

        # Variations on a difficult-to-parse subject in the Fixes trailer in
        # 0f7908a016c092cfdaa16d785fa5099d867bc1a3 (v6.1.88~1).
        ("Revert \"PCI/ASPM: Remove pcie_aspm_pm_state_change()\"", '("Revert "PCI/ASPM: Remove pcie_aspm_pm_state_change()"")'),
        ("Revert \"PCI/ASPM: Remove pcie_aspm_pm_state_change()\"", "('Revert \"PCI/ASPM: Remove pcie_aspm_pm_state_change()\"')"),
        ("Revert \"PCI/ASPM: Remove pcie_aspm_pm_state_change()\"", '(Revert "PCI/ASPM: Remove pcie_aspm_pm_state_change()")'),
        ("Revert \"PCI/ASPM: Remove pcie_aspm_pm_state_change()\"", '"Revert "PCI/ASPM: Remove pcie_aspm_pm_state_change()""'),
        ("Revert \"PCI/ASPM: Remove pcie_aspm_pm_state_change()\"", '("Revert "PCI/ASPM: Remove pcie_aspm_pm_state_change()"", extra)'),
    ],
    ids = itertools.count(), # pyrefly: ignore[bad-specialization]
)
def test_trailer_subject(text: str, expect: str):
    subject = p.trailer_subject.parse(text)
    assert isinstance(subject, p.Subject)
    assert subject.source.text == expect

# Test a difficult trailer in 0f7908a016c092cfdaa16d785fa5099d867bc1a3 (v6.1.88~1).
def test_trailer_fixes_0f7908a016c0():
    trailer = p.trailer.parse('Fixes: f93e71aea6c6 ("Revert "PCI/ASPM: Remove pcie_aspm_pm_state_change()"")\n')
    assert trailer.key == "Fixes"
    assert trailer.oid == "f93e71aea6c6"
    assert trailer.subject is not None
    assert trailer.subject.source.text == 'Revert "PCI/ASPM: Remove pcie_aspm_pm_state_change()"'
    assert trailer.subject.inner_type == p.SubjectInnerType.REVERT
    assert trailer.subject.inner_subject is not None
    assert trailer.subject.inner_subject.source.text == "PCI/ASPM: Remove pcie_aspm_pm_state_change()"
