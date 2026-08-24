# BananaPi-R4 Custom Patch Inventory

This document tracks all custom patches maintained in `configs/BananaPi-R4/` for feed updates and package builds.

---

## 1. Feed Patches (`patches-feed/`)
These patches are applied directly to the `mtk-openwrt-feeds` git repository before running autobuild:

| File | Component | Description | Upstream PR / Status |
| :--- | :--- | :--- | :--- |
| `hostapd_sh.patch` | `wifi-scripts` (`hostapd.sh`) | Use `json_for_each_item` to prevent word-splitting on spaces in `hostapd_options` | PR submitted: `fix-hostapd-options-json-for-each` |
| `mac80211_sh.patch` | `wifi-scripts` (`mac80211.sh`) | Fix 6 GHz 40 MHz channel configuration in `iw_htmode` | Merged upstream (PR #24843) |
| `common_uc.patch` | `wifi-scripts` (`common.uc`) | Custom ucode script additions | In-tree feed overlay |
| `hostap_common_uc.patch` | `hostapd` (`hostap.common.uc`) | Ucode link configuration updates | In-tree feed overlay |
| `wdev_uc.patch` | `wifi-scripts` (`wdev.uc`) | Wireless device ucode helpers | In-tree feed overlay |

---

## 2. Package Patches (`patches-package/`)

### `hostapd` (`patches-package/hostapd/`)
| Patch File | Upstream / Tracking PR | Description |
| :--- | :--- | :--- |
| `9992-hostapd-do-not-send-AID-and-listen_interval-on-set-sta.patch` | `hostapd-nl80211-fix-sta-modify-associated` | Do not send AID/listen_interval on `SET_STATION` for associated clients |
| `9993-hostapd-mlo-auth-dest-link-mac.patch` | `fix_hostapd_mld_auth_bssid_and_unassoc_lookup` | Match unassociated MLD stations in `ap_get_link_sta` via `auth_link_addr` |
| `9994-hostapd-defer-eapol-until-assoc.patch` | `hostapd-eapol-pmksa-fixes` | Defer EAPOL packet transmission until association completes |
| `9995-hostapd-pmksa-no-remove-on-bsstm.patch` | `hostapd-eapol-pmksa-fixes` | Prevent invalid PMKSA cache deletion on BSS Transition management |
| `9996-hostapd-pmksa-cache-set-portvalid.patch` | `hostapd-eapol-pmksa-fixes` | Require `portValid == true` after 4-way handshake on PMKSA cache transition |
| `9997-hostapd-pmksa-internal-eap-type-fix.patch` | `hostapd-eapol-pmksa-fixes` | Populate `eap_type_authsrv` from internal EAP server method |
| `9998-hostapd-add_require_eht.patch` | Vendor Feature | Add `require_eht` configuration option |
| `9999-hostapd-wpa_supplicant_add_eht_members.patch` | Vendor Feature | Add EHT station member tracking |
| `99990-hostapd-mlo-partner-link-assoc.patch` | Pending upstream PR | Sync partner link station MFP/SPP-AMSDU flags and clean up stale partner STAs |

### `mt76` (`patches-package/mt76/`)
| Patch File | Upstream / Tracking PR | Description |
| :--- | :--- | :--- |
| `9990-mt76-flush-stale-station-offload-and-wcid.patch` | `astrouski-google/mt76#fix_sta_remove_purge_tx_queues` | Call `mt76_wcid_cleanup()` before `dev->drv->sta_remove()` to flush pending TX queues |

### `mac80211` (`patches-package/mac80211/`)
| Patch File | Upstream / Tracking PR | Description |
| :--- | :--- | :--- |
| `0140-mac80211-mlo-mgmt-tx-link-sta.patch` | `mac80211-mlo-mgmt-tx-link-sta` | Fall back to `link_sta_info_get_bss` and honor `params->link_id` in `ieee80211_mgmt_tx` |

### `uboot-mediatek` (`patches-package/uboot-mediatek/`)
| Patch File | Target | Description |
| :--- | :--- | :--- |
| `900-add-hw-id.patch` | `package/boot/uboot-mediatek` | Add board hardware ID detection for BananaPi-R4 |
