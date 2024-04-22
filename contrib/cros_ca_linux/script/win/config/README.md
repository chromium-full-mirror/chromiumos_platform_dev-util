CrOS CA Read Me For Windows Configuration
===

The Windows Device Under Test (DUT) runs with the local `Administrator` user. It needs to configure and install some required applications for testing to work.

# Prerequisites

- Windows 10 or Windows 11.

- Put this `config` folder to the DUT.

# Allow Script Execution in Windows PowerShell

1. Right-click the PowerShell icon and choose Run as Administrator to open the PowerShell with administrator privileges.

2. Modify the script execution policy to allow script execution in Windows PowerShell.
```
Set-ExecutionPolicy -ExecutionPolicy Unrestricted -Scope LocalMachine -Force
```
3. Close the PowerShell window.

# Enable the Administrator User and Log In with It

1. Right-click the PowerShell icon and choose Run as Administrator to open the PowerShell with administrator privileges.

2. Change the directory to the folder of the config scripts `config`

3. Run the script to enable local Administrator account with automatic login.
```
.\admin_user_enable.ps1
```

4. Restart the DUT, and it will automatically log in with the local Administrator user.

5. If it's the first time logging in to the Administrator account, choose "No" for all privacy settings.

# OpenSSH Installation and Setup

After the installation, the OpenSSH server will start automatically after logging into Windows.

1. Lanuch a PowerShell window.

2. Change the directory to the folder of the config scripts `config`.

3. Run the script to install OpenSSH Server
```
.\sshd_install.ps1
```

4. Uninstall or disable firewalls other than Windows Defender Firewall if any.

5. Restart Windows to start sshd server.

# Chrome Browser Installation

1. Download [Google Chrome](https://www.google.com/chrome/).

2. Run the downloaded installer ChromeSetup.exe to install the Chrome browser.

# Python Installation

1. Download [Python 3.8.10](https://www.python.org/ftp/python/3.8.10/python-3.8.10-amd64.exe).

2. Run the Python installer.

3. Check the "Add Python 3.8 to PATH" and click "Install Now".

When the installation is successful, at the the bottom of the Setup dialog:

4. Click "Disable path length limit".

5. Click "Close" to complete the installation.

If you did not click `Disable path length limit` during the installation, you can use the following steps to disable the path length limit as well.

1. Press Windows key + R and input `regedit` to open Regedit.

2. Navigate to `Computer\HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\FileSystem`
(you can past this into the address bar in Regedit).

3. Set `LongPathsEnabled` to `1`.

# WinAppDriver Installation

1. Download WinAppDriver [WindowsApplicationDriver_1.2.1.msi](https://github.com/microsoft/WinAppDriver/releases/download/v1.2.1/WindowsApplicationDriver_1.2.1.msi).

2. Run the msi installation file and follow the UI to complete the installation.
