# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

& "${PSScriptRoot}\lib\check_admin.ps1"

# Delete sshd startup task.
Unregister-ScheduledTask -TaskName sshd -Confirm:$false

Get-Process sshd | Stop-Process -Force

# Uninstall sshd.
${Result} = Get-WindowsCapability -Online | Where-Object Name -like OpenSSH.Server* 
if (${Result}.State -eq "Installed") {
    "Uninstalling $(${Result}.Name)"
	${Result} | Remove-WindowsCapability -Online
} else {
    "OpenSSH is not installed"
}

# Remove config.
Remove-Item -r "${env:ProgramData}\ssh"
