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

topic_patches["acpi"] = [
    "fixups/0001-UPSTREAM-ACPI-video-force-native-for-Apple-MacbookPr.patch",
]

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
    "fixups/FIXUP: CHROMIUM: usb: typec: Implement UCSI driver for ChromeOS.patch",
    "fixups/FIXUP: BACKPORT: FROMLIST: pwm: Add support for different PWM output types.patch",
    "fixups/FIXUP: Revert staging: remove ashmem - unmapped_area.patch",
    "fixups/FIXUP: CHROMIUM: usb: typec: Implement UCSI driver for ChromeOS-read.patch",
    "fixups/FIXUP: CHROMIUM: platform:x86:intel:vsec: Add support for Meteor Lake ES0.patch",
    "fixups/FIXUP: CHROMIUM: HID: multitouch: Scope mt_reset actions to haptic devices.patch",
    "fixups/CHROMIUM: Revert usb: typec: ucsi: Remove unused fields from struct ucsi_connector_status.patch",
    "fixups/FIXUP: NOUPSTREAM: ANDROID: usb: gadget: f_audio_source: New gadget driver for audio output.patch",
    "fixups/FIXUP: NOUPSTREAM: ANDROID: usb: gadget: f_audio_source: New gadget driver for audio output 2.patch",
    "fixups/FIXUP: FROMLIST: platform:x86: Add virtual PMC driver used for S2Idle.patch",
    "fixups/FIXUP-CHROMIUM-virtwl-add-virtwl-driver.patch",
]

topic_patches["bluetooth"] = [
    "fixups/FIXUP: CHROMIUM: bluetooth: Support userspace SCO.patch",
]

topic_patches["cros_ec"] = [
    "fixups/FIXUP: CHROMIUM: iio: cros_ec: flush as hwfifo attribute.patch",
    "fixups/FIXUP: CHROMIUM: platform chrome: wilco_ec: Add charge scheduling sysfs.patch",
    "fixups/FIXUP: CHROMIUM: iio: cros_ec_activity: add activity sensor driver.patch",
    "fixups/FIXUP: CHROMIUM: platform chrome: wilco_ec: Add charge scheduling sysfs - groups.patch",
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
    "fixups/FIXUP: CHROMIUM: drm-img-rogue: Add 1.17 IMG PowerVR Rogue driver.patch",
    "fixups/FIXUP: CHROMIUM: drm img-rogue: Add 1.17 IMG PowerVR Rogue driver - allmodconfig.patch",
    "fixups/FIXUP: CHROMIUM: drm img-rogue: Add 1.17 IMG PowerVR Rogue driver - assign_str.patch",
    "fixups/FIXUP: CHROMIUM: drm img-rogue: Add 1.17 IMG PowerVR Rogue driver - strict.patch",
    "fixups/FIXUP: CHROMIUM: drm img-rogue: Add 1.17 IMG PowerVR Rogue driver - thermal.patch",
    "fixups/FIXUP: CHROMIUM: drm mediatek: Add interface to allocate Mediatek GEM buffer.patch",
    "fixups/FIXUP: FROMLIST: drm vkms: Support multiple DRM objects crtcs, etc. per VKMS device.patch",
    "fixups/FIXUP: BACKPORT: FROMGIT: drm: bridge: it6505: Disable IRQ when powered off.patch",
    "fixups/FIXUP: CHROMIUM: drm: udl: Add cursor drm_plane support.patch",
    "fixups/FIXUP: CHROMIUM: drm: udl: Implement cursor - makefile.patch",
]

