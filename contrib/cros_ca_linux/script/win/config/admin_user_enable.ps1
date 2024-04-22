# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

& "${PSScriptRoot}\lib\check_admin.ps1"

Disable-LocalUser -Name "Administrator"

# Disable automatic login.
& "${PSScriptRoot}\lib\remove_auto_login.ps1"
