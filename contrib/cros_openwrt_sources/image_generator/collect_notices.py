#!/usr/bin/env python3
# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Collect installed package manifest and license texts into notices files."""

import argparse
from collections.abc import Iterator
import dataclasses
import gzip
import hashlib
import os
from pathlib import Path
import sys


LICENSE_FILE_PREFIXES = (
    "LICENSE",
    "LICENCE",
    "COPYING",
    "COPYRIGHT",
    "NOTICE",
    "UNLICENSE",
    "WHENCE",
)
IGNORED_EXTENSIONS = (
    ".O",
    ".A",
    ".SO",
    ".C",
    ".H",
    ".SH",
    ".PY",
    ".PL",
    ".MK",
    ".PATCH",
    ".DIFF",
)
SKIPPED_DIR_PREFIXES = (
    "root-",
    "root.orig-",
    "ipkg-",
    "apk-",
    ".pkgdir",
    "tests",
    "test",
    "doc",
)
KERNEL_LICENSE_PATHS = (
    "COPYING",
    "LICENSES/preferred/GPL-2.0",
    "LICENSES/exceptions/Linux-syscall-note",
)

MAX_LICENSE_BYTES = 512 * 1024
MAX_BUILD_DIR_DEPTH = 3
MAX_HEADER_PATHS = 3


@dataclasses.dataclass
class LicenseNotice:
    """A unique license text and the relative paths where it was found."""

    text: str
    source_paths: list[str]


def is_readable_text_file(path: Path) -> bool:
    """Return True if path is a non-empty plain-text file under the size cap."""
    if not path.is_file():
        return False

    try:
        file_size = path.stat().st_size
        if file_size == 0 or file_size > MAX_LICENSE_BYTES:
            return False
        with path.open("rb") as handle:
            sample = handle.read(4096)
    except OSError:
        return False

    return b"\x00" not in sample


def has_license_filename(path: Path) -> bool:
    """Return True if path's filename looks like a license or notice file."""
    upper_name = path.name.upper()
    if upper_name.endswith(IGNORED_EXTENSIONS):
        return False

    for prefix in LICENSE_FILE_PREFIXES:
        if upper_name == prefix or upper_name.startswith(
            (f"{prefix}.", f"{prefix}-", f"{prefix}_")
        ):
            return True
    return False


def should_descend(dirname: str, depth: int) -> bool:
    """Return True if os.walk should enter dirname at the given depth."""
    if dirname.startswith(SKIPPED_DIR_PREFIXES):
        return False
    # Kernel license files are collected explicitly to avoid walking the full
    # kernel source tree.
    if depth >= 1 and dirname.startswith("linux-"):
        return False
    return True


def iter_package_build_files(base_dir: Path) -> Iterator[Path]:
    """Yield files under an OpenWrt target or toolchain build directory."""
    if not base_dir.is_dir():
        return

    base_depth = len(base_dir.parts)
    for dirpath, dirnames, filenames in os.walk(base_dir):
        current_dir = Path(dirpath)
        depth = len(current_dir.parts) - base_depth

        if depth >= MAX_BUILD_DIR_DEPTH:
            dirnames.clear()
        else:
            dirnames[:] = [
                name
                for name in sorted(dirnames)
                if should_descend(name, depth)
            ]

        for filename in sorted(filenames):
            yield current_dir / filename


def iter_license_files(openwrt_dir: Path) -> Iterator[Path]:
    """Yield license files across OpenWrt, Linux, and package build dirs."""
    # Known top-level OpenWrt and Linux kernel license paths.
    known_paths = [openwrt_dir / "COPYING"]

    licenses_dir = openwrt_dir / "LICENSES"
    if licenses_dir.is_dir():
        known_paths.extend(sorted(licenses_dir.rglob("*")))

    for kernel_dir in sorted(
        openwrt_dir.glob("build_dir/target-*/linux-*/linux-*")
    ):
        for rel_path in KERNEL_LICENSE_PATHS:
            known_paths.append(kernel_dir / rel_path)

    for path in known_paths:
        if is_readable_text_file(path):
            yield path

    # Package and toolchain license files discovered under build_dir/.
    for pattern in ("build_dir/target-*", "build_dir/toolchain-*"):
        for base_dir in sorted(openwrt_dir.glob(pattern)):
            for path in iter_package_build_files(base_dir):
                if has_license_filename(path) and is_readable_text_file(path):
                    yield path


def load_unique_notices(openwrt_dir: Path) -> list[LicenseNotice]:
    """Read and deduplicate license texts by SHA-256 content hash."""
    notices_by_hash: dict[str, LicenseNotice] = {}

    for path in iter_license_files(openwrt_dir):
        text = path.read_text(encoding="utf-8", errors="replace").strip()
        if not text:
            continue

        digest = hashlib.sha256(text.encode("utf-8")).hexdigest()
        rel_path = str(path.relative_to(openwrt_dir))

        notice = notices_by_hash.get(digest)
        if notice is None:
            notices_by_hash[digest] = LicenseNotice(
                text=text,
                source_paths=[rel_path],
            )
        elif rel_path not in notice.source_paths:
            notice.source_paths.append(rel_path)

    return sorted(
        notices_by_hash.values(),
        key=lambda notice: notice.source_paths[0],
    )


def format_source_header(source_paths: list[str]) -> str:
    """Format the source path summary line for a deduplicated notice."""
    shown = ", ".join(source_paths[:MAX_HEADER_PATHS])
    remaining = len(source_paths) - MAX_HEADER_PATHS
    if remaining > 0:
        return f"{shown} (+{remaining} more)"
    return shown


def collect_notices(openwrt_dir: Path) -> str:
    """Build the full THIRD_PARTY_NOTICES document for an OpenWrt build."""
    manifests = sorted(openwrt_dir.glob("bin/targets/*/*/*.manifest"))
    if not manifests:
        raise RuntimeError(f"No target .manifest found under {openwrt_dir}")

    sections = [
        "THIRD-PARTY SOFTWARE NOTICES AND INFORMATION\n\n"
        "INSTALLED PACKAGES\n"
    ]
    for manifest in manifests:
        manifest_text = manifest.read_text(
            encoding="utf-8", errors="replace"
        ).strip()
        sections.append(f"{manifest_text}\n")

    notices = load_unique_notices(openwrt_dir)
    if not notices:
        raise RuntimeError(f"No license files found under {openwrt_dir}")

    divider = "=" * 72
    for notice in notices:
        header = format_source_header(notice.source_paths)
        sections.append(
            f"\n{divider}\nSource files: {header}\n{divider}\n\n"
            f"{notice.text}\n"
        )

    return "".join(sections)


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    """Parse command-line arguments."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("openwrt_dir", type=Path, help="OpenWrt build root.")
    parser.add_argument(
        "output_txt", type=Path, help="Output path for plain-text notices."
    )
    parser.add_argument(
        "output_gz", type=Path, help="Output path for gzipped notices."
    )
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    """Generate plain-text and gzipped THIRD_PARTY_NOTICES files."""
    args = parse_args(argv)
    data = collect_notices(args.openwrt_dir).encode("utf-8")

    args.output_txt.parent.mkdir(parents=True, exist_ok=True)
    args.output_txt.write_bytes(data)

    args.output_gz.parent.mkdir(parents=True, exist_ok=True)
    args.output_gz.write_bytes(gzip.compress(data, compresslevel=9, mtime=0))

    print(
        f"Wrote third-party notices ({len(data)} bytes) to "
        f"{args.output_txt} and {args.output_gz}"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
