---
name: cros-openwrt-patcher
description: >-
  Create, edit, compile-test, and export downstream OpenWrt package patches using quilt
  inside Docker containers. Use whenever modifying or creating patches for hostapd, mt76,
  uboot-mediatek, mac80211, or other OpenWrt packages.
---

# ChromeOS OpenWrt Patch Manager (quilt automation)

## Overview
This skill provides an automated workflow to create, edit, verify, and export downstream OpenWrt patches using **`quilt`** inside the OpenWrt Docker container.

Using `quilt` inside the container solves common patch generation pitfalls:
1. **Live Tree Accuracy**: Patches are generated directly against the fully prepared OpenWrt source tree with all preceding vendor and upstream patches applied.
2. **Zero Syntax Errors**: GNU `patch` unified diff headers, hunk line counts, and context offsets are generated automatically by `quilt refresh`.
3. **Instant Compilation Feedback**: Verify that the package compiles cleanly with `make package/<pkg>/compile` in ~15 seconds without waiting for a full image build.
4. **Automated Export & Validation**: Exports the refreshed patch to `image_generator/configs/<device>/patches-package/<pkg>/` and triggers `./image_generator/validate_patches.sh` automatically.

---

## Patch Lifecycle & Commands

All operations are automated through the helper script:
```bash
python3 contrib/cros_openwrt_sources/skills/cros-openwrt-patcher/scripts/patch_manager.py <action> <device> <package> [args...]
```

### 1. Prepare Package Source Tree
Unpacks the package tarball and applies all existing patches with quilt tracking enabled:
```bash
python3 skills/cros-openwrt-patcher/scripts/patch_manager.py prepare BananaPi-R4 hostapd
```

### 2. Create a New Patch
Automatically assigns the next `9000`-series patch prefix and creates the quilt patch:
```bash
python3 skills/cros-openwrt-patcher/scripts/patch_manager.py new BananaPi-R4 hostapd mlo-sync-partner-link-fix
```
*Output: `9014-hostapd-mlo-sync-partner-link-fix.patch`*

### 3. Add Files to Patch & Edit
Add the target files you intend to modify:
```bash
python3 skills/cros-openwrt-patcher/scripts/patch_manager.py add BananaPi-R4 hostapd src/ap/ieee802_11.c
```
Then edit the file directly inside the container or using container tools.

### 4. Refresh Patch Diff
Regenerates the patch with exact hunk counts and canonical headers:
```bash
python3 skills/cros-openwrt-patcher/scripts/patch_manager.py refresh BananaPi-R4 hostapd
```

### 5. Test Compile Package
Compiles the package in the container to verify C syntax and header dependencies:
```bash
python3 skills/cros-openwrt-patcher/scripts/patch_manager.py compile BananaPi-R4 hostapd
```

### 6. Export and Validate
Copies the patch to `image_generator/configs/BananaPi-R4/patches-package/hostapd/` and runs pre-flight sequential dry-run validation:
```bash
python3 skills/cros-openwrt-patcher/scripts/patch_manager.py export BananaPi-R4 hostapd
```

---

## Downstream Patching Rules

1. **Numbering Scheme**:
   * `1000-1999`: Backports from upstream OpenWrt master/feeds.
   * `9000-9999`: ChromiumOS downstream custom fixes and experimental patches.
2. **Atomic Commits**: Keep each bug fix in a separate patch.
3. **Commit Headers**: Include standard Git email headers (`From`, `Date`, `Subject`, `Bug`, `Upstream-Status`).
4. **Documentation**: When adding a patch, update `image_generator/configs/<device>/UPGRADING.md` with upstream tracking details.
