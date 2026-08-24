# OpenWrt & MediaTek Feed Version Bump Playbook

This document describes the standard procedure for upgrading the OpenWrt and `mtk-openwrt-feeds` commit revisions for BananaPi-R4.

---

## 1. Upstream Sync & Status Check

Before bumping revisions or modifying patches, check the status of all local patches against the current tree:

```bash
./image_generator/check_patch_status.sh configs/BananaPi-R4
```

This will identify:
* **`[RETIRED]`**: Patches that are already merged in upstream commit history (safe to delete).
* **`[ACTIVE]`**: Patches that still apply cleanly as downstream overlays.
* **`[CONFLICT]`**: Patches that require rebasing against newer upstream code.

---

## 2. Bumping Revisions in `scripts/BananaPi-R4.sh`

Edit `scripts/BananaPi-R4.sh` with the new commit SHAs:

```bash
OPENWRT_COMMIT="<new-openwrt-sha>"
MTK_OPENWRT_FEEDS_COMMIT="<new-mtk-feeds-sha>"
```

---

## 3. Retiring Merged Patches

When an upstream Pull Request is merged into OpenWrt or MediaTek feeds:
1. Delete the corresponding `.patch` file from `configs/BananaPi-R4/patches-feed/` or `configs/BananaPi-R4/patches-package/<pkg>/`.
2. Update the tracking table in `configs/BananaPi-R4/PATCHES.md`.
3. Re-run `./image_generator/check_patch_status.sh configs/BananaPi-R4` to verify 0 missing dependencies.

---

## 4. Pre-Flight Validation

Run the pre-flight validator to ensure all remaining patches apply with 0 errors:

```bash
./image_generator/validate_patches.sh configs/BananaPi-R4
```

If a patch fails with hunk rejects:
1. Rebase the patch against the updated package source.
2. Update the patch file in `patches-package/<pkg>/` or `patches-feed/`.
3. Re-run `validate_patches.sh` until all checks report `[PASS]`.

---

## 5. Build & Image Verification

Build the image in Docker:

```bash
./scripts/BananaPi-R4.sh
```

Flash the generated sysupgrade image to the target testbed router and verify:
* SSH connectivity (`ssh -i ~/.ssh/testing_rsa root@<ip>`)
* `hostapd` initialization (`logread -e hostapd`)
* Wireless interfaces UP (`iw dev`)
