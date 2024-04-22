# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

${Principal} = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())

# Ensure that the current envrionment has Administrator privileges.
if (-NOT ${Principal}.IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    throw "Please run in Powershell with Administrator privileges"
}