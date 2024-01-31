# PowerShell Script to Download and Install WindowsApplicationDriver_1.2.1
# This script automates the process of downloading and installing WindowsApplicationDriver_1.2.1.

# Check if WindowsApplicationDriver is already installed
$defaultPath = "C:\Program Files (x86)\Windows Application Driver\WinAppDriver.exe"
If ((Test-Path $defaultPath)) {
  Write-Host "WinAppDriver is already installed. Exiting..."
  exit
}

# Check for Admin Privileges
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")
if (!$isAdmin) {
Write-Host "Installing WinAppDriver requires administrative privileges. Please run the script as an administrator."
exit
}


# Setup variables
$winAppDriverURL = "https://github.com/microsoft/WinAppDriver/releases/download/v1.2.1/WindowsApplicationDriver_1.2.1.msi"
$winAppDriverInstaller = "$($env:TEMP)\WindowsApplicationDriver.msi"

# Download WinAppDriver installer
Write-Host "Downloading WinAppDriver installer..."
try {
  Invoke-WebRequest -Uri $winAppDriverURL -OutFile $winAppDriverInstaller
} catch {
  Write-Host "Failed to download WinAppDriver installer. Please check your internet connection and try again."
  exit 1
}

# Install WinAppDriver
Write-Host "Installing WinAppDriver..."
try {
  Start-Process msiexec "/i $winAppDriverInstaller /norestart /qn" -Wait;
} catch {
  Write-Host "WinAppDriver installation failed:" $_
  exit 1
}

# Post-installation cleanup
Write-Host "Cleaning up downloaded files..."
Remove-Item -Path $winAppDriverInstaller -Force

Write-Host "WinAppDriver installation completed!"

# Enable-Developer-Mode
Write-Host "Enabling developer mode..."
reg add "HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\AppModelUnlock" /t REG_DWORD /f /v "AllowDevelopmentWithoutDevLicense" /d "1"
Write-Host "Enabling of developer mode completed!"