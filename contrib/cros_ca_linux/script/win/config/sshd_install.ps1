# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

& "${PSScriptRoot}\lib\check_admin.ps1"

# Install sshd.
${Result} = Get-WindowsCapability -Online | Where-Object Name -like OpenSSH.Server*
if (${Result}.State -eq "Installed") {
	"OpenSSH has already installed"
} else {
    "Installing $(${Result}.Name)"
	${Result} | Add-WindowsCapability -Online
}
