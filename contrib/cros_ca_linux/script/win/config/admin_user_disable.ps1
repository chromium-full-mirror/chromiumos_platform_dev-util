# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

param (
    [Alias("p")][string]${PW} = "test0000"
)
${UserName} = "Administrator"

& "${PSScriptRoot}\lib\check_admin.ps1"

Enable-LocalUser -Name ${UserName}

# Set password.
$Password = ConvertTo-SecureString ${PW} -AsPlainText -Force
Set-LocalUser -Name ${UserName} -Password $Password

# Enable automatic login.
Invoke-Expression "${PSScriptRoot}\lib\auto_login.ps1 -u ${UserName} ${PW}"
