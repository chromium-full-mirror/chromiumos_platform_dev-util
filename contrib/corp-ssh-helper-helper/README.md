# corp-ssh-helper-helper

corp-ssh-helper-helper is a helper tool to interact with DUTs (DUT stands for Device under test) from inside the ChromiumOS SDK chroot where corp-ssh-helper is not available.

## Prerequisites

If you have installed a similar tool (e.g. [go/old-corp-ssh-helper-helper](http://go/old-corp-ssh-helper-helper)), please remember to uninstall it before using this.

Namely, there should be nothing existing at `/usr/bin/corp-ssh-helper` in your ChromiumOS SDK chroot.
Also `src/scripts/.local_mounts` shouldn't contain `/usr/bin/corp-ssh-helper`.

```shell
(inside the chroot) $ ls /usr/bin/corp-ssh-helper  # This should fail
(inside the chroot) $ grep /usr/bin/corp-ssh-helper ~/chromiumos/src/scripts/.local_mounts # This shouldn't match with any line
```

## Install

To install corp-ssh-helper-helper, mount the corp-ssh-helper-helper-client.py file as /usr/bin/corp-ssh-helper inside the ChromiumOS SDK chroot.

To do this, run the following command **outside** the chroot.

```shell
(outside the chroot) $ TOP=$HOME/chromiumos  # Full path to the Chromium OS top directory
(outside the chroot) $ echo $TOP/src/platform/dev/contrib/corp-ssh-helper-helper/corp-ssh-helper-helper-client.py /usr/bin/corp-ssh-helper >> $TOP/src/scripts/.local_mounts
```

## Set up ~/.ssh/config

If you are lucky, you don't have to do anything, as `cros_sdk` copies ~/.ssh/config outside the chroot when entering the chroot, and your config file may contain the required ProxyCommand setup. Otherwise, you need to create ~/.ssh/config inside the chroot, and the following is to use the helper for $DUT (replace $DUT with an IP address or a host name).

You may also need to copy your `testing_rsa` or `partner_testing_rsa` keys from outside of the chroot to inside of the chroot, in the ~/.ssh directory. More information on these keys is available at http://go/chromeos-lab-duts-ssh#setup-private-key-and-ssh-config.

To ensure that this config is persistent every time you enter the chroot, be sure to add this in the ssh-config OUTSIDE the chroot, then re-enter the chroot to confirm that the ssh config in your chroot updated properly.
```
Host $DUT
  ProxyCommand corp-ssh-helper %h %p
```

See http://go/arc-corp-ssh-helper-notes and http://go/chromeos-lab-duts-ssh to learn more.

## How to use

### Run the server outside the chroot

To use corp-ssh-helper-helper, you need to start the server **outside** the chroot.

```shell
(outside the chroot) ~/chromiumos$ src/platform/dev/contrib/corp-ssh-helper-helper/corp-ssh-helper-helper-server.py
```

### Use SSH

After starting the server, you can use SSH in the same way as you do outside the chroot. The client script will automatically forward the connection request to the server running outside the chroot.

```shell
(inside the chroot) $ ssh $DUT
```

The same goes for tools like `tast run` and `cros deploy`.

## Uninstall

You can uninstall this tool by deleting ~/chromiumos/.cshh.sock and deleting the corp-ssh-helper mount point from .local_mounts  **inside** the chroot.

```shell
(inside the chroot) $ rm ~/chromiumos/.cshh.sock
(inside the chroot) $ vi ~/chromiumos/src/scripts/.local_mounts  # To remove the corp-ssh-helper mount point
```
