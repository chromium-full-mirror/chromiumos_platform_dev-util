# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""provision a new cpd device via pyftdi"""

import os
import random

from pyftdi import eeprom
from pyftdi import usbtools


CONFIG_DEFAULTS = "ft4232h_template.ini"
FT4232H_VPS = [(0x0403, 0x6011)]


def provision_device(device_url_to_provision):
    device_eeprom = eeprom.FtdiEeprom()
    device_eeprom.open(device_url_to_provision)

    # loads device defaults and fills serial number field
    with open(
        os.path.join(os.path.dirname(__file__), CONFIG_DEFAULTS),
        encoding="utf-8",
    ) as config:
        device_eeprom.load_config(config, section="values")

    random.seed()
    serial = f"FTDIGEN{random.randrange(1, 2**32):#010}"
    device_eeprom.set_serial_number(serial)

    device_eeprom.commit(dry_run=False)


def find_unique_devices():
    return usbtools.UsbTools.find_all(FT4232H_VPS, nocache=False)


if __name__ == "__main__":
    devices = find_unique_devices()
    for device in devices:
        ((_, _, bus, addr, sn, _, _), _) = device
        device_url = f"ftdi://::{bus:x}:{addr:x}/1"
        provision_request = f"provision device {device_url} (s/n {sn})? (y/n)"
        if not input(provision_request) == "y":
            print("skipped", device_url)
            continue

        provision_device(device_url)
        print("provisioned", device_url)
