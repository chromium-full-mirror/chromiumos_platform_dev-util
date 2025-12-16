# Sample Cache Server

This cache server is useful for local-cft and PVS, but is not used in satlab or
the mainlab.

## Build and run locally

Inside the chroot:
```
~/chromiumos/src/platform/dev/fast_build.sh -b go.chromium.org/chromiumos/prototytpe/cache/cmd/cacheserver -o ~/go/bin/cacheserver && ~/go/bin/cacheserver -port 8082
```

Outside the chroot:
```shell
cd cmd/cacheserver/ && go run . -port 8082
```

## Build and publish to CIPD for PVS

See [PVS Playbook](http://docs/document/d/1lf7lMfiSGvmG2OOHnnZAo3hmk-r6MG1gZ2CSoKZlio4?tab=t.0#heading=h.eqrno89hh29d)

## Build and publish docker image for local-cft

If you know how, please add instructions here. Meanwhile
[create a bug](https://b.corp.google.com/issues/new?component=1335856&template=1800751)
for the Test Scheduling team to do it.
