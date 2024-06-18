# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable=W0212,W0621

"""Tests for pacina.py file."""

from .. import conftest
from .. import pacina


def test_smoke__run_measurements():
    pacina.run_measurements(
        ftdi_urls=["ftdi://2"],
        configs=["pacina/cpd.tests.py"],
        measurements=2,
        sample_time=0.1,
        device_controller_const=conftest.device_controller_const,
        initialize_ftdi=lambda _: (),
    )


def test_smoke__run_measurements__generate_viz():
    pacina.run_measurements(
        ftdi_urls=["ftdi://2"],
        configs=["pacina/cpd.tests.py"],
        measurements=2,
        sample_time=0.1,
        device_controller_const=conftest.device_controller_const,
        initialize_ftdi=lambda _: (),
        generate_viz=True,
    )
