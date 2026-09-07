# BananaPi-R4 Custom Patch Inventory

This document tracks all custom patches maintained in `configs/BananaPi-R4/` for feed updates and package builds, along with their upstream OpenWrt PR and commit references.

For details on the patch naming and numbering rules, see [`patches-package/README.md`](patches-package/README.md).

---

## 1. Feed Patches (`patches-feed/`)
These patches are applied directly to the `mtk-openwrt-feeds` git repository before running autobuild:

| File | Component | Description | Upstream PR / Reference |
| :--- | :--- | :--- | :--- |
| `hostapd_sh.patch` | `wifi-scripts` (`hostapd.sh`) | Use `json_for_each_item` to prevent word-splitting on spaces in `hostapd_options` | PR branch: [`fix-hostapd-options-json-for-each`](https://github.com/astrouski-google/openwrt/tree/fix-hostapd-options-json-for-each) |
| `mac80211_sh.patch` | `wifi-scripts` (`mac80211.sh`) | Fix 6 GHz 40 MHz channel configuration in `iw_htmode` | **Merged Upstream:** [OpenWrt PR #24843](https://github.com/openwrt/openwrt/pull/24843) |
| `common_uc.patch` | `wifi-scripts` (`common.uc`) | Add `prepare_key_wep()`, `hex:` prefix bypass in `escape_string()`, and `append_list()` | PR branch: [`wifi-scripts-add-wep-support`](https://github.com/astrouski-google/openwrt/tree/wifi-scripts-add-wep-support) |
| `iface_uc.patch` | `wifi-scripts` (`iface.uc`) | Add WEP encryption parsing (`wep`, `wep-open`, `wep-shared`, `auth_algs`) | PR branch: [`wifi-scripts-add-wep-support`](https://github.com/astrouski-google/openwrt/tree/wifi-scripts-add-wep-support) |
| `supplicant_uc.patch` | `wifi-scripts` (`supplicant.uc`) | Add WEP station configuration generation (`wep_key0`..`3`, `wep_tx_keyidx`, `auth_alg`, `key_mgmt=NONE`) | PR branch: [`wifi-scripts-add-wep-support`](https://github.com/astrouski-google/openwrt/tree/wifi-scripts-add-wep-support) |
| `hostap_common_uc.patch` | `hostapd` (`hostap.common.uc`) | Ucode link configuration updates | In-tree feed overlay |
| `wdev_uc.patch` | `wifi-scripts` (`wdev.uc`) | Only manage wdev interfaces recorded in statefile (drop `add_existing`) | **Backport:** [OpenWrt commit `e6bd2af652`](https://github.com/openwrt/openwrt/commit/e6bd2af652721a9c238a21a987e923333738c508) |
| `ap_uc.patch` | `wifi-scripts` (`ap.uc`, `wireless.wifi-iface.json`) | Add WEP AP configuration (`wep_key0`..`3`, `wep_default_key`, `wep_rekey_period`, schema options), `rnr_colocation`, `sae_pwe` by band, `append_list`, `transition_disable`, dynamic `group_mgmt_cipher`, `wmm_enabled=1`, `vendor_elements`, and `edcca_enable`/`edcca_compensation` UCI options | PR branch: [`wifi-scripts-add-wep-support`](https://github.com/astrouski-google/openwrt/tree/wifi-scripts-add-wep-support) / Backports |
| `hostapd_uc.patch` | `wifi-scripts` (`hostapd.uc`, `wireless.wifi-device.json`) | Align `EHT320` channelization and offset, fix `device_extended_features` bit test, fix VHT160/beamformee antenna count, protect HE center freq for 6E discovery, and add `edcca_enable`/`edcca_compensation` device options | **Backport:** [commit `a8bdb1e6d6`](https://github.com/openwrt/openwrt/commit/a8bdb1e6d6), [commit `28c6f5e6f7`](https://github.com/openwrt/openwrt/commit/28c6f5e6f7), [commit `fc5aed2ff2`](https://github.com/openwrt/openwrt/commit/fc5aed2ff2), [commit `02e2065203`](https://github.com/openwrt/openwrt/commit/02e2065203) |

---

## 2. Package Patches (`patches-package/`)

### `hostapd` (`patches-package/hostapd/`)
| Patch File | Type | Upstream / Tracking PR | Description |
| :--- | :--- | :--- | :--- |
| `1001-hostapd-add_require_eht.patch` | Feature | Downstream Feature | Add `require_eht` configuration option |
| `1002-hostapd-wpa_supplicant_add_eht_members.patch` | Feature | Downstream Feature | Add EHT station member tracking |
| `9002-hostapd-mlo-auth-reply-bssid.patch` | Bugfix | PR branch: [`fix_hostapd_mld_auth_bssid_and_unassoc_lookup`](https://github.com/astrouski-google/openwrt/tree/fix_hostapd_mld_auth_bssid_and_unassoc_lookup) | Set `reply->bssid` to `hapd->own_addr` in `send_auth_reply` |
| `9003-hostapd-defer-eapol-until-assoc.patch` | Bugfix | PR branch: [`hostapd-eapol-pmksa-fixes`](https://github.com/astrouski-google/openwrt/tree/hostapd-eapol-pmksa-fixes) | Defer EAPOL packet transmission until association completes |
| `9004-hostapd-pmksa-no-remove-on-bsstm.patch` | Bugfix | PR branch: [`hostapd-eapol-pmksa-fixes`](https://github.com/astrouski-google/openwrt/tree/hostapd-eapol-pmksa-fixes) | Prevent invalid PMKSA cache deletion on BSS Transition management |
| `9005-hostapd-pmksa-cache-set-portvalid.patch` | Bugfix | PR branch: [`hostapd-eapol-pmksa-fixes`](https://github.com/astrouski-google/openwrt/tree/hostapd-eapol-pmksa-fixes) | Require `portValid == true` after 4-way handshake on PMKSA cache transition |
| `9006-hostapd-pmksa-internal-eap-type-fix.patch` | Bugfix | PR branch: [`hostapd-eapol-pmksa-fixes`](https://github.com/astrouski-google/openwrt/tree/hostapd-eapol-pmksa-fixes) | Populate `eap_type_authsrv` from internal EAP server method |
| `9007-hostapd-mlo-skip-partner-link-ssid-and-wpa-ie-checks.patch` | Bugfix | PR branch: [`hostapd-mlo-partner-link-fixes`](https://github.com/astrouski-google/openwrt/tree/hostapd-mlo-partner-link-fixes) | Skip redundant SSID and WPA IE checks when parsing partner link profiles |
| `9008-hostapd-mlo-sync-partner-link-mfp-and-spp-amsdu.patch` | Bugfix | PR branch: [`hostapd-mlo-partner-link-fixes`](https://github.com/astrouski-google/openwrt/tree/hostapd-mlo-partner-link-fixes) | Propagate `WLAN_STA_MFP` and `WLAN_STA_SPP_AMSDU` flags from `assoc_wpa_sm` |
| `9009-hostapd-mlo-free-stale-partner-link-sta-on-reassoc.patch` | Bugfix | PR branch: [`hostapd-mlo-partner-link-fixes`](https://github.com/astrouski-google/openwrt/tree/hostapd-mlo-partner-link-fixes) | Clean up stale partner link stations during re-association |
| `9010-hostapd-mlo-open-auth-link-addr.patch` | Backport | Upstream OpenWrt Patch 192 | Do not address open authentication replies to MLD MAC |
| `9011-hostapd-mlo-sae-auth-link-addr.patch` | Backport | Upstream OpenWrt Patch 193 | Address SAE Authentication replies to peer link address |
| `9012-hostapd-mlo-tx-status-link-sta.patch` | Backport | Upstream OpenWrt Patch 194 | Fall back to link address lookup in TX status callbacks |
| `9013-hostapd-add-rnr-colocation-option.patch` | Feature | Downstream Feature | Add `rnr_colocation` option to suppress 6GHz RNR in lower-band beacons (b/389178440) |
| `9014-hostapd-mlo-flush-stale-link-sta-on-auth.patch` | Bugfix | PR branch: [`hostapd-mlo-partner-link-fixes`](https://github.com/astrouski-google/openwrt/tree/hostapd-mlo-partner-link-fixes) | Flush stale link station on receiving authentication frame |
| `9015-hostapd-fix-eht-mcs-validation-for-narrower-sta.patch` | Bugfix | PR branch: [`hostapd-fix-eht-mcs-validation-narrower-sta`](https://github.com/astrouski-google/openwrt/tree/hostapd-fix-eht-mcs-validation-narrower-sta) | Fix EHT MCS validation and out-of-bounds access for narrower bandwidth STAs on 320 MHz APs |

### `mt76` (`patches-package/mt76/`)
| Patch File | Type | Upstream / Tracking PR | Description |
| :--- | :--- | :--- | :--- |
| `9001-mt76-flush-stale-station-offload-and-wcid.patch` | Bugfix | PR branch: [`fix_sta_remove_purge_tx_queues`](https://github.com/astrouski-google/mt76/tree/fix_sta_remove_purge_tx_queues) | Call `mt76_wcid_cleanup()` before `dev->drv->sta_remove()` to flush pending TX queues |
| `9002-mt76-mt7996-fix-disable-inband-discovery.patch` | Bugfix | PR branch: [`fix_disable_inband_discovery`](https://github.com/astrouski-google/mt76/tree/fix_disable_inband_discovery) | Properly disable in-band discovery in MCU when interval is 0 (b/389178357) |
| `9003-mt76-mt7996-fix-beacon-and-inband-discovery-link-id.patch` | Bugfix | PR branch: [`fix_beacon_and_inband_discovery_link_id`](https://github.com/astrouski-google/mt76/tree/fix_beacon_and_inband_discovery_link_id) | Set correct link ID for continuous beacons / in-band discovery and add fallback in TXWI |
| `9004-mt76-mt7996-preserve-and-honor-edcca-enable.patch` | Bugfix | PR branch: [`fix_mt7996_preserve_edcca_enable`](https://github.com/astrouski-google/mt76/tree/fix_mt7996_preserve_edcca_enable) | Preserve `edcca_enable` state in `struct mt7996_phy` and honor it during `mt7996_set_channel` (b/552891389) |

### `mac80211` (`patches-package/mac80211/`)
| Patch File | Type | Upstream / Tracking PR | Description |
| :--- | :--- | :--- | :--- |
| `9001-mac80211-mlo-mgmt-tx-link-sta.patch` | Bugfix | **Submitted Upstream:** [OpenWrt PR #24880](https://github.com/openwrt/openwrt/pull/24880) | Fall back to `link_sta_info_get_bss` and honor `params->link_id` in `ieee80211_mgmt_tx` |

### `uboot-mediatek` (`patches-package/uboot-mediatek/`)
| Patch File | Type | Target | Description |
| :--- | :--- | :--- | :--- |
| `9001-add-hw-id.patch` | Board Quirk | `package/boot/uboot-mediatek` | Add board hardware ID detection for BananaPi-R4 |
