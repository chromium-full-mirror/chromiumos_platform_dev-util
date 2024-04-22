# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

#TODO: To be deleted after all DUTs are converted to default SSHD config.

& "${PSScriptRoot}\lib\check_admin.ps1"

# Delete sshd startup task.
Unregister-ScheduledTask -TaskName sshd -Confirm:$false

Get-Process sshd | Stop-Process -Force

# Remove config.
Remove-Item -r "${env:ProgramData}\ssh"

# Undo sshd default shell setting.
Remove-ItemProperty -Path "HKLM:\SOFTWARE\OpenSSH" -Name DefaultShell

# Revoke SeAssignPrimaryTokenPrivilege privilege from current user
Invoke-Expression "${PSScriptRoot}\lib\grant_user_rights.ps1 -r SeAssignPrimaryTokenPrivilege"

Set-Service sshd -StartupType Auto
Start-Service sshd
