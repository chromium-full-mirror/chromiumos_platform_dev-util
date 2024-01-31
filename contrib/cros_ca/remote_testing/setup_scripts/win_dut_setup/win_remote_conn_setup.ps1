echo "Set OpenSSH Server service to start automatically"
Set-Service -Name sshd -StartupType Automatic
echo "Start the OpenSSH Server service"
Start-Service -Name sshd

# Move testing_rsa.pub to .ssh folder and rename it to administrators_authorized_keys
Move-Item -Path "testing_rsa.pub" -Destination "C:\ProgramData\ssh\administrators_authorized_keys" -Force
Write-Output "Moved testing_rsa.pub to C:\ProgramData\ssh folder and renamed it to administrators_authorized_keys"

# Change administrators_authorized_keys permissions
$command = "icacls C:\ProgramData\ssh\administrators_authorized_keys /inheritance:r /grant `"Administrators:F`" /grant `"SYSTEM:F`""
Invoke-Expression $command

# Check if .ssh directory exists, create it if not
if (-not (Test-Path -Path ".ssh" -PathType Container)) {
    New-Item -ItemType Directory -Name ".ssh"
    Write-Output "Created .ssh directory"
}

Copy-Item -Path "C:\ProgramData\ssh\administrators_authorized_keys" -Destination ".ssh" -Force
Write-Output "Copy administrators_authorized_keys to .ssh folder"

echo "Update sshd_config file"
$sshdConfigPath = "C:\ProgramData\ssh\sshd_config"
$sshdConfigContent = Get-Content -Path $sshdConfigPath
$pubkeyAuthenticationUpdated = $false
# Iterate through each line in the content
foreach ($index in 0..($sshdConfigContent.Length - 1)) {
    $line = $sshdConfigContent[$index]
    if ($line -match "^\s*#?\s*PubkeyAuthentication(?:\s+no|\s+yes|\s)?$") {
        $sshdConfigContent[$index] = "PubkeyAuthentication yes"
        $pubkeyAuthenticationUpdated = $true
        break
    }
}

# Check if the PubkeyAuthentication line was found and updated
if ($pubkeyAuthenticationUpdated) {
    # Write the modified content back to the sshd_config file
    $sshdConfigContent | Set-Content -Path $sshdConfigPath -Force

    Write-Host "PubkeyAuthentication setting updated successfully."
} else {
    Write-Host "PubkeyAuthentication setting not found in the sshd_config file."
}

echo "Restart the OpenSSH Server service"
Restart-Service -Name sshd
