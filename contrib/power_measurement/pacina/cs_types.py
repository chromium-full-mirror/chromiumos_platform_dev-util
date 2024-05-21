# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Common types used by pacina.py"""

import enum
import typing as t


ALL_CHANNELS_ENABLED: t.Final[int] = 0b0000
ALL_CHANNELS_DISABLED: t.Final[int] = 0b1111


class Polarity(enum.Enum):
    """Polarity options for current sensor ICs."""

    UNIPOLAR = "unipolar"
    BIPOLAR = "bipolar"

    def __str__(self):
        return str(self.value)
