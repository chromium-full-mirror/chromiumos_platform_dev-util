# How to build a test OpenWrt OS image with `cros_openwrt_image_builder`

The `cros_openwrt_image_builder` CLI utility can be used to build test OpenWrt
OS images. It makes use of official OpenWrt sdks to compile custom OpenWrt
packages and then uses official OpenWrt image builders to build the image with
these custom packages and other image modification settings.

OpenWrt images can be built in their entirety from source, but using the
combined approach of the sdk and the image builder ensures that our
built images include only the minimal amount of customizations necessary by only
overriding the specific packages we define. Plus, it's faster.

The `cros_openwrt_image_builder` is designed to allow for building test OpenWrt
OS images for any device already supported by OpenWrt, with minimal effort for
switching between devices and/or OpenWrt versions/snapshots.

This is not guaranteed to work with every OpenWrt version and device, but it
should at least be able to attempt to build the packages and images given
that the provided target has an accessible official sdk and image builder.

Tested with the following devices and OpenWrt versions:

* [Ubiquiti - UniFi 6 Lite](https://openwrt.org/toh/hwdata/ubiquiti/ubiquiti_unifi_6_lite)
    * [21.02.5](https://downloads.openwrt.org/releases/21.02.5/targets/ramips/mt7621/)
    * [22.03.2](https://downloads.openwrt.org/releases/22.03.2/targets/ramips/mt7621/)

## Install system dependencies

See https://openwrt.org/docs/guide-user/additional-software/imagebuilder#prerequisites
for instructions.

You will also need ~3GB of hard drive space for the build files.

## Build & Install cros_openwrt_image_builder

Use the `./install.sh` bash script to build and install
cros_openwrt_image_builder so that it may be run as a regular command,
`cros_openwrt_image_builder`. The built copy will reside at
`./bin/cros_openwrt_image_builder`.

Note: The installation script is meant to be run on your main system, not in a
chroot.

Note: Syncing an updated version of this source repository will not
automatically rebuild an updated version of `cros_openwrt_image_builder`. To
update your local build of `cros_openwrt_image_builder`, simply re-run
`./install.sh` (or `./build.sh`) to rebuild it.

```text
$ $ bash ./install.sh help
Usage: install.sh [options]

Options:
 --dir|-d <path>    Path to directory where cros_openwrt_image_builder is to be
                    installed, which should be in your $PATH
                    (default = '~/lib/depot_tools').
```

## Build the default custom image from a fresh install

1. Find your router in
   the [list of supported devices](https://openwrt.org/supported_devices)
   ([table lookup](https://openwrt.org/toh/start)).

2. Open the data page associated with your router from the supported devices
   table (e.g. https://openwrt.org/toh/hwdata/ubiquiti/ubiquiti_unifi_6_lite).

3. Make note of the "Firmware OpenWrt Upgrade URL" (
   e.g. `https://downloads.openwrt.org/releases/22.03.2/targets/ramips/mt7621/openwrt-22.03.2-ramips-mt7621-ubnt_unifi-6-lite-squashfs-sysupgrade.bin
   `).

4. Run `cros_openwrt_image_builder build all` and specify `--auto_url` as the
   "Firmware OpenWrt Upgrade URL":

```text
$ cros_openwrt_image_builder build all --auto_url <Firmware OpenWrt Upgrade URL>
```

Note: This assumes your chroot is installed at `~/chromiumos`. If it isn't, you
need to supply the path to this source directory (`cros_openwrt`) with the
`--src_dir <path_to_cros_openwrt>` flag.

5. When prompted, confirm that the resolved sdk and image builder archive URLs
   are for your desired build target, which was parsed from the `--auto_url`
   param.

6. Wait for the sdk to be downloaded, extracted, and used to build all the
   custom packages and their dependencies. This will take a long time (~1hr) the
   first time this is run for the sdk. Note: In future runs for the same build
   target you can specify the `--use_existing_sdk` flag to not start from
   scratch,
   which is much faster.

7. Once the sdk finishes building the custom packages, the official image
   builder is downloaded, extracted, and used to build the custom OpenWrt OS
   image.
   When prompted, choose the correct build profile based on your target device.
   You
   can find it in your devices "Firmware OpenWrt Upgrade URL" as well (e.g.
   `ubnt_unifi-6-lite`). You can skip this prompt in future runs if you know the
   profile already with the `--image_profile <profile>` flag.

8. Wait for the image builder to build the image (<5m). This will download any
   needed official precompiled packages. The custom packages built by the local sdk
   are
   included as well, and override any official ones. Any dependency of a custom
   package is downloaded if it is not explicitly included (i.e. we only use the
   customized packages and not its dependencies which the sdk also
   builds).

9. Open the path to the image directory displayed. For convenience, the
   directory includes all related image files for local use as well as
   a timestamped *.tar.xz archive of these files for easy distribution.

If you want to test a different OpenWrt OS version, simply provide an
`--auto_url` for a different version and repeat the steps starting from step 4.

By default, the working directory (`/tmp/cros_openwrt/` by default) is not
deleted after the command is finished so that it may be used as a reference. To
delete just the intermediary build files, you can run
`cros_openwrt_image_builder build clean`. To delete the whole
working directory, including copies of previously built images, you can run
`cros_openwrt_image_builder build clean --all`.

Example call for building an image with OpenWrt version `21.02.5` for a
`Ubiquiti - UniFi 6 Lite` router:

```text
$ cros_openwrt_image_builder build --auto_url=https://downloads.openwrt.org/releases/21.02.5/targets/ramips/mt7621/ --image_profile ubnt_unifi-6-lite
```

### Build summary

The built image includes a summary of the build which can be referenced on
the device it is installed on at `/`

## Install the custom image

Custom-built images are installed the same way as normal OpenWrt images
([offical docs](https://openwrt.org/docs/guide-quick-start/factory_installation)).

### On a device not yet running OpenWrt

Follow the instructions on the device's info page on the OpenWrt wiki. Just be
sure to use the corresponding custom image binary instead of one downloaded
from the OpenWrt image repository.

### On a device already running OpenWrt

Follow the instructions on the device's info page on the OpenWrt wiki, but in
general it should be the following steps:

1. Use `scp` to copy the custom image (should have the `.bin` file extension) to
   the router's `/tmp` directory:

```text
$ scp <path_to_image.bin> <host>:/tmp
```

If you get the `ash: /usr/libexec/sftp-server: not found` error when using scp,
add the `-O` flag (newer OpenSSH versions default to use SFTP which OpenWrt
does not support).

```text
$ scp -O <path_to_image.bin> <host>:/tmp
```

3. Run `sysupgrade /tmp/your_custom_image.bin`

4. Wait a few minutes for the image to be installed and for the device to
   reboot.

5. Upon reconnecting to the device, you can check the build info of the
   installed image at `/etc/cros/cros_openwrt_image_build_info.json`

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

## Accessing the router after installing the custom OpenWrt image

The CROS customizations will disable the router's DHCP server and not turn on
any wireless networks, but will configure the device to connect to act as a DHCP
client and thus allow it to be accessed via ssh through other networks it is
physically connected to (such as a lab network).

As long as you know the IP address of the router and can connect to the network
the router is connected to, you can ssh into the router as the `root` user and
the regular
cros [testing_rsa](../../../../../chromeos-admin/puppet/modules/profiles/files/user-common/ssh/testing_rsa)
private key. This is the same way DUTs and Gale routers are accessed.

If you do not know the IP address of the router, you will need to obtain it by
checking the network the router is connected to and identify the router using
its MAC address. It is recommended configure the parent network to statically
assign the router a consistent IP based on its MAC address for long-term usage.

## `cros_openwrt_image_builder`

```text
$ cros_openwrt_image_builder --help
Utility for building custom OpenWrt OS images with custom compiled packages

Usage:
  cros_openwrt_image_builder [command]

Available Commands:
  build       Commands for building custom OpenWrt images.
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  release     Commands for managing, releasing, and retrieving released custom OpenWrt images

Flags:
  -h, --help      help for cros_openwrt_image_builder
  -v, --version   version for cros_openwrt_image_builder

Use "cros_openwrt_image_builder [command] --help" for more information about a command.
```

### `build`

```text
cros_openwrt_image_builder build --help
Commands for building custom OpenWrt images.

Usage:
  cros_openwrt_image_builder build [command]

Available Commands:
  all         Compiles custom OpenWrt packages and builds a custom OpenWrt image.
  clean       Deletes temporary files.
  image       Builds a custom OpenWrt image.
  packages    Compiles custom OpenWrt packages.

Flags:
      --auto_url string                        Download URL to use to auto resolve unset --sdk_url and --image_builder_url values from.
      --chromiumos_src_dir string              Path to local chromiumos source directory. (default "/usr/local/google/home/jaredbennett/chromiumos")
      --disable_auto_sdk_compile_retry         Include to disable the default behavior of retrying the compilation of custom packages once if the first attempt fails. (default true)
      --disable_service stringArray            Services to disable in the built image. (default [wpad,dnsmasq])
      --exclude_package stringArray            Packages to exclude from the built image. (default [hostapd,hostapd-basic,hostapd-basic-openssl,hostapd-basic-wolfssl,hostapd-mini,hostapd-openssl,hostapd-wolfssl,wpad,wpad-mesh-openssl,wpad-mesh-wolfssl,wpad-basic,wpad-basic-openssl,wpad-basic-wolfssl,wpad-mini,wpad-wolfssl,wpa-supplicant,wpa-supplicant-mesh-openssl,wpa-supplicant-mesh-wolfssl,wpa-supplicant-basic,wpa-supplicant-mini,wpa-supplicant-openssl,wpa-supplicant-p2p,eapol-test,eapol-test-openssl,eapol-test-wolfssl])
      --extra_image_name string                A custom name to add to the image, added as a suffix to existing names.
  -h, --help                                   help for build
      --image_builder_url string               URL to download the image builder archive from. Leave unset to use the last downloaded image builder.
      --image_feature stringArray              Wifi router features this image supports for testing (possible features: [WIFI_ROUTER_FEATURE_UNKNOWN, WIFI_ROUTER_FEATURE_INVALID, WIFI_ROUTER_FEATURE_IEEE_802_11_A, WIFI_ROUTER_FEATURE_IEEE_802_11_B, WIFI_ROUTER_FEATURE_IEEE_802_11_G, WIFI_ROUTER_FEATURE_IEEE_802_11_N, WIFI_ROUTER_FEATURE_IEEE_802_11_AC, WIFI_ROUTER_FEATURE_IEEE_802_11_AX, WIFI_ROUTER_FEATURE_IEEE_802_11_AX_E, WIFI_ROUTER_FEATURE_IEEE_802_11_BE])
      --image_profile string                   The profile to use with the image builder when making images. Leave unset to prompt for selection based off of available profiles.
      --include_custom_package stringArray     Names of packages that should be included in built images that are built using a local sdk and included in the image builder as custom packages. Only custom packages in this list are saved from sdk package compilation. (default [cros-send-management-frame,hostapd-common,hostapd-utils,wpad-openssl,wpa-cli])
      --include_official_package stringArray   Names of packages that should be included in built images that are downloaded from official OpenWrt repositories. (default [iputils-ping,iputils-arping,kmod-veth,tcpdump,procps-ng-pkill,netperf,iperf,sudo,python3-email,python3-idna,python3-light,python3-urllib])
      --sdk_compile_max_cpus int               The maximum number of CPUs to use for custom package compilation. Values less than 1 indicate that all available CPUs may be used. (default -1)
      --sdk_config stringToString              Config options to set for the sdk when compiling custom packages. (default [CONFIG_WPA_MBO_SUPPORT=y,CONFIG_WPA_ENABLE_WEP=y,CONFIG_DRIVER_11N_SUPPORT=y,CONFIG_DRIVER_11AC_SUPPORT=y,CONFIG_DRIVER_11AX_SUPPORT=y])
      --sdk_make stringArray                   The sdk package makefile paths to use to compile custom app. Making a package with the sdk will build all the package dependencies, but only need to be included if they are expected to differ from official versions. (default [cros-send-management-frame,feeds/base/hostapd])
      --sdk_url string                         URL to download the sdk archive from. Leave unset to use the last downloaded sdk.
      --use_existing                           Shortcut to set both --use_existing_sdk and --use_existing_image_builder.
      --use_existing_image_builder             Use image builder in working directory as-is (must exist).
      --use_existing_sdk                       Use sdk in working directory as-is (must exist).
      --working_dir string                     Path to working directory to store downloads, sdk, image builder, and built packages and images. (default "/tmp/cros_openwrt")

Use "cros_openwrt_image_builder build [command] --help" for more information about a command.
```

### `release`

```text
$ cros_openwrt_image_builder release --help
Commands for managing, releasing, and retrieving released custom OpenWrt images

Usage:
  cros_openwrt_image_builder release [command]

Available Commands:
  config      Commands for reading and updating the release config
  download    Downloads a released image
  upload      Uploads a local image to GCS and adds it to the release config

Flags:
      --bucket string             GCS storage bucket to use (both prod and test use the same bucket) (default "chromeos-connectivity-test-artifacts")
      --gcloud_cred_file string   The gcloud credential file to use with the GCS API (uses gcloud CLI application-default credentials when unset)
  -h, --help                      help for release
  -p, --prod                      Uses the production wifi router config when present, or the test config when not present

Use "cros_openwrt_image_builder release [command] --help" for more information about a command.
```

### `release config`

```text
$ cros_openwrt_image_builder release config --help
Commands for reading and updating the release config

Usage:
  cros_openwrt_image_builder release config [command]

Available Commands:
  get         Downloads the release config from storage
  set         Updates the release config in storage

Flags:
  -h, --help   help for config

Global Flags:
      --bucket string             GCS storage bucket to use (both prod and test use the same bucket) (default "chromeos-connectivity-test-artifacts")
      --gcloud_cred_file string   The gcloud credential file to use with the GCS API (uses gcloud CLI application-default credentials when unset)
  -p, --prod                      Uses the production wifi router config when present, or the test config when not present

Use "cros_openwrt_image_builder release config [command] --help" for more information about a command.
```

#### Example config file
```json
{
  "openwrt": {
    "Ubiquiti UniFi 6 Lite": {
      "buildTargetPackages": [],
      "currentImageUuid": "4533ddf0-f255-4644-a17f-ab022b7f7ca5",
      "nextImageUuid": "",
      "nextImageVerificationDutPool": [],
      "images": [
        {
          "imageUuid": "4533ddf0-f255-4644-a17f-ab022b7f7ca5",
          "archivePath": "gs://chromeos-connectivity-test-artifacts/wifi_router/openwrt_images/Ubiquiti_UniFi_6_Lite/4533ddf0-f255-4644-a17f-ab022b7f7ca5/openwrt-21.02.5-cros-1.1.0-upreved-hostapd-v2.11-devel-ramips-mt7621-ubnt_unifi-6-lite_20230517-212718.tar.xz",
          "minDutReleaseVersion": "0"
        }
      ]
    }
  }
}
```

### `release config set`

```text
$ cros_openwrt_image_builder release config set --help
Updates the release config in storage

Usage:
  cros_openwrt_image_builder release config set [flags]

Flags:
      --force        Bypass config validation failures and forcefully update the config
  -h, --help         help for set
      --src string   Path to read new config from (default "./wifi_router_config_openwrt.json")

Global Flags:
      --bucket string             GCS storage bucket to use (both prod and test use the same bucket) (default "chromeos-connectivity-test-artifacts")
      --gcloud_cred_file string   The gcloud credential file to use with the GCS API (uses gcloud CLI application-default credentials when unset)
  -p, --prod                      Uses the production wifi router config when present, or the test config when not present
```

### `release config get`

```text
 $ cros_openwrt_image_builder release config get --help
Downloads the release config from storage

Usage:
  cros_openwrt_image_builder release config get [flags]

Flags:
      --dst string   Path to save config to (default "./wifi_router_config_openwrt.json")
  -h, --help         help for get
      --print        Print the config rather than save it to a file

Global Flags:
      --bucket string             GCS storage bucket to use (both prod and test use the same bucket) (default "chromeos-connectivity-test-artifacts")
      --gcloud_cred_file string   The gcloud credential file to use with the GCS API (uses gcloud CLI application-default credentials when unset)
  -p, --prod                      Uses the production wifi router config when present, or the test config when not present
```

### `release download`

```text
$ cros_openwrt_image_builder release download --help
Downloads a released image configured for the specified device. The current image is downloaded by default, but specific images may be downloaded by using the --uuid, --dut, ---cros_version, or --next flags.

Usage:
  cros_openwrt_image_builder release download <device_name> [flags]

Flags:
      --cros_version string   Download the image that has the highest MinDutCrosReleaseVersion that is equal to or less than this CHROMEOS_RELEASE_VERSION (compatible with --dut)
      --dst string            Directory to download image archive file to (default ".")
      --dut string            Download the next image if the dut is in the verification pool, or the current image otherwise
  -h, --help                  help for download
      --next                  Download the next image for this device
      --uuid string           Download the image configured for this device with a matching ImageUUID (case-insensitive)

Global Flags:
      --bucket string             GCS storage bucket to use (both prod and test use the same bucket) (default "chromeos-connectivity-test-artifacts")
      --gcloud_cred_file string   The gcloud credential file to use with the GCS API (uses gcloud CLI application-default credentials when unset)
  -p, --prod                      Uses the production wifi router config when present, or the test config when not present
```

### `release upload`

```text
$ cros_openwrt_image_builder release upload --help
Uploads a local image to GCS and adds it to the release config

Usage:
  cros_openwrt_image_builder release upload <min_cros_version> <path_to_builder_output_dir_with_image_archive_and_build_config> [flags]

Flags:
      --current                    Sets the current image to this new image (will be set as current if no other images are configured for device)
  -h, --help                       help for upload
      --next                       Sets the next image to this new image
      --overwrite_cros_version     Allows for replacing other images in the image config with matching min_cros_version values
      --overwrite_existing_image   Allows overwriting of an image that already exists with the same ImageUUID

Global Flags:
      --bucket string             GCS storage bucket to use (both prod and test use the same bucket) (default "chromeos-connectivity-test-artifacts")
      --gcloud_cred_file string   The gcloud credential file to use with the GCS API (uses gcloud CLI application-default credentials when unset)
  -p, --prod                      Uses the production wifi router config when present, or the test config when not present
```

## Project Structure

The `./custom_packages` directory contains source code for custom OpenWrt
packages.

The `./image_builder` directory contains the source code for and builds of the
`cros_openwrt_image_builder` CLI utility.

The `./included_image_files` directory contains files that are added to built
OpenWrt OS images.
