# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""CPD Configuration file for testing purposes."""

# Config for testing CPD
revs = [0]

# these devices are pac1954 (4-channels/i2c address) devices
inas = [
    # drvname      slv         name         nom   sense   mux   is_calib
    ("pac1954", "0x10:0", "CH1", 12.0, 0.020, "rem", True),
]
