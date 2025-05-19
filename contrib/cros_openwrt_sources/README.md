# How to Build a Test OpenWrt OS Image from Sources

OpenWrt images would be built in their entirety from source.

---

## Architecture Overview

To build your image for a new router, you should follow a specific file structure, since all routers share one Dockerfile. This makes the whole pipeline easy to modify and allows you to add custom features for each router.

---

## Adding a New Router

Create a folder in `./image_generator/configs`.
This folder should have `run.sh` in it that would be executed inside a Docker container that already has OpenWrt downloaded with all dependencies installed.

`run.sh` should execute all commands that you would do manually when working inside the OpenWrt folder, including preparing the OpenWrt folder for the build and running the necessary make commands.

**Reference:** Take a look into `./image_generator/configs/U6Plus` for an example.

---

## Project Folder Structure

* **`./build_dir`**: This directory is generated for each router model and contains `/bin` and `/log` folders mounted to the Docker's `openwrt/bin` and `openwrt/log` folders respectively.
* **`./image_generator/configs`**: This folder contains custom folders for each router model.
    * **`./${DEVICE}/`**: This folder should have a `run.sh` file, which is the entrypoint for Docker to run. This script should execute all required make commands and also prepare the OpenWrt folder for the build.
* **`./image_generator/custom_packages`**: Contains all packages that will be copied into the `openwrt/package` folder.
* **`./image_generator/custom_files`**: Contains all files that will be copied into the `openwrt/files` folder and later included in the OpenWrt generated image. These files are shared across all router models. If you want to add specific files, you can do that in `./run.sh` for a concrete router model.
* **`./scripts`**: This folder contains per-device `.sh` files that should start executing image generation. As a minimal requirement, they should set an `OPENWRT_COMMIT` that we should use for image generation. For BPi specifically, we have `OPENWRT_MTK_COMMIT` that describes the MTK repo commit to use. Ignore that if it's not required for your router.

---

## Generating an Image

It is a straightforward process; you just need to run the `./${DEVICE}.sh` file. This will trigger a Docker run that generates images and puts them inside the `./build_dir/${DEVICE}-${OPENWRT_COMMIT}/bin` folder. Then you can copy the required image from that folder.

### Build Summary

The built image includes a summary of the build, which can be referenced on the device it is installed on at `/`.

---

## Installing the Custom Image

