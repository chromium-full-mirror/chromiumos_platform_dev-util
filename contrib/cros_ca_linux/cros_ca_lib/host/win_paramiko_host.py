# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""The paramiko Windows host implementation."""

# The following line disable the abstract function implementation warning.
# pylint: disable=W0223

from cros_ca_lib.host import paramiko_host
from cros_ca_lib.host import win_remote_host


class WinParamikoHost(
    win_remote_host.WinRemoteHost, paramiko_host.ParamikoHost
):
    """A class representing a remote Windows host communicating with Paramiko"""
