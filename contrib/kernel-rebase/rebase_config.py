# Commits to revert on each topic branch *before* topic fixups
# Copyright 2022 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Parts of the file are automatically generated
# pylint: disable=line-too-long

"""automatic rebase-specific data"""

import hooks  # pylint: disable=unused-import


verify_board = "brya-kernelnext"
verify_package = "chromeos-kernel-upstream"
rebase_repo = "kernel-upstream"
baseline_repo = "baseline/kernel-upstream/"

global_reverts = []

commit_hooks = {}
# calls a function on a selection of events, listed in the 'types' list.
# the different types are as follows:
# - conflict: called if a patch conflicts.
# - pre: called before the patch is applied.
# - post: called after successful application.
# - post_empty: called after a successful application that causes the commit to
#               be empty.
# - post_drop: called after triage drops a conflicting commit.
# The hook can be applied either only for a given commit (set the key to its SHA)
# or to all commits (set the key to '*')
# examples:
#   commit_hooks['e0c3be8f6bce'] = {'hook': hooks.pause, 'types': ['post']}
#   commit_hooks['*'] = {'hook': hooks.pause, 'types': ['conflict']}
# commit_hooks['a49b8bb6e63d'] = {'hook': hooks.pause, 'types': ['pre']}

topic_patches = {}
# patches and fixups applied on top of each topic branch
#
# examples:
# topic_patches['arch'] = [
#    'patches/KVM: mmu: introduce new gfn_to_pfn_page functions.patch',
# ]
#
# topic_patches['cros_ec'] = [
#    'fixups/platform: x86: add ACPI driver for ChromeOS.patch',
# ]

topic_patches["block-fs"] = [
    "fixups/FIXUP: FROMLIST: fuse: Introduce passthrough for mmap - i_mtime.patch",
    "fixups/FIXUP: FROMLIST: overlayfs: handle XATTR_NOSECURITY flag for get xattr method.patch",
    "fixups/FIXUP: FROMLIST: fuse: Handle asynchronous read and write in passthrough.patch",
    "fixups/CHROMIUM: trace: rename events fs_trace - events fs.patch",
]

topic_patches["drivers"] = [
    "fixups/FIXUP: CHROMIUM: pci allowlist: sysfs attribute to lockdown external devices completely.patch",
    "fixups/FIXUP: Revert 'staging: remove ashmem'.patch",
]

topic_patches["arch"] = [
    "fixups/FIXUP: CHROMIUM: hrtimer: Add a sysctl to lower resolution of timers.patch",
]

topic_patches["bluetooth"] = [
    "fixups/FIXUP: CHROMIUM: bluetooth: Support userspace SCO.patch",
]

topic_patches["cros_ec"] = [
    "fixups/FIXUP: CHROMIUM: iio: cros_ec: flush as hwfifo attribute.patch",
    "fixups/FIXUP: CHROMIUM: platform chrome: wilco_ec: Add charge scheduling sysfs.patch",
]
# order for automatic branch merging in rebase.py.
# branches that aren't specified are merged in an unspecified order.
# example that first merges drm, then gpu/other, then others:
# merge_order_override = [
#     "drm",
#     "gpu/other"
# ]
merge_order_override = []

# patches to be cherry-picked after automatic merge
# example:
# merge_fixups = [
#     "e0783589ae58"
# ]
merge_fixups = []

# cherry-pick a list of patches before a given patch
# during automatic rebase.
# example:
# patch_deps = {
#     '0d022b4a1e19': ['6e18e51a1c19']
# }
patch_deps = {}

# Add entry here to overwrite default disposition on particular commit
# WARNING: lines can be automatically appended here when user chooses
# to drop the commit from rebase script.
disp_overwrite = {}
# example
# disp_overwrite['62b865c66db4'] = 'drop'
disp_overwrite[
    "4dac2b933823"
] = "drop"  # FROMLIST: overlayfs: handle XATTR_NOSECURITY flag for get xattr method
