# cros-fw-provision

This directory contains cros-fw-provision: a service for preparing
devices-under-test with specific application processor ("AP") and embedded
controller ("EC") firmware ("fw") builds.  Preparing devices with specific
builds is called "provisioning".

See go/cft-fw-provision-v2 for background.

## Launching

A prebuilt cros-fw-provision will be available in the chroot.  To launch it,
execute, `cros-fw-provision server` or `cros-fw-provision cli`.  `server` spins
up a service and waits for a gRPC request to come over the network before
provisioning starts.  `cli` immediately executes the request, which is required
to be passed in during launch.

### Building locally

See also go/cft-fw-provision-local-guide.

To build and run locally:

These example assume some vars are set in your shell:
```
BOARD=myBoard
MODEL=myModel
D=192.168.0.0
FW_IMAGE=$(gsutil ls -l gs://chromeos-image-archive/firmware-{brya,glados,oak,${BOARD?}}*-branch/*/${BOARD?}/firmware_from_source.tar.bz2 | grep -v TOTAL | sort -k2 | tail -1 | awk '{print $3}')
```

Outside chroot
```
cd ~/chromiumos
cros workon --host start local-cft && cros_sdk -- sudo emerge local-cft

./chroot/usr/bin/local-cft -board ${BOARD?} -host ${D?} -stayalive -port cros-dut=8123 -port cache-server=8082
# If you made changes in another service like cros-dut run
# ./chroot/usr/bin/local-cft -board ${BOARD?} -host ${D?} -stayalive -port cros-dut=8123 -port cache-server=8082 --chroot ~/chromiumos/chroot --localservices cros-dut
```

To test detailed request via ssh
```
YOUR_IP_ADDRESS=192.168.100.1
cat >/tmp/startup.json <<STARTUP
{
        "dut": {
                "chromeos": {
                        "dut_model": {
                                "build_target": "${BOARD?}",
                                "model_name": "${MODEL?}"
                        }
                }
        },
        "dut_server": {
                "address": "localhost",
                "port": 8123
        },
        "metadata": {
                "@type": "chromiumos.test.api.FirmwareProvisionStartupMetadata",
                "cache_server_address": {
                        "address": "${YOUR_IP_ADDRESS?}",
                        "port": 8082
                }
        }
}
STARTUP
cat >/tmp/install.json <<INSTALL
{
        "metadata": {
                "@type": "chromiumos.test.api.FirmwareProvisionInstallMetadata",
                "firmware_config": {
                        "mainRoPayload": {
                                "firmwareImagePath": {
                                        "hostType": "GS",
                                        "path": "${FW_IMAGE?}"
                                }
                        },
                        "ecRoPayload": {
                                "firmwareImagePath": {
                                        "hostType": "GS",
                                        "path": "${FW_IMAGE?}"
                                }
                        }
                }
        }
}
INSTALL
~/chromiumos/src/platform/dev/fast_build.sh -b chromiumos/test/provision/v2/cros-fw-provision \
        -o ~/go/bin/cros-fw-provision && \
        ~/go/bin/cros-fw-provision cli -startup /tmp/startup.json -install /tmp/install.json
```

## Dependencies

cros-fw-provision expects cros-cache, and either cros-dut or cros-servod, to be
running.

In the lab, the cache server should already be up.  Local runs of
cros-fw-provision will need the cache server started.

`cros-dut` is needed if provisioning directly.

`cros-servod` is needed if provisioning over
[servo]("https://chromium.googlesource.com/chromiumos/third_party/hdctools/+/HEAD/README.md").

## Testing

Run the test with fast_build.sh:

```
~/chromiumos/src/platform/dev/fast_build.sh -t chromiumos/test/provision/v2/cros-fw-provision
```

### Generate coverage report

Build the ebuild with coverage enabled:

```
cros workon --host start cros-fw-provision
sudo USE="coverage" FEATURES="test noclean" emerge cros-fw-provision
```

Then open the html file at `/var/tmp/portage/dev-util/cros-fw-provision-9999/temp/coverage_logs/cros-fw-provision.html`
