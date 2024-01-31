# Remote Testing Helper Scripts

This directory contains scripts that will reduce the setup steps in [README.md](..%2FREADME.md) file for the server and
the DUT (Windows and Chrome). The
aim of these steps is to reduce the manual effort and to avoid the mistakes that could be generated due to the manual
effort.

You need to follow the order in the [README.md](..%2FREADME.md) file.

## Server Setup

### Work Flow

This section will describe the script inside ```server_setup``` directory that will do the following:

1. Create ```.ssh``` directory if it doesn't exist.
2. Copy the testing SSH keys from ```chromiumos/chromite/ssh_keys``` that is used by tast to ```.ssh``` directory.
3. Set permissions for the private key file.
4. Add the private testing key to SSH agent.

### Usage

1. Move the script to ```/home/user_name```.
2. Double check that you have the chromium project in ```/home/user_name``` it will be required to copy the testing keys
   from inside it.
3. Open terminal and set executable permissions to the script:
   ```
   chmod +x server_setup.sh
   ```
4. From the terminal run the server_setup.sh bash script:
   ```
   ./server_setup.sh
   ```

## Windows DUT Setup

### Work Flow

This section will describe the two scripts inside ```win_dut_setup``` directory that is responsible for installing
OpenSSHServer feature and do the required steps to make the SSH use public key instead of the password login, this can
be done as below:

<b> Note:</b>The below steps should be executed for a user in the Administrators group.

1. Install OpenSSHServer feature.
2. Change OpenSSHServer settings to be run Automatically and start it.
3. Move the public key to ```C:\ProgramData\ssh``` and rename it to <b>administrators_authorized_keys</b>.
4. Change the permission to the administrators_authorized_keys file, using the following command:
   ```
   icacls administrators_authorized_keys /inheritance:r /grant "Administrators:F" /grant "SYSTEM:F"
   ```
5. Create ```.ssh``` directory if it doesn't exist.
6. Copy ```administrators_authorized_keys``` file to ```username/.ssh```.
7. Update the ```C:\ProgramData\ssh\sshd_config``` file by enabling public key authentication.
8. Restart OpenSSHServer.


- The first Step (Install OpenSSHServer feature) will be done by ```win_ssh_server_install.ps1```.
- The other steps will be done by ```win_remote_conn_setup.ps1```.

### Usage

1. Move the scripts and the public key to ```C:\Users\username ```
2. Open powerShell terminal as Admin, you can follow the below:
   ```
    win + x -> Terminal (Admin) or Windows PowerSell (Admin).
   ```
3. Enable the terminal to execute powerShell scripts by updating the execution policy, using the below command the
   current terminal will have the ability to run powerShell scripts:
   ```
   Set-ExecutionPolicy -Scope Process Bypass
   ```
4. Run ```win_ssh_server_install.ps1``` script as below:
   ```
   .\win_ssh_server_install.ps1
   ```
5. Restart the device.
6. Repeat the <b>second and third steps</b> then run ```win_remote_conn_setup.ps1``` as below:
   ```
   .\win_remote_conn_setup.ps1
   ```

## Chrome DUT Setup

### Work Flow

This section will describe the script inside ```chrome_dut_setup``` directory that is responsible for preparing the
chrome DUT to accept ssh connection wing a public key used by the server and the DUT devices. where the script will do
the following:

1. Create .ssh directory if it doesn't exist.
2. Move the public key to ```.ssh``` directory.
3. Rename the public key to ```authorized_keys```.

### Usage

1. Move the public key and the script to ```/home/chronos/user```.
2. Login to chronos as an admin by click on ```ctrl + alt + F2``` and sign in with <b> chronos </b>.
3. Change the permissions of the script:
   ```
    chmod +x chrome_remote_conn_setup.sh 
   ```
4. Copy the script to <b> /bin </b> directory to have the required access to be run, for this do the following:
   ```
    sudo mv chrome_remote_conn_setup.sh ../../../bin
   ```
5. Run the script from ```/bin```:
   ```
    sudo ./chrome_remote_conn_setup.sh
   ```