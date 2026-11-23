#!/usr/bin/env python3
# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Unit tests for collect_notices.py."""

from pathlib import Path
import tempfile
import unittest

import collect_notices


class CollectNoticesTest(unittest.TestCase):
    """Tests for installed-package filtering, parsing, and notice synthesis."""

    def _write(self, root: Path, rel: str, content: str) -> Path:
        path = root / rel
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")
        return path

    def test_notice_store_deduplicates_and_rejects_binary(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            p1 = self._write(root, "pkg_a/LICENSE", "MIT License Text\n")
            p2 = self._write(root, "pkg_b/COPYING", "MIT License Text\n")
            bin_file = root / "pkg_c/LICENSE"
            bin_file.parent.mkdir(parents=True, exist_ok=True)
            bin_file.write_bytes(b"\x00\x01binary")

            store = collect_notices.NoticeStore(root)
            self.assertTrue(store.add_file(p1))
            self.assertTrue(store.add_file(p2))
            self.assertFalse(store.add_file(bin_file))

            notices = store.sorted_notices()
            self.assertEqual(len(notices), 1)
            self.assertEqual(
                notices[0].source_paths, ["pkg_a/LICENSE", "pkg_b/COPYING"]
            )

    def test_parse_installed_recipes_merges_multi_package_stanzas(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self._write(
                root,
                "tmp/.packageinfo",
                "\n".join(
                    [
                        "Source-Makefile: package/libs/libubox/Makefile",
                        "Package: libubox",
                        "Provides: libubox20260213",
                        "License: ISC",
                        "Package: jshn",
                        "License: ISC",
                        "LicenseFiles: debian/copyright",
                    ]
                )
                + "\n",
            )
            recipes = collect_notices.parse_installed_recipes(
                root, {"libubox20260213", "jshn"}, set()
            )
            self.assertEqual(len(recipes), 1)
            self.assertEqual(
                recipes[0].recipe_dir, Path("package/libs/libubox")
            )
            self.assertEqual(recipes[0].packages, ["libubox", "jshn"])
            self.assertEqual(recipes[0].license_files, ["debian/copyright"])

    def test_collect_notices_filters_and_synthesizes(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self._write(root, "COPYING", "OpenWrt GPL-2.0 text\n")
            self._write(root, "LICENSES/ISC", "ISC template text\n")
            self._write(
                root,
                "files/usr/share/licenses/"
                "chromiumos-openwrt-customizations.LICENSE",
                "Copyright 2022-2026 The ChromiumOS Authors\nBSD-3-Clause\n",
            )
            self._write(
                root,
                "bin/targets/mediatek/filogic/openwrt.manifest",
                "\n".join(
                    [
                        "libacl - 2.3.2-r2",
                        "mt7988-wo-firmware - 20251125-r1",
                        "firewall4 - 2025.03.17-r2",
                        "cros-send-management-frame - 1.0.0-r1",
                    ]
                )
                + "\n",
            )
            self._write(
                root,
                "tmp/.packageinfo",
                "\n".join(
                    [
                        "Source-Makefile: package/feeds/packages/acl/Makefile",
                        "Package: libacl",
                        "License: LGPL-2.1 GPL-2.0",
                        "LicenseFiles: doc/COPYING doc/COPYING.LGPL",
                        "",
                        "Source-Makefile: "
                        "package/firmware/linux-firmware/Makefile",
                        "Package: mt7988-wo-firmware",
                        "LicenseFiles: LICENCE.mediatek",
                        "",
                        "Source-Makefile: "
                        "package/network/config/firewall4/Makefile",
                        "Package: firewall4",
                        "License: ISC",
                        "",
                        "Source-Makefile: "
                        "package/cros-send-management-frame/Makefile",
                        "Package: cros-send-management-frame",
                        "License: BSD-3-Clause",
                        "LicenseFiles: LICENSE",
                        "",
                        "Source-Makefile: "
                        "package/feeds/luci/luci-base/Makefile",
                        "Package: luci-base",
                        "License: Apache-2.0",
                    ]
                )
                + "\n",
            )

            target = "build_dir/target-aarch64_cortex-a53_musl"
            # acl: license files live under doc/
            (root / target / "acl-2.3.2/.pkgdir/libacl").mkdir(parents=True)
            self._write(
                root,
                f"{target}/acl-2.3.2/doc/COPYING.LGPL",
                "ACL LGPL-2.1 notice\n",
            )
            # linux-firmware: only LICENCE.mediatek should be collected
            (
                root
                / target
                / "linux-firmware-20251125/.pkgdir/mt7988-wo-firmware"
            ).mkdir(parents=True)
            self._write(
                root,
                f"{target}/linux-firmware-20251125/LICENCE.mediatek",
                "MediaTek Firmware License\n",
            )
            self._write(
                root,
                f"{target}/linux-firmware-20251125/LICENSE.nvidia",
                "Uninstalled Nvidia Firmware EULA\n",
            )
            # firewall4: no standalone LICENSE file, copyright in .uc header
            (root / target / "firewall4-2025.03.17/.pkgdir/firewall4").mkdir(
                parents=True
            )
            self._write(
                root,
                f"{target}/firewall4-2025.03.17/root/usr/share/ucode/fw4.uc",
                "// Copyright (C) 2021-2023 Jo-Philipp Wich <jo@mein.io>\n",
            )
            # cros-send-management-frame: nested version build_dir with LICENSE
            (
                root / target / "cros-send-management-frame/1.0.0/.pkgdir/"
                "cros-send-management-frame"
            ).mkdir(parents=True)
            self._write(
                root,
                f"{target}/cros-send-management-frame/1.0.0/LICENSE",
                "Copyright 2022 The ChromiumOS Authors\nJouni Malinen\n",
            )
            # Uninstalled package build_dir should be ignored
            (root / target / "luci-base/.pkgdir/luci-base").mkdir(parents=True)
            self._write(
                root,
                f"{target}/luci-base/LICENSE",
                "Uninstalled LuCI Apache Notice\n",
            )
            # Toolchain: keep musl, skip host-only gdb
            self._write(
                root,
                "build_dir/toolchain-aarch64/musl-1.2.5/COPYRIGHT",
                "musl MIT copyright\n",
            )
            self._write(
                root,
                "build_dir/toolchain-aarch64/gdb-16.3/COPYING3",
                "Host-only GDB GPL-3.0\n",
            )

            report = collect_notices.build_notice_report(root)
            self.assertEqual(len(report.manifest_sections), 1)
            output = collect_notices.render_notices(report)

            self.assertIn("ACL LGPL-2.1 notice", output)
            self.assertIn("MediaTek Firmware License", output)
            self.assertIn("Jo-Philipp Wich <jo@mein.io>", output)
            self.assertIn("Jouni Malinen", output)
            self.assertIn("Copyright 2022-2026 The ChromiumOS Authors", output)
            self.assertIn("musl MIT copyright", output)
            self.assertNotIn("Uninstalled Nvidia Firmware EULA", output)
            self.assertNotIn("Uninstalled LuCI Apache Notice", output)
            self.assertNotIn("Host-only GDB GPL-3.0", output)


if __name__ == "__main__":
    unittest.main()
