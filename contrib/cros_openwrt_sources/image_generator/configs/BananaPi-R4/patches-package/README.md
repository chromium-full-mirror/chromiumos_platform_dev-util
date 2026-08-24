# Package Patch Naming & Organization Convention

This directory contains package build patches for BananaPi-R4, categorized by target package.

OpenWrt's build system applies patches in **alphabetical (numerical) sort order**. To ensure predictable patch ordering, avoid collisions with vendor feeds, and simplify upstream tracking, all patches follow this standardized numbering convention.

---

## 1. Numbering Scheme

| Prefix Range | Category | Description | Example |
| :--- | :--- | :--- | :--- |
| **`0001` – `0899`** | **Upstream / Vendor Backports** | Backports from newer upstream OpenWrt, kernel, or MediaTek feeds. | `0100-backport-eht-rates.patch` |
| **`0900` – `0999`** | **Hardware / Board Quirks** | Permanent device-specific fixes (e.g., board ID detection, PHY calibration). | `0900-add-hw-id.patch` |
| **`1000` – `8999`** | **Downstream Features** | Long-term ChromiumOS / lab-specific custom features not intended for upstream OpenWrt. | `1001-hostapd-add_require_eht.patch` |
| **`9000` – `9999`** | **Downstream Bugfixes (Under Upstream Review)** | Temporary bugfixes created for ChromiumOS tests that have active upstream Pull Requests. **These patches must be deleted when their upstream PR merges.** | `9005-hostapd-pmksa-cache-set-portvalid.patch` |

---

## 2. Directory Structure

Each subfolder matches the target OpenWrt package path:

```text
patches-package/
├── README.md               <-- This documentation
├── hostapd/                <-- package/network/services/hostapd
│   ├── 1001-hostapd-add_require_eht.patch
│   ├── 1002-hostapd-wpa_supplicant_add_eht_members.patch
│   ├── 9001-hostapd-do-not-send-AID-and-listen_interval-on-set-sta.patch
│   ├── 9002-hostapd-mlo-auth-dest-link-mac.patch
│   ├── 9003-hostapd-defer-eapol-until-assoc.patch
│   ├── 9004-hostapd-pmksa-no-remove-on-bsstm.patch
│   ├── 9005-hostapd-pmksa-cache-set-portvalid.patch
│   ├── 9006-hostapd-pmksa-internal-eap-type-fix.patch
│   ├── 9007-hostapd-mlo-skip-partner-link-ssid-and-wpa-ie-checks.patch
│   ├── 9008-hostapd-mlo-sync-partner-link-mfp-and-spp-amsdu.patch
│   └── 9009-hostapd-mlo-free-stale-partner-link-sta-on-reassoc.patch
├── mac80211/               <-- package/kernel/mac80211/patches/subsys
│   └── 9001-mac80211-mlo-mgmt-tx-link-sta.patch
├── mt76/                   <-- package/kernel/mt76
│   └── 9001-mt76-flush-stale-station-offload-and-wcid.patch
└── uboot-mediatek/         <-- package/boot/uboot-mediatek
    └── 0900-add-hw-id.patch
```

---

## 3. Patch Header Standard

Every patch file should include standard metadata headers at the top:

```patch
From: Author Name <email@example.com>
Subject: [PATCH] package: short description of change

Detailed explanation of why this fix is needed and what issue it resolves.

Upstream-Status: Submitted | Backport | Inappropriate | Merged
Upstream-PR: https://github.com/openwrt/openwrt/pull/XXXXX
---
--- a/path/to/file.c
+++ b/path/to/file.c
@@ ... @@
```

---

## 4. How to Add a New Patch

1. Identify the target package (`hostapd`, `mt76`, `mac80211`, `uboot-mediatek`).
2. Pick the next available number in the appropriate range:
   * If submitting an upstream PR: use `9000+`
   * If adding a permanent local feature: use `1000+`
   * If adding a hardware quirk: use `0900+`
3. Save the patch in `patches-package/<package>/<number>-<package>-<description>.patch`.
4. Update `PATCHES.md` with the patch filename, description, and PR link.
5. Run `./image_generator/validate_patches.sh configs/BananaPi-R4` to confirm it applies cleanly with 0 errors.

---

## 5. How to Retire a Merged Patch

When an upstream PR is merged:
1. Run `./image_generator/check_patch_status.sh configs/BananaPi-R4` (it will mark the patch as `[RETIRED]`).
2. `git rm patches-package/<package>/<patch-name>.patch`
3. Update `PATCHES.md`.
4. Re-run `./image_generator/validate_patches.sh configs/BananaPi-R4` to verify the build remains clean.
