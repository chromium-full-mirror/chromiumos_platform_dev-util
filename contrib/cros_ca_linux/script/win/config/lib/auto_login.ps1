# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

param (
    [Alias("u")][string]${UserName} = ${env:USERNAME},
    [Parameter(Mandatory = $true, Position = 1)][string]${PW}
)

& "${PSScriptRoot}\check_admin.ps1"

# Modify registry to enable automatic login.
New-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon" -Name AutoAdminLogon -Value "1" -PropertyType String -Force
New-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon" -Name DefaultUserName -Value ${UserName} -PropertyType String -Force
New-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon" -Name DefaultPassword -Value ${PW} -PropertyType String -Force
