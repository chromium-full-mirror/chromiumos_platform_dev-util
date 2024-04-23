# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
param (
    [Parameter(Position = 0)][string[]]${SourcePath}
)

function Write-Log {
    param(
        [string]${Message}
    )
    Write-Output "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') ${Message}"
}

function Write-Error-Log {
    param(
        [string]${Message}
    )
    Write-Error "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') ${Message}"
}

function Install-Poetry {
    # Check and install poetry if not installed yet.
    python -m poetry
    if ($?) {
        return
    }
    Write-Log "Start to install poetry."
    python -m pip install --user poetry
    if ($?) {
        return
    }
    Write-Error-Log "Failed to install poetry"
    exit 3
}

function New-Venv {
    Write-Log "Deploying the DUT virtual envrionment on $PWD"

    Install-Poetry

    # Workaroud for the keyring
    # https://stackoverflow.com/questions/74392324/poetry-install-throws-winerror-1312-when-running-over-ssh-on-windows-10 
    $env:PYTHON_KEYRING_BACKEND="keyring.backends.null.Keyring"

    # Install or upgrade all dependencies.
    # It will create the virtual environment if it has not been created yet.
    python -m poetry lock --no-update
    if ($?) {
        python -m poetry install
        if ($?) {
            Write-Log "The DUT virtual environment deployment has completed."
            return
        }
    }
    Write-Error-Log "Failed to deploy DUT virtual environment!"
    exit 4

}

Push-Location "${SourcePath}"
try {
    New-Venv
} catch {
    Write-Error-Log $_
    exit 2
} finally {
    Pop-Location
}
