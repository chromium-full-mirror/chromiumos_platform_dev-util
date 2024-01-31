echo "Install OpenSSH Server feature"
Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0

Write-Output "Please restart Windows to run sshd!"
