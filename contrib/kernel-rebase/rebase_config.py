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
    "fixups/FIXUP: FROMLIST: overlayfs: handle XATTR_NOSECURITY flag for get xattr method.patch",
    "fixups/FIXUP: CHROMIUM: verity: bring over dm-verity-chromeos.c.patch",
    "fixups/FIXUP: CHROMIUM: Revert ext4: get discard out of jbd2 commit kthread contex.patch",
    "fixups/FIXUP: FROMLIST: Add flags option to get xattr method paired to __vfs_getxattr.patch",
    "fixups/FIXUP: FROMLIST: Add flags option to get xattr method paired to __vfs_getxattr2.patch",
    "fixups/FIXUP: UBUNTU: SAUCE: trace: add trace events for open exec and userlib.patch",
    "fixups/FIXUP: CHROMIUM: verity: bring over dm-verity-chromeos.c-partno.patch",
]

topic_patches["drivers"] = [
    "fixups/FIXUP: CHROMIUM: pci allowlist: sysfs attribute to lockdown external devices completely.patch",
    "fixups/FIXUP: Revert 'staging: remove ashmem'.patch",
    "fixups/FROMGIT: xhci: check for a pending command completion during command timeout.patch",
    "fixups/FIXUP: NOUPSTREAM: ANDROID: usb: gadget: f_audio_source: New gadget driver for audio output - string: Remove strlcpy.patch",
    "fixups/FIXUP: NOUPSTREAM: ANDROID: usb: gadget: configfs: Add Uevent to notify userspace2.patch",
    "fixups/FIXUP: NOUPSTREAM: ANDROID: usb: gadget: f_audio_source: New gadget driver for audio output.patch",
    "fixups/FIXUP: CHROMIUM: usb: typec: Implement UCSI driver for ChromeOS.patch",
    "fixups/FIXUP: BACKPORT: FROMLIST: pwm: Add support for different PWM output types.patch",
    "fixups/FIXUP: Revert staging: remove ashmem - unmapped_area.patch",
    "patches/Revert gpiolib: legacy: Remove unused gpio_request_array and gpio_free_array.patch",
]

topic_patches["bluetooth"] = [
    "fixups/FIXUP: CHROMIUM: bluetooth: Support userspace SCO.patch",
    "fixups/FIXUP: CHROMIUM: Bluetooth: Optimize the LE connection sequence.patch",
]

topic_patches["cros_ec"] = [
    "fixups/FIXUP: CHROMIUM: iio: cros_ec: flush as hwfifo attribute.patch",
    "fixups/FIXUP: CHROMIUM: platform chrome: wilco_ec: Add charge scheduling sysfs.patch",
]

topic_patches["gpu/other"] = [
    "fixups/FIXUP: CHROMIUM: gpu: mali: Apply r40p0 EAC release.patch",
]

topic_patches["drm"] = [
    "fixups/FIXUP: CHROMIUM: drm img-rogue: Add 1.17 IMG PowerVR Rogue driver.patch",
    "fixups/FIXUP: CHROMIUM: drm img-rogue: Add 1.17 IMG PowerVR Rogue driver2.patch",
    "fixups/FIXUP: FROMLIST: drm msm dpu: Replace definitions for dpu debug macros.patch",
    "fixups/FIXUP: CHROMIUM: drm img-rogue: Add 1.17 IMG PowerVR Rogue driver - string: Remove strlcpy.patch",
    "fixups/FIXUP: FROMLIST: drm vkms: Support multiple DRM objects crtcs etc per VKMS device.patch",
    "fixups/FIXUP: CHROMIUM: drm rockchip: Add GEM create ioctl support.patch",
    "fixups/FIXUP: CHROMIUM: drm-img-rogue: Add 1.17 IMG PowerVR Rogue driver.patch",
    "fixups/FIXUP: CHROMIUM: drm img-rogue: Add 1.17 IMG PowerVR Rogue driver - allmodconfig.patch",
    "fixups/FIXUP: CHROMIUM: drm img-rogue: Add 1.17 IMG PowerVR Rogue driver - assign_str.patch",
    "fixups/FIXUP: CHROMIUM: drm img-rogue: Add 1.17 IMG PowerVR Rogue driver - strict.patch",
    "fixups/FIXUP: CHROMIUM: drm img-rogue: Add 1.17 IMG PowerVR Rogue driver - thermal.patch",
    "fixups/FIXUP: CHROMIUM: drm mediatek: Add interface to allocate Mediatek GEM buffer.patch",
    "fixups/FIXUP: FROMLIST: drm vkms: Support multiple DRM objects crtcs, etc. per VKMS device.patch",
]

