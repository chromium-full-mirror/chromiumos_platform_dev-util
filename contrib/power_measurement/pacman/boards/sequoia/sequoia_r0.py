# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Power Supply : PSU Board

"""Config for Sequoia r0"""

PACS = [
    # addr:ch  name           nom    rsense"
    ("0x10:0", "PP1800_MEM_S3", 1.8, 0.005),  # R745
    ("0x10:1", "PP1065_MEM_S3", 1.065, 0.003),  # R1299
    ("0x10:2", "PP1065_SOC_S3", 1.065, 0.003),  # R1300
    ("0x10:3", "PP0500_MEM_S3", 0.5, 0.001),  # R1296
    ("0x11:0", "PPVAR_SYS_R", 20, 0.002),  # R763
    ("0x11:1", "PP1800_Z1", 1.8, 0.1),  # R73
    ("0x11:2", "PP3300_Z1", 3.3, 0.01),  # R535
    ("0x11:3", "PP3300_GSC_Z1", 3.3, 0.01),  # R758
    ("0x12:0", "PPVAR_SYS_VCCSA_VIN", 20, 0.01),  # R735
    ("0x12:1", "PPVAR_SYS_VCCORE_VIN", 20, 0.005),  # R739
    ("0x12:2", "PPVAR_SYS_PP0P77_SOC_S5_VIN", 20, 0.01),  # R736
    ("0x12:3", "PPVAR_SYS_VCCGT_VIN", 20, 0.005),  # R744
    ("0x13:0", "PP1800_S5", 1.8, 0.003),  # R166
    ("0x13:1", "PP3300_S5", 3.3, 0.005),  # RS1
    ("0x13:2", "PP5000_B_Z1", 5, 0.003),  # R775
    ("0x13:3", "PP3300_SSD_X", 3.3, 0.005),  # R506
    ("0x14:0", "PP3300_WLAN_S5", 3.3, 0.01),  # R517
    ("0x14:1", "PP3300_LAN_S5", 3.3, 0.01),  # R753
    ("0x14:2", "PPVAR_SYS_PP1250_SOC_S5_VIN", 20, 0.01),  # R875
    ("0x14:3", "PP5000_A_Z1", 5, 0.003),  # R540
]
RAILS = []
GPIOS = [
    # addr  rail
    (0x10, "PG_MEM_VDD"),
    (0x11, "SOC_PWROK"),
    (0x12, "PLT_PCIE_RST_L"),
    (0x13, "PG_PP1250_SOC_S5"),
    (0x14, "SOC_PROCHOT_R_ODL"),
]
