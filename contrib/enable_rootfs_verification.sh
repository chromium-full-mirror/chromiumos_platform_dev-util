#!/bin/bash
# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Disable the ssh variable expansion lint, since it is exactly what we want to
# do to expand variables at the client.
# shellcheck disable=SC2029

set -eEu

HELP="Usage: $0 DUT

Enables the rootfs verification on DUT. This script calculates the hash of
the rootfs and enables the verity of the rootfs.
"

if [[ $# -ne 1 || "$1" == -h || "$1" == --help ]]; then
    echo "${HELP}"
    exit 1
fi

DUT="$1"

main() {
    local rootdev partition rootdev_s
    echo "Before:"
    rootdev="$(ssh "${DUT}" rootdev)"
    partition="$(( $(echo -n "${rootdev}" | grep -o "[0-9]$") - 1 ))"
    echo "rootdev:    ${rootdev}"
    rootdev_s="$(ssh "${DUT}" rootdev -s)"
    echo "rootdev -s: ${rootdev_s}"
    echo "partition: ${partition}"

    if [[ "${rootdev}" =~ dm-0 ]]; then
        echo "Rootfs verification is already turned on"
        exit 1
    fi

    # FIXME:
    # In theory "sync"s should suffice instead of a reboot, but we often
    # observe disk change after reboot for some reason.
    echo "Reboot before remounting"
    reboot_and_wait "${DUT}"

    echo "Remounting rootfs as RO"
    ssh "${DUT}" "/usr/share/vboot/bin/make_dev_ssd.sh --save_config /tmp/config --partitions ${partition}"
    ssh "${DUT}" "sed 's| rw | ro |g' -i /tmp/config.${partition}"
    ssh "${DUT}" "/usr/share/vboot/bin/make_dev_ssd.sh --set_config /tmp/config --partitions ${partition}"
    reboot_and_wait "${DUT}"

    local root_fs_blocks root_fs_block_sz root_fs_size root_fs_blocks salt alg table
    echo "Running verity"
    root_fs_blocks=$(ssh "${DUT}" "dumpe2fs ${rootdev} 2>/dev/null" | grep "Block count" | tr -d ' ' | cut -f2 -d:)
    echo "root_fs_blocks: ${root_fs_blocks}"
    root_fs_block_sz=$(ssh "${DUT}" "dumpe2fs ${rootdev} 2>/dev/null" | grep "Block size" | tr -d ' ' | cut -f2 -d:)
    echo "root_fs_block_sz: ${root_fs_block_sz}"
    root_fs_size=$(( root_fs_blocks * root_fs_block_sz ))
    echo "root_fs_size: ${root_fs_size}"
    root_fs_blocks=$(( root_fs_size / 4096 ))
    echo "root_fs_blocks: ${root_fs_blocks}"
    salt=$(ssh "${DUT}" "cat /proc/cmdline" | sed 's/.*salt=\([^ \t"]\+\).*/\1/g')
    echo "salt: ${salt}"
    alg=$(ssh "${DUT}" "cat /proc/cmdline" | sed 's/.*alg=\([^ \t"]\+\).*/\1/g')
    echo "alg: ${alg}"
    table=$(ssh "${DUT}" "verity --mode=create --alg=${alg} --payload=${rootdev} --payload_blocks=${root_fs_blocks} --hashtree=/tmp/rootfs.hash --salt=${salt}")
    echo "Table:"
    echo "${table}"
    echo

    local rootdev_disk hash_offset
    echo "Writing hash data to disk"
    rootdev_disk=$(ssh "${DUT}" rootdev -d)
    hash_offset=$(ssh "${DUT}" "cgpt show -i $(( partition + 1 )) -b ${rootdev_disk}")
    hash_offset=$(( hash_offset + (root_fs_size / 512) ))
    ssh "${DUT}" "dd bs=512 seek=${hash_offset} if=/tmp/rootfs.hash of=${rootdev_disk} conv=notrunc status=none"
    echo

    local root_hexdigest_string
    echo "Modifying command line"
    ssh "${DUT}" "/usr/share/vboot/bin/make_dev_ssd.sh --save_config /tmp/config --partitions ${partition}"
    ssh "${DUT}" "sed 's|root=PARTUUID=%U/PARTNROFF=1|root=/dev/dm-0|g' -i /tmp/config.${partition}"
    ssh "${DUT}" "sed 's|dm_verity.dev_wait=0|dm_verity.dev_wait=1|g' -i /tmp/config.${partition}"
    ssh "${DUT}" "sed 's|payload=ROOT_DEV|payload=PARTUUID=%U/PARTNROFF=1|g' -i /tmp/config.${partition}"
    ssh "${DUT}" "sed 's|hashtree=HASH_DEV|hashtree=PARTUUID=%U/PARTNROFF=1|g' -i /tmp/config.${partition}"
    # shellcheck disable=SC2001
    root_hexdigest_string=$(echo "${table}" | sed 's/.*\(root_hexdigest=\S\+\).*/\1/g')
    ssh "${DUT}" "sed 's/root_hexdigest=\S\+/${root_hexdigest_string}/g' -i /tmp/config.${partition}"
    ssh "${DUT}" "/usr/share/vboot/bin/make_dev_ssd.sh --set_config /tmp/config --partitions ${partition}"
    ssh "${DUT}" "cat /tmp/config.${partition}"
    echo

    reboot_and_wait "${DUT}"
    echo

    local new_rootdev new_rootdev_s
    echo "After:"
    new_rootdev=$(ssh "${DUT}" rootdev)
    echo "rootdev:    ${new_rootdev}"
    new_rootdev_s=$(ssh "${DUT}" rootdev -s)
    echo "rootdev -s: ${new_rootdev_s}"

    if [[ "${new_rootdev}" != /dev/dm-0 ]]; then
        echo "The new root dev is not /dev/dm-0. Enabling rootfs verification failed."
        exit 1
    fi
    if [[ "${new_rootdev_s}" != "${rootdev_s}" ]]; then
        echo "The new and the old root dev partition don't match. Enabling rootfs verification failed."
        exit 1
    fi
}

reboot_and_wait() {
    echo "Rebooting..."
    timeout 10s ssh "$1" reboot || true
    echo "Waiting for a ssh connection"
    sleep 10
    while ! timeout 10s ssh "$1" echo test &> /dev/null; do sleep 5; done
    echo
}

main