topic_patches["chromeos"] = [
    "fixups/FIXUP: CHROMIUM: LSM: chromiumos security module.patch",
    "fixups/FIXUP: CHROMIUM: LSM: chromiumos security module - mark.patch",
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
    "fixups/FIXUP: CHROMIUM: mtk-mdp: add driver to probe mdp components.patch",
    "fixups/FIXUP: CHROMIUM: media: platform: mediatek: mdp3: add support for ISP direct-linked to MDP.patch.patch",
    "fixups/0001-FIXUP-BACKPORT-FROMLIST-media-platform-mediatek-isp-Add-Mediatek-sen.patch",
    "fixups/0002-FIXUP-BACKPORT-FROMLIST-media-i2c-Add-a-driver-for-the-Galaxycore-GC.patch",
    "fixups/0003-FIXUP-BACKPORT-FROMLIST-media-platform-mediatek-isp-Add-Mediatek-CAM.patch",
    "fixups/0001-FIXUP-CHROMIUM-media-camx-rework-client-refcount-han.patch",
    "fixups/0001-FIXUP-update-remove-funcitions-definition.patch",
    "fixups/0002-FIXUP-CHROMIUM-media-add-camx-drivers-fix-gpio_free_.patch",
]

topic_patches["dts"] = [
    "fixups/FIXUP: CHROMIUM: arm64: dts: mt8186: Add corsola-steelix and rusty board.patch",
]

topic_patches["arch"] = [
    "patches/FIXUP: Fix incorrect pageref management from rebase.patch",
    "fixups/FIXUP: BACKPORT: FROMLIST: KVM: x86: Add a new param slot to op get_mt_mask in kvm_x86_ops.patch",
]

