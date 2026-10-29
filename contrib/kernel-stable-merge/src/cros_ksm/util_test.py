# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from datetime import UTC, datetime
from operator import itemgetter

import pytest

from cros_ksm.util import chomp, parse_utc_datetime

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