topic_patches["chromeos"] = [
    "fixups/FIXUP: CHROMIUM: LSM: chromiumos security module.patch",
    "fixups/FIXUP: CHROMIUM: LSM: chromiumos security module - mark.patch",
    "fixups/FIXUP: CHROMIUM: create alt-syscall infrastructure - strict.patch",
]

topic_patches["mm"] = [
    "fixups/FIXUP: CHROMIUM: mm: per-process reclaim pages.patch",
]

topic_patches["media"] = [
    "fixups/FIXUP: CHROMIUM: media: add camx drivers - support in SPI core.patch",
    "fixups/FIXUP: CHROMIUM: media: add camx drivers - Stop circularly including of_device.h and of_platform.h.patch",
    "fixups/FIXUP: CHROMIUM: media: add camx drivers - Rename min_buffers_needed field in vb2_queue.patch",
    "fixups/FIXUP: CHROMIUM: media: add camx drivers - string: Remove strlcpy.patch",
    "fixups/FIXUP: CHROMIUM: media: add camx drivers - Makefile.patch",
    "fixups/FIXUP: CHROMIUM: media: add camx drivers - assign_str.patch",
    "fixups/FIXUP: CHROMIUM: media: add camx drivers - vmalloc.patch",
]

topic_patches["dts"] = [
    "fixups/FIXUP: CHROMIUM: arm64: dts: mt8186: Add corsola-steelix and rusty board.patch",
    "fixups/FIXUP: CHROMIUM: arm64: dts: mt8186: Add dpi node.patch",
]

topic_patches["arch"] = [
    "patches/FIXUP: Fix incorrect pageref management from rebase.patch",
    "fixups/FIXUP: BACKPORT: FROMLIST: KVM: x86: Add a new param slot to op get_mt_mask in kvm_x86_ops.patch",
]

