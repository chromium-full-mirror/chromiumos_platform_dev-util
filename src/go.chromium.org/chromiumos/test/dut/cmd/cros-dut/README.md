# Build and run locally

Enter chroot and run

```shell
CACHE_SERVER=192.168.100.1
~/chromiumos/src/platform/dev/fast_build.sh -b go.chromium.org/chromiumos/test/dut/cmd/cros-dut -o ~/go/bin/cros-dut && ~/go/bin/cros-dut -cache_address ${CACHE_SERVER?}:8082 -dut_address ${DUT_HOSTNAME?}:22
```

## Send a test message

Outside chroot run

```shell
source ~/chromiumos/out/home/${USER?}/.cftmeta
grpc_cli call localhost:$SERVICE_PORT chromiumos.test.api.DutService.ExecCommand --channel_creds_type=insecure --call_creds=none <<EOF
command: "ls /"
EOF
```