topic_patches["power-thermal"] = [
    "fixups/FIXUP: CHROMIUM: mt8195: backport legacy SOC_LVTS_TEMP as the out-of-tree driver - thermal.patch",
    "fixups/FIXUP: CHROMIUM: mt8195: backport legacy SOC_LVTS_TEMP as the out-of-tree driver.patch",
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

#
# devicetree
#
disp_overwrite[
    "b64d69a884ab"
] = "drop"  #  FROMLIST: dt-bindings: mediatek: Add Mediatek MDP3 dt-bindings
disp_overwrite[
    "af209b301e75"
] = "drop"  #  FROMLIST: dt-bindings: i2c: add attribute default-timing-adjust
disp_overwrite[
    "00dc43451ac4"
] = "drop"  #  CHROMIUM: dt-bindings: arm: mediatek: add early mt8186-corsola devices
disp_overwrite[
    "b4667bfee08a"
] = "pick"  #  FIXUP: FROMLIST: dt-bindings: usb: Add Type-C switch binding
disp_overwrite[
    "d32f140fc9c3"
] = "drop"  #  CHROMIUM: dt-bindings: arm: mediatek: Add MT8186 Chinchou/Chinchou360 Chromebooks

#
# mm
#
disp_overwrite[
    "bf27ad8f9d1c"
] = "drop"  #  CHROMIUM: mm/mglru: simplify reset_ctrl_pos
disp_overwrite[
    "4ea7628b2be5"
] = "drop"  #  CHROMIUM: mm/mglru: Split scan seqno from gen seqno
disp_overwrite[
    "31865023586b"
] = "drop"  #  CHROMIUM: mm/mglru: split max_seq into file vs anon types
disp_overwrite[
    "a97196e4754f"
] = "drop"  #  CHROMIUM: mm/mglru: age at different rates
disp_overwrite[
    "57d2c2ba37c2"
] = "drop"  #  CHROMIUM: Add a config to tie generation age together
disp_overwrite[
    "bc4946831df4"
] = "drop"  #  CHROMIUM: mm/mglru: Improve isolated page access bit harvesting
disp_overwrite[
    "f57164f428fb"
] = "drop"  #  CHROMIUM: mm/mglru: slow down promotion through page tables
disp_overwrite[
    "89f5fb0eb30b"
] = "drop"  #  CHROMIUM: mm/mglru: discount potential refaults by gen size
disp_overwrite["79555f7efc88"] = [
    "move",
    "mm",
]  #  CHROMIUM: mm: do not leave lazy MMU mode in per-process reclaim

#
# bluetooth
#
disp_overwrite["f2968dcf8e73"] = [
    "move",
    "bluetooth",
]  # FROMLIST: devcoredump: Add per device sysfs entry to enable/disable coredump
disp_overwrite[
    "7dd4aecb1f30"
] = "drop"  # CHROMIUM: Bluetooth: Use addr instead of hci_conn in LE Connection complete
disp_overwrite["a1d603d646dd"] = "drop"  # CHROMIUM: Bluetooth: Fix LE pair
disp_overwrite[
    "343d22969268"
] = "drop"  # FROMLIST: Bluetooth: btusb: Fix failed to send func ctrl for MediaTek devices.
disp_overwrite[
    "fe8b3b7530db"
] = "drop"  # CHROMIUM: Add a new MGMT error code for 0x3E HCI error.
disp_overwrite[
    "c3960a2af1a7"
] = "drop"  # CHROMIUM: bluetooth: Remove Mode Change from suspend event mask
disp_overwrite[
    "6cf88d1fae63"
] = "drop"  # CHROMIUM: Use link policies to disallow role switches
disp_overwrite[
    "e7a013b46db8"
] = "drop"  # CHROMIUM: Bluetooth: Optimize the LE connection sequence
disp_overwrite[
    "7cc71dbe32f3"
] = "drop"  # CHROMIUM: Bluetooth: hci_sync: keep advertisements during power off
disp_overwrite[
    "fec604565502"
] = "drop"  # CHROMIUM: Fix hci_connect_le argument order

#
# net
#
disp_overwrite[
    "0f31c6bc52d7"
] = "drop"  # BACKPORT: FROMLIST: net: wwan: t7xx: Fix remove rescan block
disp_overwrite[
    "fccf284589f4"
] = "drop"  # CHROMIUM: net: wwan: t7xx: Modify the LINUX/LK trigger source
disp_overwrite[
    "40a319ebd082"
] = "drop"  # CHROMIUM: net: wwan: t7xx: Delete flash FW end reboot rescan
disp_overwrite[
    "c940646a64e5"
] = "drop"  # CHROMIUM: net: wwan: t7xx: Add support for cold reboot
disp_overwrite[
    "4c11c48db0a6"
] = "drop"  # CHROMIUM: net: wwan: t7xx: Add delay between remove and rescan
disp_overwrite[
    "ca4893020e63"
] = "drop"  # CHROMIUM: net: wwan: t7xx: Compat for MR2 software
disp_overwrite[
    "fc836abd5821"
] = "drop"  # CHROMIUM: net: wwan: t7xx: Restore udev events
disp_overwrite[
    "839fccb8f6e4"
] = "drop"  # FROMLIST: net: wwan: t7xx: Enable fw flashing and coredump collection
disp_overwrite[
    "e38ee2af3e5f"
] = "drop"  # FROMLIST: net: wwan: t7xx: PCIe reset rescan
disp_overwrite[
    "0806c68cfea4"
] = "drop" # CHROMIUM: net: wwan: t7xx Fix use-after-free in rescan logic

#
# drivers
#
disp_overwrite[
    "c7f3abf7e224"
] = "drop"  #  CHROMIUM: HID: multitouch: skip driver reset for Zinitix device
disp_overwrite[
    "59e047f756be"
] = "drop"  #  CHROMIUM: HID: multitouch: skip driver reset for elantech touchpad
disp_overwrite[
    "6cecbf77f5ac"
] = "drop"  #  CHROMIUM: HID: multitouch: skip driver reset for elantech touchpad
disp_overwrite[
    "ca9e8243d508"
] = "drop"  #  CHROMIUM: HID: multitouch: skip driver reset for pixart device

#
# uncategorized
#
disp_overwrite[
    "4dac2b933823"
] = "drop"  # FROMLIST: overlayfs: handle XATTR_NOSECURITY flag for get xattr method
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
disp_overwrite["57edb04330b8"] = [
    "move",
    "block-fs",
]  # FROMLIST: Add flags option to get xattr method paired to __vfs_getxattr
disp_overwrite[
    "dd70fa5a0c6e"
] = "drop"  # BACKPORT: FROMLIST: media: ucvideo: Add quirk for Logitech Rally Bar
disp_overwrite["c4305a040e64"] = [
    "move",
    "drivers",
]  # CHROMIUM: usb: typec: Implement UCSI driver for ChromeOS
disp_overwrite[
    "b4c93c2ddd99"
] = "drop"  # FIXUP: CHROMIUM: Restrict swapon() to "zram" devices / lock down zram
disp_overwrite[
    "8318416bf47a"
] = "drop"  # FROMLIST: drm/mediatek: Implement shutdown
disp_overwrite["f808e6d14128"] = [
    "move",
    "mm",
]  # CHROMIUM: mm: per-process reclaim: fix RECLAIM_FILE
disp_overwrite[
    "d62edf60bfc9"
] = "drop"  # CHROMIUM: arm64: mt8186: dts: enable MALI gpu on Corsola
disp_overwrite[
    "fd7ab9476ab9"
] = "drop"  # FROMLIST: media: imx258: add vblank control to support wide frame rate range
disp_overwrite[
    "c861af77011e"
] = "drop"  # CHROMIUM: ARM64: Define compat syscall numbers
disp_overwrite[
    "31a6a5970717"
] = "drop"  # CHROMIUM: arm64: dts: mt8183: Modify sustainable-power for cerise and fennel
disp_overwrite[
    "791e80ea0961"
] = "drop"  # CHROMIUM: media: uvcvideo: Revert downstream race-condition fix
disp_overwrite[
    "9007c6a54539"
] = "drop"  # CHROMIUM: media: usb: uvc: Renumber downstream UVC_QUIRK_DISABLE_AUTOSUSPEND
disp_overwrite[
    "077df6d333e0"
] = "drop"  # FROMLIST: media: uvc: Handle cameras with invalid descriptors
disp_overwrite[
    "0b3a1c843466"
] = "drop"  #  FROMLIST: media: uvcvideo: Cancel async worker earlier
disp_overwrite[
    "4e0dfeaa0125"
] = "drop"  #  FROMLIST: media: uvcvideo: Lock video streams and queues while unregistering
disp_overwrite[
    "c3bbefdfcd24"
] = "drop"  #  FROMLIST: media: uvcvideo: Release stream queue when unregistering video device
disp_overwrite[
    "d3de69550a64"
] = "drop"  #  FROMLIST: media: uvcvideo: Protect uvc queue file operations against disconnect
disp_overwrite[
    "bcd3e90a5e56"
] = "drop"  #  FROMLIST: media: uvcvideo: Abort uvc_v4l2_open if video device is unregistered
disp_overwrite[
    "d6bd2210f1f4"
] = "drop"  # CHROMIUM: kallsyms: Display differences between kallsyms passes
disp_overwrite[
    "caa46d0439a3"
] = "drop"  #  CHROMIUM: media: ov08x40: fixup ov08x40_start_streaming()
disp_overwrite["bd5f054941fd"] = [
    "move",
    "media",
]  # CHROMIUM: soc: mediatek: mmsys: add support for ISP control
disp_overwrite[
    "dd3af47588d4"
] = "drop"  # CHROMIUM: thermal: Add notifier call chain for hot/critical events
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
    "b8c1a1d0b270"
] = "drop"  # CHROMIUM: media/ipu6: use vm_insert_pages() and check the return value
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
    "2bd0f90d4f40"
] = "drop"  # CHROMIUM: media/ipu6: retry auth if driver get unexpected response from CSE
disp_overwrite[
    "5279ef6bd95a"
] = "drop"  # CHROMIUM: media: ipu6: grab dma_buf ref-count for driver ownership
disp_overwrite[
    "a096e87d2ec2"
] = "drop"  # CHROMIUM: ipu6: merge global and local ipu-isys headers
disp_overwrite[
    "986067e9e5b6"
] = "drop"  # CHROMIUM: media: ipu6: update the media model name of IPU6 downstream driver
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
    "b8c1a1d0b270"
] = "drop"  # CHROMIUM: media/ipu6: use vm_insert_pages() and check the return value
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
    "2bd0f90d4f40"
] = "drop"  # CHROMIUM: media/ipu6: retry auth if driver get unexpected response from CSE
disp_overwrite[
    "5279ef6bd95a"
] = "drop"  # CHROMIUM: media: ipu6: grab dma_buf ref-count for driver ownership
disp_overwrite[
    "a096e87d2ec2"
] = "drop"  # CHROMIUM: ipu6: merge global and local ipu-isys headers
disp_overwrite[
    "986067e9e5b6"
] = "drop"  # CHROMIUM: media: ipu6: update the media model name of IPU6 downstream driver
disp_overwrite[
    "8025236810ae"
] = "drop"  # CHROMIUM: media: mtk-vpu: Ensure alignment of 8 for DTCM buffer
disp_overwrite[
    "6db105707a9e"
] = "drop"  # FROMGIT: drm/mediatek: Fix color format MACROs in OVL
disp_overwrite[
    "d83c1db3fde0"
] = "drop"  # CHROMIUM: drm/ttm: Remove wrong WARNING
disp_overwrite[
    "c8df13e235ef"
] = "drop"  # CHROMIUM: arm64: dts: mediatek: mt8183: Add compatible and opp-core-mask
disp_overwrite[
    "3948fca618e4"
] = "drop"  # CHROMIUM: arm64: dts: mt8186: Add thermal device nodes
disp_overwrite[
    "147e5871ff49"
] = "drop"  # CHROMIUM: arm64: mt8186: dts: Use legacy XHCI nodes
disp_overwrite[
    "0e91fa96dd78"
] = "drop"  # CHROMIUM: arm64: dts: mt8186: Set thermal trip point_1 temperature
disp_overwrite[
    "a998670c8fea"
] = "drop"  # FROMGIT: ACPI: video: force native for Apple MacbookPro11,2 and Air7,2
disp_overwrite["2e18aa079d2b"] = [
    "move",
    "drivers",
]  # CHROMIUM: virtwl: add virtwl driver
disp_overwrite[
    "d1a31b6250b7"
] = "drop"  # CHROMIUM: gsmi: Log event for critical thermal thresholds
disp_overwrite[
    "e71a49e36931"
] = "drop"  # CHROMIUM: drm/img-rogue: Add 1.17 IMG PowerVR Rogue driver
disp_overwrite[
    "e9f2ff6480f0"
] = "drop"  # CHROMIUM: img-rogue/1.17: remove misleading warning in buffer_sync
disp_overwrite[
    "48e297466ab4"
] = "drop"  # CHROMIUM: img-rogue/1.17: hide host trace buffer in release builds
disp_overwrite[
    "410de9affbf9"
] = "drop"  # CHROMIUM: img-rogue/1.17: fix UAF in DevmemIntUnmapPMR
disp_overwrite[
    "d9a0f3330f91"
] = "drop"  # CHROMIUM: img-rogue/1.17: security fixes in PMR related code
disp_overwrite[
    "e6ddd0772e09"
] = "drop"  # CHROMIUM: img-rogue/1.17: fix for UAF race condition in DevmemIntAcquireRemoteCtx
disp_overwrite[
    "26acfdc72631"
] = "drop"  # CHROMIUM: img-rogue/1.17: add refcounting for freelist's PMR resources
disp_overwrite[
    "0a02b878bebb"
] = "drop"  # CHROMIUM: img-rogue/1.17: guard against integer overflows in RGXCreateFreelist path
disp_overwrite[
    "c14ec7bd4429"
] = "drop"  # CHROMIUM: img-rogue/1.17: fix ref counting for ZS buffer resources
disp_overwrite[
    "6c1d65c7052f"
] = "drop"  # CHROMIUM: img-rogue/1.17: Do not map read-only PMRs as read-write
disp_overwrite[
    "a5852aa5f570"
] = "drop"  # CHROMIUM: img-rogue/1.17: Fix validation of ui32Log2DataPageSize
disp_overwrite[
    "cf75b05e4fb8"
] = "drop"  # CHROMIUM: img-rogue/1.17: validate PhysmemNewRamBackedPMR Flags
disp_overwrite[
    "38ca4c54ded0"
] = "drop"  # CHROMIUM: img-rogue/1.17: do not store pages if still referenced
disp_overwrite[
    "fd1f40df76f9"
] = "drop"  # CHROMIUM: img-rogue/1.17: fix for possible KE during mmap
disp_overwrite[
    "25beb4a4736f"
] = "drop"  # CHROMIUM: img-rogue/1.17: fix write OOB due to integer overflow
disp_overwrite[
    "79786fc523c1"
] = "drop"  # CHROMIUM: img-rogue/1.17: validate mapping smaller than PMR page size
disp_overwrite[
    "e470ac077c41"
] = "drop"  # CHROMIUM: img-rogue/1.17: potential UAF in Sparse memory handling
disp_overwrite[
    "36421f6d3104"
] = "drop"  # CHROMIUM: img-rogue/1.17: fix for UAF in the DevMemX framework
disp_overwrite[
    "4e86fa4a7b60"
] = "drop"  # CHROMIUM: img-rogue/1.17: improve validation in VA mapping calls
disp_overwrite[
    "b15899c241a1"
] = "drop"  # CHROMIUM: img-rogue/1.17: validate bridge {in,out}_buffer sizes
disp_overwrite[
    "afb3794ce882"
] = "drop"  # CHROMIUM: img-rogue/1.17: fix possible heap use-after-free condition
disp_overwrite[
    "0c8898a3f64c"
] = "drop"  # CHROMIUM: img-rogue/1.17: BridgeHandleDestroy
disp_overwrite[
    "1977e750e7fd"
] = "drop"  # CHROMIUM: img-rogue/1.17: fix potential kernel data leak
disp_overwrite[
    "04d3ff5ae710"
] = "drop"  # CHROMIUM: img-rogue/1.17: Enable PVR_LINUX_PHYSMEM_ZERO_ALL_PAGES
disp_overwrite[
    "0ecbc55924a0"
] = "drop"  # CHROMIUM: img-rogue: Mitigate OOB Kernel Write Vulnerability
disp_overwrite[
    "44beb64893c7"
] = "drop"  # CHROMIUM: img-rogue: Prevent overriding PMR R/W permission flags
disp_overwrite[
    "38e4c2574cc9"
] = "drop"  # CHROMIUM: img-rogue/1.17: hold the mmap lock when calling find_vma
disp_overwrite[
    "faaa8f128461"
] = "drop"  # CHROMIUM: img-rogue: Prevent Use-After-Free (UAF) Vulnerability
disp_overwrite[
    "b207b674631b"
] = "drop"  # CHROMIUM: img-rogue: Enhance Parameter Validation to Prevent OOB
disp_overwrite[
    "a7de0b00a0c0"
] = "drop"  # CHROMIUM: img-rogue: Fixed DIReadEntryKM refcount overflow
disp_overwrite[
    "1c6e16841d6a"
] = "drop"  # CHROMIUM: img-rogue: Security fix for RGXCreateFreeList
disp_overwrite[
    "b5bb5f4ae32c"
] = "drop"  # CHROMIUM: img-rogue/1.17: Remove SPARSE_REMAP_MEM support
disp_overwrite[
    "5c6bf9c2043c"
] = "drop"  # CHROMIUM: img-rogue/1.17: Fix for LeftOverLocals
disp_overwrite[
    "592638e71b4b"
] = "drop"  # CHROMIUM: img-rogue/1.17: Fix thermal zone name
