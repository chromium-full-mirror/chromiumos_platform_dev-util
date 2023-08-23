# Tast AB test script

This directory contains a script to compare the performance difference between
two versions of a binary (`binaryA` and `binaryB`) by running a tast test many
times. This script repeats the following sequence 30 times:

1.  Replace the target binary (e.g. `/usr/bin/vm_concierge`) with `binaryA`
1.  Run the give tast test
1.  Replace the target binary with `binaryB`
1.  Run the give tast test

This script was used for go/arcvm-cache-always-data-media.

## Example usage

First, build vm_concierge before and after applying a commit. Put the first
binary to `/mnt/stateful_partition/vm_concierge-main` and the second to
`/mnt/stateful_partition/vm_concierge-modified`.

Then, run the following command:

```shell
# Run in cros_sdk
TAST_NAME=arc.RuntimePerf.sniper3d_virtio_fs_vm

TARGET_BINARY_PATH/usr/bin/vm_concierge
MAIN_BINARY=/mnt/stateful_partition/vm_concierge-original
MODIFIED_BINARY=/mnt/stateful_partition/vm_concierge-modified

./run_abtest.sh ${DUT} ${TAST_NAME} ${TARGET_BINARY_PATH} ${MAIN_BINARY} ${MODIFIED_BINARY}
```
