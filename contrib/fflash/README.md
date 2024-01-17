# fflash

[go/cros-fflash](https://goto.google.com/cros-fflash)

`fflash` is a tool to update a ChromiumOS device with a test image from `gs://` (Google Cloud Storage).

## Dependencies

[Install depot_tools](https://commondatastorage.googleapis.com/chrome-infra-docs/flat/depot_tools/docs/html/depot_tools_tutorial.html#_setting_up)
and you are all set!
If you are a Chromium or ChromiumOS developer you are probably already set up.

## Set up and Run

1.  Make sure:
    *   You can `ssh ${dut_host}` without typing a password.
    *   Your DUT has access to the internet.

2.  ```
    ~/chromiumos/src/platform/dev/contrib/fflash/fflash ${dut_host}
    ```

    Where `${dut_host}` is the ssh target.

    The first time you invoke `fflash`, it might do some extra set up:

    *   Build itself from source.
    *   If prompted, use `luci-auth` to grant `fflash` access to Google Cloud Storage, and run again.

### Alternative: Run without full ChromiumOS checkout

```
git clone https://chromium.googlesource.com/chromiumos/platform/dev-util
./dev-util/contrib/fflash/fflash
```

## Usage examples

```
fflash ${dut_host}  # flash latest canary
fflash ${dut_host} --clobber-stateful=yes  # flash latest canary and clobber stateful
fflash ${dut_host} -R104  # flash latest R104
fflash ${dut_host} -R104-14911.0.0  # flash 104-14911.0.0
fflash ${dut_host} --board=cherry64 -R104  # flash latest R104 for board cherry64
fflash ${dut_host} --gs=gs://chromeos-image-archive/cherry-release/R104-14911.0.0  # flash specified gs:// directory
fflash ${dut_host} --gs=gs://chromeos-image-archive/volteer-cq/R118-15584.0.0-86601-8772438681633523617  # From CQ+1
fflash ${dut_host} --snapshot  # flash a recent snapshot build
fflash ${dut_host} --snapshot -R119  # flash a R119 snapshot build
fflash ${dut_host} --snapshot -R119-15634.0.0-88565-  # flash the snapshot 88565
fflash ${dut_host} --postsubmit  # flash a recent postsubmit build
fflash ${dut_host} --postsubmit -R115  # flash a R115 postsubmit build
fflash ${dut_host} --postsubmit -R115-15474.0.0-82578-  # flash the snapshot 82578
fflash ${dut_host} --dry-run  # print what would be flashed without actually flashing
fflash --help
```

## What `fflash` does

*   flashes the specified image, or latest image on the device
*   clobbers stateful (optional, disabled by default)
*   disables verified boot
*   clears tpm owner (optional, disabled by default)
*   reboot the device

### it does not

*   ask for sudo password
*   flash minios

`fflash` is faster than `cros flash` if the connection between `cros flash` and the DUT is slow.
`cros flash` proxies the `gs://chromeos-image-archive` images for the DUT (Google Cloud -> workstation -> DUT).
`fflash` makes the DUT download the OS images directly, by granting the DUT a restricted
access token to the Chrome OS release image directory (Google Cloud -> DUT).

## Development

fflash is written in go and built with Bazel. Bazel handles the go toolchain.

You can invoke `bazel` through the `tools/bazel` launcher. If you obtained
`bazel` through other channels, as long as it satisifies `.bazelversion`, it
should also work.

### Testing

#### Unit tests

```
tools/bazel test //...
```

#### Integration tests

```
tools/bazel run //cmd/integration-test ${dut_host}
```

### Add go source / deps

Add the source normally and:

```
go get -u -d ./...
go mod tidy
tools/bazel run //:gazelle
```

to update the BUILD.bazel files.

Bazel emits a warning and a `buildozer` command to automatically fix the `MODULE.bazel` file if
it is outdated.

## Troubleshooting

### partner_testing_rsa

If using crosfleet devices, you might need to copy
https://chrome-internal.googlesource.com/chromeos/sshkeys/+/HEAD/partner_testing_rsa
to `~/.ssh/partner_testing_rsa`.
Refer to [b/260010430](https://issuetracker.google.com/issues/260010430)
for more information.

## Bugs & Feedback

*   Join the [fflash-users] Google Group (Googlers only).

*   [File a bug] in our [issue tracker].

[fflash-users]: https://groups.google.com/a/google.com/g/fflash-users
[File a bug]: https://issuetracker.google.com/issues/new?component=1264059
[issue tracker]: https://issuetracker.google.com/issues?q=status:open%20componentid:1264059&s=created_time:desc

## Related tools

*   [cros flash] is the officially supported tool to image ChromiumOS devices.
    It also allow flashing locally-built images.

*   [quick-provision] is useful if your OS image is already on your ChromiumOS device.

[cros flash]: https://chromium.googlesource.com/chromiumos/docs/+/HEAD/cros_flash.md
[quick-provision]: https://source.chromium.org/chromiumos/chromiumos/codesearch/+/main:src/platform/dev/quick-provision/
