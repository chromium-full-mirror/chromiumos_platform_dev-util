# Package Patches Guide (`patches-package/`)

This directory contains target package patches applied during the OpenWrt image compilation.
Patches placed here are copied into the corresponding `package/.../patches/` directories during the autobuild process.

---

## Directory Organization

Each subdirectory matches the target package being patched:

```text
patches-package/
├── hostapd/                <-- package/network/services/hostapd
│   ├── 1001-hostapd-add_require_eht.patch
│   ├── 1002-hostapd-wpa_supplicant_add_eht_members.patch
│   ├── 9002-hostapd-mlo-auth-reply-bssid.patch
│   ├── 9003-hostapd-defer-eapol-until-assoc.patch
│   ├── 9004-hostapd-pmksa-no-remove-on-bsstm.patch
│   ├── 9005-hostapd-pmksa-cache-set-portvalid.patch
│   ├── 9006-hostapd-pmksa-internal-eap-type-fix.patch
│   ├── 9007-hostapd-mlo-skip-partner-link-ssid-and-wpa-ie-checks.patch
│   ├── 9008-hostapd-mlo-sync-partner-link-mfp-and-spp-amsdu.patch
│   ├── 9009-hostapd-mlo-free-stale-partner-link-sta-on-reassoc.patch
│   ├── 9010-hostapd-mlo-open-auth-link-addr.patch
│   ├── 9011-hostapd-mlo-sae-auth-link-addr.patch
│   ├── 9012-hostapd-mlo-tx-status-link-sta.patch
│   └── 9013-hostapd-add-rnr-colocation-option.patch
├── mac80211/               <-- package/kernel/mac80211/patches/subsys
│   └── 9001-mac80211-mlo-mgmt-tx-link-sta.patch
├── mt76/                   <-- package/kernel/mt76
│   ├── 9001-mt76-flush-stale-station-offload-and-wcid.patch
│   └── 9002-mt76-mt7996-fix-disable-inband-discovery.patch
└── uboot-mediatek/         <-- package/boot/uboot-mediatek
    └── 9001-add-hw-id.patch
```

---

## Numbering Convention

Patch filenames must follow the 4-digit prefix pattern: `NNNN-<component>-<description>.patch`.

| Prefix Range | Category | Description | Example |
| :--- | :--- | :--- | :--- |
| `0001`–`0899` | **Upstream / Vendor Backports** | Fixes cherry-picked from upstream OpenWrt or vendor feeds before the feed is bumped. | `0191-hostapd-add-support-...patch` |
| `9000`–`9999` | **Downstream Bugfixes & Quirks** | Bugfixes and board quirks applied after upstream package patches. | `9001-add-hw-id.patch` |
| `1000`–`8999` | **Downstream Features** | ChromeOS-specific features not intended for upstream OpenWrt. | `1001-hostapd-add_require_eht.patch` |
| `9000`–`9999` | **Downstream Bugfixes (PRs)** | Bugfixes developed locally and submitted upstream as PRs. | `9001-hostapd-do-not-send-AID-...patch` |

---

## Patch Header Requirements

Every patch file must include standard Git headers (`From:`, `Date:`, `Subject:`, `Upstream-Status:`):

```patch
From 0000000000000000000000000000000000000000 Mon Sep 17 00:00:00 2001
From: Author Name <author@google.com>
Date: Mon, 24 Aug 2026 10:00:00 +0000
Subject: [PATCH] component: concise description of change

Detailed explanation of why this patch is necessary, what failure it
addresses, and how it was tested.

Upstream-Status: Pending / Backport [commit <hash>] / Inappropriate [reason]
Signed-off-by: Author Name <author@google.com>
---
--- a/path/to/file.c
+++ b/path/to/file.c
@@ -100,6 +100,7 @@
```

---

## Adding a New Patch

1. Place the `.patch` file in `patches-package/<package_name>/`.
2. Follow the numbering convention based on patch category.
3. Update [`../PATCHES.md`](../PATCHES.md) to document the patch and upstream PR tracking link.
4. Run the pre-flight validator:
   ```bash
   ../../validate_patches.sh ../BananaPi-R4
   ```

---

## Retiring a Patch

When an upstream feed bump or PR merge makes a patch obsolete:
1. Run `../../check_patch_status.sh ../BananaPi-R4` to detect merged patches.
2. Remove the retired patch file from `patches-package/<package>/`.
3. Update `../PATCHES.md` to reflect the retirement.
