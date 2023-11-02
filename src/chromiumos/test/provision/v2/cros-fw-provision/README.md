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

./chroot/usr/bin/local-cft board $BOARD -host $D -stayalive -port cros-dut=8123
# If you made changes in another service like cros-dut add
# --chroot ~/chromiumos/chroot --localservices cros-dut
```

To test detailed request via ssh
```
~/chromiumos/src/platform/dev/fast_build.sh -b chromiumos/test/provision/v2/cros-fw-provision -o ~/go/bin && \
cat >/tmp/input.json <<EOF && ~/go/bin/cros-fw-provision cli -input /tmp/input.json -output /tmp/output.json ; cat /tmp/output.json
{
        "board": "${BOARD?}",
        "model": "${MODEL?}",
        "dut_server_address": {
                "address": "localhost",
                "port": 8123
        },
        "useServo": false,
        "detailedRequest": {
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
EOF
```

## Dependencies

cros-fw-provision expects cros-cache, and either cros-dut or cros-servod, to be
running.

In the lab, the cache server should already be up.  Local runs of
cros-fw-provision will need the cache server started.

`cros-dut` is needed if provisioning directly.

`cros-servod` is needed if provisioning over
[servo]("https://chromium.googlesource.com/chromiumos/third_party/hdctools/+/HEAD/README.md").
