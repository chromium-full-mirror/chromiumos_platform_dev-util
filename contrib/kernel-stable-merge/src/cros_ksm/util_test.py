# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import urllib.request
from datetime import UTC, datetime
from io import BytesIO
from operator import itemgetter

import pytest

from cros_ksm.util import (
    MTV_TZ,
    chomp,
    fetch_milestone_branch_date,
    parse_utc_datetime,
)

chomp_data = [
    # easy
    (b"", b""),
    (b"\n", b""),
    (b"x", b"x"),
    (b"x\n", b"x"),
    (b"x\r", b"x"),
    (b"x\r\n", b"x"),

    # weird
    (b"x\n\n", b"x\n"),
    (b"x\n ", b"x\n "),
]

@pytest.mark.parametrize("data", chomp_data, ids = itemgetter(0))
def test_chomp_bytes(data):
    assert chomp(data[0]) == data[1]

@pytest.mark.parametrize("data", chomp_data, ids = itemgetter(0))
def test_chomp_str(data):
    assert chomp(data[0].decode()) == data[1].decode()

parse_utc_datetime_data = [
    ("2026-10-26", datetime(2026, 10, 26, 0, 0, 0, tzinfo=UTC)),
    ("2026-10-26T00:00:00", datetime(2026, 10, 26, 0, 0, 0, tzinfo=UTC)),
    ("2026-10-26T08:01:35Z", datetime(2026, 10, 26, 8, 1, 35, tzinfo=UTC)),
    ("2026-10-26T17:01:35+09:00", datetime(2026, 10, 26, 8, 1, 35, tzinfo=UTC)),
]

@pytest.mark.parametrize("data", parse_utc_datetime_data, ids = itemgetter(0))
def test_parse_utc_datetime(data):
    assert parse_utc_datetime(data[0]) == data[1]

def test_parse_utc_datetime_default_tz():
    # 2026-10-26 is in PDT (UTC-7); 2026-01-19 is in PST (UTC-8).
    assert parse_utc_datetime("2026-10-26T00:00:00", default_tz=MTV_TZ) == (
        datetime(2026, 10, 26, 7, 0, 0, tzinfo=UTC)
    )
    assert parse_utc_datetime("2026-01-19T00:00:00", default_tz=MTV_TZ) == (
        datetime(2026, 1, 19, 8, 0, 0, tzinfo=UTC)
    )

def test_fetch_milestone_branch_date(monkeypatch: pytest.MonkeyPatch):
    def fake_urlopen(url: str, **_kwargs):
        assert url == "https://chromiumdash.appspot.com/fetch_milestone_schedule?mstone=158"
        return BytesIO(b'{"mstones":[{"branch_point":"2026-10-26T00:00:00","mstone":158}]}')

    monkeypatch.setattr(urllib.request, "urlopen", fake_urlopen)
    assert fetch_milestone_branch_date(158) == datetime(2026, 10, 26, 7, 0, 0, tzinfo=UTC)
