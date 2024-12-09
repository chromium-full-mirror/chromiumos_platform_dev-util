# Build and run locally

1) Get a request.json file by downloading one from a luci test run or create one by hand.
1) Magically get a directory of test metadata proto files. It's in the
   us-docker.pkg.dev/cros-registry/test-services/foil-test container.
1) Then run:

```shell
cd ~/chromiumos
cros_sdk -- /mnt/host/source/src/platform/dev/fast_build.sh \
  -b go.chromium.org/chromiumos/test/execution/cmd/cros-test -o /home/$USER/go/bin/cros-test && \
  ~/chromiumos/out/home/$USER/go/bin/cros-test cli -input ~/Downloads/request.json
```

# Run tests

```shell
cd ~/chromiumos
cros_sdk -- /mnt/host/source/src/platform/dev/fast_build.sh -t go.chromium.org/chromiumos/test/execution/cmd/cros-test/...
```

# Rebuild and push docker image

## Rebuild locally (doesn't work)
These instructions should work, but the build-dockerimages.py tool is broken.
```shell
cd ~/chromiumos && \
cros_sdk -- setup_board -b amd64-generic && \
cros workon --board amd64-generic start cros-test && \
cros_sdk -- emerge-amd64-generic cros-test && \
src/platform/dev/src/chromiumos/test/python/src/docker_libs/cli/build-dockerimages.py \
  --service cros-test --out-dir out --build_type local chroot /build/amd64-generic
```

## Use CQ build container (doesn't work for tradefed)

Upload a cl with your changes, vote CQ+1, then look at the
`amd64-generic-cq` build, find the step `create test service containers`, click
on the link for cros-test, and copy the container path.

However, this doesn't appear to have the adb binary or the tradefed test metadata present.

## Replace the cros-test binary in an existing container

If you only want to change the binary and not anything else, you can just
download an existing container and replace the binary.

1) Run the container: `docker run --rm -d --name cros-test-tmp <containerPath@sha256:etc> sleep 3600`
2) Build cros-test using fast_build.sh as documented above:
```shell
cros_sdk -- /mnt/host/source/src/platform/dev/fast_build.sh \
  -b go.chromium.org/chromiumos/test/execution/cmd/cros-test -o /home/$USER/go/bin/cros-test
```
3) Copy binary into the container:
```shell
docker cp out/home/$USER/go/bin/cros-test cros-test-tmp:/usr/bin
docker exec cros-test-tmp sudo chown chromeos-test:chromeos-test /usr/bin/cros-test
docker exec cros-test-tmp sudo chmod 755 /usr/bin/cros-test
```
4) Outside the container again, `docker commit cros-test-tmp us-docker.pkg.dev/cros-registry/test-services/foil-test:$USER-testing`
5) `docker push us-docker.pkg.dev/cros-registry/test-services/foil-test:$USER-testing`
6) Cleanup the container: `docker stop cros-test-tmp`

# Test with luci led tool

Find a test run that you want to try out with your new code in luci. For example
http://ci.chromium.org/b/8728904563748506801

```shell
led get-build 8728904563748506801 >~/job.json
```

Edit job.json, and find all of the `cros-test` container entries, and edit the
containerImagePath to point to your container from above.

Example:
```json
"orderedContainerRequests": [
        {
                "container": {
                        "crosTest": {
                        }
                },
                "containerImageKey": "cros-test",
                "containerImagePath": "us-docker.pkg.dev/cros-registry/test-services/foil-test@sha256:00e632fd72d9e2d3ba94ca760b97266c973f61d7286a39ef5675e295ee96593e",
                "dynamicIdentifier": "cros-test",
                "network": "adb-network"
                }
        ],
```

Also find the field named `deadline` and remove it.

Run the job, this will also pick up any recipe edits:
```shell
cd ~/chromiumos
( cd infra/recipes ; cat ~/job.json | led edit-recipe-bundle | led launch )
```