topic_patches["power-thermal"] = [
    "fixups/FIXUP: CHROMIUM: mt8195: backport legacy SOC_LVTS_TEMP as the out-of-tree driver - thermal.patch",
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
patch_deps = {
    "ddce70d74ef7": ["a9a10c006900", "ff5f250f00ad"],
    "ec6cf3d62980": ["4e20fae6c6f1"],
    "c4305a040e64": ["41f5cb08b0c2", "268a3f9a5de0", "dbc44b0d9303"],
}

# Add entry here to overwrite default disposition on particular commit
# WARNING: lines can be automatically appended here when user chooses
# to drop the commit from rebase script.
disp_overwrite = {}
# example
# disp_overwrite['62b865c66db4'] = 'drop'
disp_overwrite[
    "4dac2b933823"
] = "drop"  # FROMLIST: overlayfs: handle XATTR_NOSECURITY flag for get xattr method
disp_overwrite["f2968dcf8e73"] = [
    "move",
    "bluetooth",
]  # FROMLIST: devcoredump: Add per device sysfs entry to enable/disable coredump
disp_overwrite["c4b7214921e3"] = [
    "move",
    "block-fs",
]  # CHROMIUM: trace: rename events/fs_trace/ -> events/fs/
disp_overwrite["b92a48381564"] = [
    "move",
    "drivers",
]  # FROMGIT: r8152: Choose our USB config with choose_configuration() rather than probe()
disp_overwrite["3c6ead5c25ad"] = [
    "move",
    "mm",
]  # CHROMIUM: mm: Reschedule during per-process reclaim
disp_overwrite[
    "8f91972ac6f4"
] = "drop"  # Revert "Bluetooth: Fix a refcnt underflow problem for hci_conn"
disp_overwrite[
    "7dd4aecb1f30"
] = "drop"  # CHROMIUM: Bluetooth: Use addr instead of hci_conn in LE Connection complete
disp_overwrite["a1d603d646dd"] = "drop"  # CHROMIUM: Bluetooth: Fix LE pair
disp_overwrite[
    "bccfdf53c1b2"
] = "drop"  # FROMLIST: fuse: Definitions and ioctl for passthrough
disp_overwrite[
    "e807baabcf21"
] = "drop"  # FROMLIST: fuse: Passthrough initialization and release
disp_overwrite[
    "89a75916a564"
] = "drop"  # FROMLIST: fuse: Introduce synchronous read and write for passthrough
disp_overwrite[
    "7df297fddff2"
] = "drop"  # FROMLIST: fuse: Handle asynchronous read and write in passthrough
disp_overwrite[
    "bdf09513d0ea"
] = "drop"  # FROMLIST: fuse: Use daemon creds in passthrough mode
disp_overwrite[
    "5d03cf94b4e6"
] = "drop"  # FROMLIST: fuse: Introduce passthrough for mmap
disp_overwrite[
    "55e6f1de885e"
] = "drop"  # FROMLIST: fuse: Allocate unlikely used ioctl number for passthrough V1
disp_overwrite[
    "8f6407c6b59f"
] = "drop"  # CHROMIUM: fuse: Reposition FUSE_PASSTHROUGH flag
disp_overwrite[
    "d53cd35fcab1"
] = "drop"  # FROMLIST: x86/topology: Fix max_siblings calculation
disp_overwrite[
    "c484c2cc10d0"
] = "drop"  # CHROMIUM: hrtimer: Add a sysctl to lower resolution of timers
disp_overwrite[
    "719aa220616b"
] = "drop"  # CHROMIUM: tick-sched: Set last_tick correctly so that timer interrupts happen less
disp_overwrite[
    "2cf6f2d48db1"
] = "drop"  # CHROMIUM: arm64: dts: mt8186: Add krabby board
disp_overwrite[
    "8b097990bc68"
] = "drop"  # CHROMIUM: arm64: dts: mt8186: Add corsola-steelix and rusty board
disp_overwrite[
    "3a863e619c68"
] = "drop"  # CHROMIUM: arm64: dts: mt8186: Add tentacruel board
disp_overwrite[
    "b1ed2ad77ad5"
] = "drop"  # CHROMIUM: arm64: dts: mt8186: Add corsola-magneton board
disp_overwrite[
    "3be5eaf58b34"
] = "drop"  # CHROMIUM: arm64: dts: mt8186: Add magikarp board
disp_overwrite[
    "6de4bf555daa"
] = "drop"  # CHROMIUM: arm64: dts: mt8186: Add kingler board
disp_overwrite[
    "fbf81184897b"
] = "drop"  # CHROMIUM: arm64: dts: mt8186: Rename some properties of it6505dptx node
disp_overwrite[
    "6de4bf555daa"
] = "drop"  # CHROMIUM: arm64: dts: mt8186: Add kingler board
disp_overwrite[
    "e4bd95034d5c"
] = "drop"  # CHROMIUM: arm64: dts: mt8186: Adjust swing level-2
disp_overwrite[
    "971d4e980317"
] = "drop"  # CHROMIUM: arm64: dts: mt8186: Fix the dma-ranges for jpgenc
disp_overwrite[
    "3ec9388f79a4"
] = "drop"  # CHROMIUM: arm64: dts: mt8186: Adjust i2c-scl-delay to 22000
disp_overwrite[
    "a3805dd36acd"
] = "drop"  # CHROMIUM: arm64: dts: sc7280: Add all details needed for EAS enablement
disp_overwrite[
    "c810b19c3dc2"
] = "drop"  # REVISIT: ANDROID: power: wakeup_reason: add an API to log wakeup reasons
disp_overwrite[
    "c4f808c7b1a5"
] = "drop"  # FROMLIST: PM: sleep: Expose last succeeded resumed timestamp in sysfs
disp_overwrite["e7899c93671a"] = [
    "move",
    "mm",
]  # CHROMIUM: per-process reclaim correct isolate_lru_page usage
disp_overwrite[
    "fccf284589f4"
] = "drop"  # CHROMIUM: net: wwan: t7xx: Modify the LINUX/LK trigger source
disp_overwrite[
    "0f31c6bc52d7"
] = "drop"  # BACKPORT: FROMLIST: net: wwan: t7xx: Fix remove rescan block
disp_overwrite["933ba49b7349"] = [
    "move",
    "drm",
]  # FIXUP: CHROMIUM: drm/print: rename drm_debug* to be more syslog-centric
disp_overwrite["6f55d555a2bb"] = [
    "move",
    "drm",
]  # FIXUP: CHROMIUM: drm/print: Add tracefs support to the drm logging helpers
disp_overwrite[
    "d5d9543bbaaa"
] = "drop"  # CHROMIUM: arm64: dts: mt8186: Enable gpio pin for wowlan
disp_overwrite[
    "98238c3bcd94"
] = "drop"  # CHROMIUM: arm64: dts: mt8186: Update mode-switch nodes for it6505
disp_overwrite[
    "14f70eacae1e"
] = "drop"  # CHROMIUM: arm64: dts: mt8186: Add decoder device nodes
disp_overwrite[
    "05110e76c61c"
] = "drop"  # CHROMIUM: arm64: dts: mt8186: Add venc node
disp_overwrite[
    "1245bbd2e04e"
] = "drop"  # CHROMIUM: arm64: dts: mt8186: Add spmi node
disp_overwrite[
    "f724b94f5dab"
] = "drop"  # CHROMIUM: FIXUP: Ensure iwl7000 has precedence over iwlwifi
disp_overwrite["57edb04330b8"] = [
    "move",
    "block-fs",
]  # FROMLIST: Add flags option to get xattr method paired to __vfs_getxattr
disp_overwrite[
    "397d52eba612"
] = "drop"  # CHROMIUM: media: imx319: add workaround for link freq definition mismatch
disp_overwrite[
    "80e5d2e6b77f"
] = "drop"  # CHROMIUM: media: add Intel IPU6 drivers
disp_overwrite[
    "f4dfa4be51fe"
] = "drop"  # CHROMIUM: media: intel/ipu6: Do not use autosuspend for psys anymore
disp_overwrite[
    "f703c328dfa8"
] = "drop"  # CHROMIUM: media: ipu6: workaround multiple double frees on error paths
disp_overwrite[
    "e004e96fe00f"
] = "drop"  # CHROMIUM: media: ipu6: properly init pad flags
disp_overwrite[
    "1f7cd39fc4c0"
] = "drop"  # CHROMIUM: media: ipu6: remove pm_qos on error path
disp_overwrite["ca577f52b053"] = "drop"  # CHROMIUM: media: ipu6: remove TPG
disp_overwrite[
    "24c456038825"
] = "drop"  # CHROMIUM: media: ipu6: remove ipu_psys_unmapbuf_locked() forward declaration
disp_overwrite[
    "34c9abe3823f"
] = "drop"  # CHROMIUM: media: ipu6: rename bufmap member
disp_overwrite[
    "44430d7c4fa6"
] = "drop"  # CHROMIUM: media: ipu6: factor out kbuffer allocation
disp_overwrite[
    "a86dc4e3b2f4"
] = "drop"  # CHROMIUM: media: ipu6: validate buffer that we unmap
disp_overwrite[
    "aab9f85abd27"
] = "drop"  # CHROMIUM: media: ipu6: fix ipu_psys_kbuf_unmap()
disp_overwrite[
    "67f04c687f2c"
] = "drop"  # CHROMIUM: media: ipu6: add kbuffer map-count
disp_overwrite[
    "6ffdabad2947"
] = "drop"  # CHROMIUM: media: ipu6: introduce psys descriptors
disp_overwrite[
    "f3743e7b5e1d"
] = "drop"  # CHROMIUM: media: ipu6: remove extra buffer lookups
disp_overwrite[
    "20355d5b04a7"
] = "drop"  # CHROMIUM: media: ipu6: Use unlocked dma-buf APIs
disp_overwrite[
    "ea6e524959d6"
] = "drop"  # CHROMIUM: media/ipu6: Optimize the IPU MMU mapping and unmapping flow
disp_overwrite[
    "409c38ed52e0"
] = "drop"  # CHROMIUM: media/ipu6: correct the 'dev' argument in pkg_dir dma free
disp_overwrite[
    "f2080e4863f3"
] = "drop"  # CHROMIUM: media/ipu6: use 'unsigned long' to avoid type casting
disp_overwrite[
    "cc8cd102a7cf"
] = "drop"  # CHROMIUM: media/ipu6: initialise the pad flags for all pads of subdev
disp_overwrite[
    "64424bcc94c1"
] = "drop"  # CHROMIUM: media/ipu6: replace the strlcpy() with strscpy()
disp_overwrite[
    "511d2565e8d9"
] = "drop"  # CHROMIUM: media/ipu6: dump the stream config after all fields updated
disp_overwrite[
    "33171b7f77d6"
] = "drop"  # CHROMIUM: media/ipu6: remove the version.h inclusion
disp_overwrite[
    "e3da60571af8"
] = "drop"  # CHROMIUM: media/ipu6: Add video_nr module parameter for ISYS
disp_overwrite[
    "12887f1d56bc"
] = "drop"  # CHROMIUM: media/ipu6: load the firmware binaries from two paths
disp_overwrite[
    "c00133007231"
] = "drop"  # CHROMIUM: media/ipu6: cleanup the MMU before remove the bus devices
disp_overwrite[
    "f9cecfb4784e"
] = "drop"  # CHROMIUM: media: intel/ipu6: Fix some redundant resources freeing in pci_remove()
disp_overwrite[
    "dd70fa5a0c6e"
] = "drop"  # BACKPORT: FROMLIST: media: ucvideo: Add quirk for Logitech Rally Bar
