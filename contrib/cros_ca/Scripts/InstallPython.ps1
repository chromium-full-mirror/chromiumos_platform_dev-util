# PowerShell Script to Download and Install Python 3.12.0
# This script automates the process of downloading and installing Python 3.12.0.

# Check if Python 3.12.0 is already installed
$installedVersion = & python --version 2>&1
if ($installedVersion -match "Python 3.12.0") {
  Write-Host "Python 3.12.0 is already installed. Exiting..."
  exit
}

# Parse Command-Line Arguments
$installForAllUsers = $false
if ($args -contains '--all-users') {
  $installForAllUsers = $true
}

# Check for Admin Privileges
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")
if (!$isAdmin) {
Write-Host "Installing Python requires administrative privileges. Please run the script as an administrator."
exit
}


# Setup variables
$pythonUrl = "https://www.python.org/ftp/python/3.12.0/python-3.12.0-amd64.exe"
$pythonInstaller = "$($env:TEMP)\python.exe"

# Download Python installer
Write-Host "Downloading Python installer..."
try {
  Invoke-WebRequest -Uri $pythonUrl -OutFile $pythonInstaller
} catch {
  Write-Host "Failed to download Python installer. Please check your internet connection and try again."
  exit 1
}

# Install Python
Write-Host "Installing Python..."
try {
  $installAllUsersArg = if ($installForAllUsers) { "1" } else { "0" }
  Start-Process -FilePath $pythonInstaller -ArgumentList "/quiet InstallAllUsers=$installAllUsersArg PrependPath=1" -Wait
} catch {
  Write-Host "Python installation failed:" $_
  exit 1
}

# Post-installation cleanup
Write-Host "Cleaning up downloaded files..."
Remove-Item -Path $pythonInstaller -Force

Write-Host "Python installation completed!"