Custom-built images are installed the same way as normal OpenWrt images ([official docs](https://openwrt.org/docs/guide-quick-start/factory_installation)).

### On a Device Not Yet Running OpenWrt

Follow the instructions on the device's info page on the OpenWrt wiki. Just be sure to use the corresponding custom image (should have "sdcard" in its name) binary instead of one downloaded from the OpenWrt image repository.

### On a Device Already Running OpenWrt

Follow the instructions on the device's info page on the OpenWrt wiki, but in general, it should be the following steps:

1.  Use `scp` to copy the custom image (should have the `.bin` or `.itb` file extension) to the router's `/tmp` directory:

    ```text
    $ scp <path_to_image.bin> <host>:/tmp
    ```

    If you get the `ash: /usr/libexec/sftp-server: not found` error when using scp, add the `-O` flag (newer OpenSSH versions default to use SFTP which OpenWrt does not support).

    ```text
    $ scp -O <path_to_image.bin> <host>:/tmp
    ```

2.  Run `sysupgrade /tmp/your_custom_image.bin`

3.  Wait a few minutes for the image to be installed and for the device to reboot.

4.  If the device is not connected to the Internet, it will reboot 5 times, because `z_cros_test` daemon tests internet connection and reboots if it can't connect to google.com.

5.  Upon reconnecting to the device, you can check the build info of the installed image at `/etc/cros/cros_openwrt_image_build_info.json`.

### Example cros_openwrt_image_build_info.json
```json
{
  "imageUuid": "4533ddf0-f255-4644-a17f-ab022b7f7ca5",
  "customImageName": "cros-1.1.0-upreved-hostapd-v2.11-devel",
  "osRelease": {
    "version": "21.02.5",
    "buildId": "r16688-fa9a932fdb",
    "openwrtBoard": "ramips/mt7621",
    "openwrtArch": "mipsel_24kc",
    "openwrtRelease": "OpenWrt 21.02.5 r16688-fa9a932fdb"
  },
  "standardBuildConfig": {
    "openwrtRevision": "r16688-fa9a932fdb",
    "openwrtBuildTarget": "ramips/mt7621",
    "buildProfile": "ubnt_unifi-6-lite",
    "deviceName": "Ubiquiti UniFi 6 Lite",
    "buildTargetPackages": [
      "base-files",
      "ca-bundle",
      "dropbear",
      "fstools",
      "libc",
      "libgcc",
      "libustream-wolfssl",
      "logd",
      "mtd",
      "netifd",
      "opkg",
      "uci",
      "uclient-fetch",
      "urandom-seed",
      "urngd",
      "busybox",
      "procd",
      "kmod-leds-gpio",
      "kmod-gpio-button-hotplug",
      "wpad-basic-wolfssl",
      "dnsmasq",
      "firewall",
      "ip6tables",
      "iptables",
      "kmod-ipt-offload",
      "odhcp6c",
      "odhcpd-ipv6only",
      "ppp",
      "ppp-mod-pppoe"
    ],
    "profilePackages": [
      "kmod-mt7603",
      "kmod-mt7915e"
    ],
    "supportedDevices": [
      "ubnt,unifi-6-lite"
    ]
  },
  "routerFeatures": [
    "WIFI_ROUTER_FEATURE_IEEE_802_11_A",
    "WIFI_ROUTER_FEATURE_IEEE_802_11_B",
    "WIFI_ROUTER_FEATURE_IEEE_802_11_G",
    "WIFI_ROUTER_FEATURE_IEEE_802_11_N",
    "WIFI_ROUTER_FEATURE_IEEE_802_11_AC",
    "WIFI_ROUTER_FEATURE_IEEE_802_11_AC",
    "WIFI_ROUTER_FEATURE_IEEE_802_11_AX"
  ],
  "buildTime": "2023-05-18T04:26:41.547679944Z",
  "crosOpenwrtImageBuilderVersion": "1.1.0",
  "customIncludedFiles": {
    "etc/dropbear/authorized_keys": "408f2b0c95706cbf38aa44469204ef87221a9beeb9fff2901b93cbebabf62b2c",
    "etc/init.d/z_cros_test.sh": "460e8689559c44731c078615f4dd2dd6c594310a29c5cfeca88c5448a340fd36",
    "etc/uci-defaults/99_cros_customizations.sh": "70b49dcd8d73bad2f273545e1a95528c99bb8850dfe794551028c6a9184e9854"
  },
  "customPackages": {
    "Packages": "1e567b34ecbef28a28e9e6187273dbb74fc91dff0cb9c4e3ae08caa1c0b7a90d",
    "Packages.gz": "a003b67e975c918ad0b02807a2bb4d0b1f65462058340c5ae72d2b205af4e573",
    "Packages.sig": "40c590ca2efcb36bf6814d41a30a3c97f97203bab1599034916f6e6ebbdd9c4c",
    "cros-send-management-frame_1.0.0-1_mipsel_24kc.ipk": "3af053f501d09832f6145f5fd17b37582c5838ed3be6f866a412addf60e6a489",
    "hostapd-common_2022-07-29-b704dc72-1.1_mipsel_24kc.ipk": "adb2ffb3d9e02e933933f25831237d93323d67d6037b6caf5f00063a37a1f8eb",
    "hostapd-utils_2020-06-08-5a8b3662-41_mipsel_24kc.ipk": "eb9270f29f920d20de703f96b904615dec62d16637a11d575e039babfc105375",
    "wpa-cli_2020-06-08-5a8b3662-41_mipsel_24kc.ipk": "2197f8be921c117f1d0fdb65c0b5d86e53a37ab01605c7a89cca44ad355cdd1f",
    "wpad-openssl_2022-07-29-b704dc72-1.1_mipsel_24kc.ipk": "ca56c9bb773d1a57a1466ef5bfb073448631ded1976f6370788d7cf2e227ca8a"
  },
  "extraIncludedPackages": [
    "cros-send-management-frame",
    "hostapd-common",
    "hostapd-utils",
    "wpad-openssl",
    "wpa-cli",
    "iputils-ping",
    "iputils-arping",
    "kmod-veth",
    "tcpdump",
    "procps-ng-pkill",
    "netperf",
    "iperf",
    "sudo",
    "python3-email",
    "python3-idna",
    "python3-light",
    "python3-urllib"
  ],
  "excludedPackages": [
    "hostapd",
    "hostapd-basic",
    "hostapd-basic-openssl",
    "hostapd-basic-wolfssl",
    "hostapd-mini",
    "hostapd-openssl",
    "hostapd-wolfssl",
    "wpad",
    "wpad-mesh-openssl",
    "wpad-mesh-wolfssl",
    "wpad-basic",
    "wpad-basic-openssl",
    "wpad-basic-wolfssl",
    "wpad-mini",
    "wpad-wolfssl",
    "wpa-supplicant",
    "wpa-supplicant-mesh-openssl",
    "wpa-supplicant-mesh-wolfssl",
    "wpa-supplicant-basic",
    "wpa-supplicant-mini",
    "wpa-supplicant-openssl",
    "wpa-supplicant-p2p",
    "eapol-test",
    "eapol-test-openssl",
    "eapol-test-wolfssl"
  ],
  "disabledServices": [
    "wpad",
    "dnsmasq"
  ]
}
```

## Accessing the Router After Custom OpenWrt Image Installation

The CROS customizations disable the router's DHCP server and do not turn on any wireless networks. Instead, the device is configured to act as a DHCP client, allowing it to be accessed via SSH through other physically connected networks (such as a lab network).

As long as you know the IP address of the router and can connect to the network the router is connected to, you can SSH into the router as the `root` user using the regular cros [testing_rsa](../../../../../chromeos-admin/puppet/modules/profiles/files/user-common/ssh/testing_rsa) private key. This is the same way DUTs and Gale routers are accessed.

If you do not know the IP address of the router, you will need to obtain it by checking the network the router is connected to and identifying the router using its MAC address. It is recommended to configure the parent network to statically assign the router a consistent IP based on its MAC address for long-term usage.