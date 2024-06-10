"""Configuration file for CPD device."""

# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Config for testing CPD
revs = [0]

# these devices are pac1954 (4-channels/i2c address) devices
inas = [
    # drvname      slv         name         nom   sense   mux   is_calib
    # When R10 is used
    ("pac1954", "0x10:1", "VBAT_CH2", 12.0, 0.020, "rem", True),
    # When R17 is used
    # ("pac1954", "0x10:2", "VBAT_CH3", 12.0, 0.020, "rem", True),
]
