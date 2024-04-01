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
```

Outside chroot
```
cd ~/chromiumos
cros workon --host start local-cft && cros_sdk -- sudo emerge local-cft

gcloud auth application-default print-access-token >/dev/null || gcloud auth application-default login
```

If you are running with a local dut, you will need a cache server. Then set
the CACHE_SERVER variable below with your IP as seen from the DUT, otherwise
set the CACHE_SERVER variable to any cache server that is visible from the
DUT.
```
docker run -d --rm -P --cap-add=NET_RAW --name visible-cache-server --expose 8082 -p 8082:8082 --volume $HOME/.config/:/root/.config --volume /tmp/local-cft/cacheserver:/tmp/cacheserver us-docker.pkg.dev/cros-registry/test-services/cacheserver:prod cacheserver -location /tmp/cacheserver -port 8082
```

Alternatively, you can build the cache server from source and run it.
```
sudo ufw allow 8082 # Open the port in the linux firewall.
~/chromiumos/src/platform/dev/fast_build.sh -b go.chromium.org/chromiumos/prototytpe/cache/cmd/cacheserver -o ~/go/bin/cacheserver && ~/go/bin/cacheserver -port 8082
```

Then run the local-cft binary to run the cros-dut server.
```
./chroot/usr/bin/local-cft -board ${BOARD?} -host ${D?} -stayalive -port cros-dut=8123
# If you made changes in another service like cros-dut run
# ./chroot/usr/bin/local-cft -board ${BOARD?} -host ${D?} -stayalive -port cros-dut=8123 --chroot ~/chromiumos/chroot --localservices cros-dut
```

Inside chroot
```
FW_IMAGE=$(gsutil ls -l gs://chromeos-image-archive/firmware-{brya,glados,oak,${BOARD?}}*-branch/*/${BOARD?}/firmware_from_source.tar.bz2 | grep -v TOTAL | sort -k2 | tail -1 | awk '{print $3}')
CACHE_SERVER=192.168.100.1
cat >startup.json <<STARTUP
{
        "dut": {
                "chromeos": {
                        "dut_model": {
                                "build_target": "${BOARD?}",
                                "model_name": "${MODEL?}"
                        }
                },
                "cache_server": {
                        "address": {
                                "address": "${CACHE_SERVER?}",
                                "port": 8082
                        }
                }
        },
        "dut_server": {
                "address": "localhost",
                "port": 8123
        }
}
STARTUP
cat >install.json <<INSTALL
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
~/chromiumos/src/platform/dev/fast_build.sh -b go.chromium.org/chromiumos/test/provision/v2/cros-fw-provision \
        -o ~/go/bin/cros-fw-provision && \
        ~/go/bin/cros-fw-provision cli -startup startup.json -install install.json
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
~/chromiumos/src/platform/dev/fast_build.sh -t go.chromium.org/chromiumos/test/provision/v2/cros-fw-provision
```

### Generate coverage report

Build the ebuild with coverage enabled:

```
cros workon --host start cros-fw-provision
sudo USE="coverage" FEATURES="test noclean" emerge cros-fw-provision
```

Then open the html file at `/var/lib/chromeos/package-artifacts/dev-util/cros-fw-provision-9999/cros-artifacts/coverage_logs/cros-fw-provision.html`
