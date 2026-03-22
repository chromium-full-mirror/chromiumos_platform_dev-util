# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from operator import itemgetter

import pytest

from cros_ksm.util import chomp

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
