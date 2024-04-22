# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

#TODO: To be deleted after all DUTs are converted to default SSHD config.

param (
    [Alias("r")][switch]${Revoke},
    [Alias("u")][string]${UserName} = "${env:USERNAME}",
    [Parameter(Position = 0)][string]${Right}
)

& "${PSScriptRoot}\check_admin.ps1"

try {

    secedit /export /cfg ".\export.inf"

    # Get the user list for the right
    ${RightUsers} = $(Get-Content ".\export.inf" | Findstr "${Right}")

    # Remove the user before adding to prevent duplicates.
    ${RightUsers} = ${RightUsers} -replace ",${UserName}","" -replace "${UserName},", ""

    # Add the user right
    if (-not ${Revoke}) {
        ${RightUsers} += ",${UserName}"
    }
    Write-Output "New RightUsers=${RightUsers}."

    # Crete the tempoarary user rights files for updating.
    @"
[Unicode]
Unicode=yes
[Privilege Rights]
${RightUsers}
[Version]
signature="`$CHICAGO$"
Revision=1
"@ | Set-Content -Encoding unicode -Path ".\import.inf"

    # Update the user rights
    secedit /configure /db ".\rights.db" /cfg ".\import.inf" /areas user_rights
} finally {
    # Remove temporary files
    Remove-Item ".\export.inf",".\import.inf",".\rights.db",".\rights.jfm"
}
