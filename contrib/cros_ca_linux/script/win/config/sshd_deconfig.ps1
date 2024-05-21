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

# The following command can set powershell as the default shell:
# New-ItemProperty -Path "HKLM:\SOFTWARE\OpenSSH" -Name DefaultShell -Value "C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe" -PropertyType String -Force
# Undo sshd default shell setting with the following command.
Remove-ItemProperty -Path "HKLM:\SOFTWARE\OpenSSH" -Name DefaultShell

# Revoke SeAssignPrimaryTokenPrivilege privilege from current user
Invoke-Expression "${PSScriptRoot}\lib\grant_user_rights.ps1 -r SeAssignPrimaryTokenPrivilege"

Set-Service sshd -StartupType Auto
Start-Service sshd
