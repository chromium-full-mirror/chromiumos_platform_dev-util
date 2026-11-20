#!/usr/bin/env python3
# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Package filtered corresponding source files for an OpenWrt image build."""

import argparse
from collections.abc import Iterator
import os
from pathlib import Path
import re
import sys
import tarfile


CORE_FILES = (
    "Makefile",
    "rules.mk",
    "Config.in",
    "COPYING",
    ".config",
    "feeds.conf",
    "feeds.conf.default",
)
CORE_DIRS = ("LICENSES", "config", "include", "scripts", "toolchain", "files")
ALWAYS_INCLUDED_DL_GLOBS = ("linux-*.tar.*", "musl-*.tar.*", "gcc-*.tar.*")
EXCLUDED_DL_PREFIXES = ("linux-firmware-", "broadcom-wl-")
EXCLUDED_DIR_NAMES = frozenset({".git", "__pycache__"})
EXCLUDED_PATH_PATTERNS = (
    "as21xxx",
    "aqr10g",
    "an8855_sdk",
    "mtk_vct",
    "crypto-eip",
)
GIT_HASH_RE = re.compile(r"([~-])([0-9a-f]{9,40})(?=\.)")


def load_installed_packages(
    openwrt_dir: Path,
) -> tuple[set[str], set[str], list[Path]]:
    """Return installed package names, built target arches, and manifests."""
    manifests = sorted(openwrt_dir.glob("bin/targets/*/*/*.manifest"))
    if not manifests:
        raise RuntimeError(f"No target .manifest found under {openwrt_dir}")

    selected_pkgs: set[str] = set()
    target_arches: set[str] = set()

    for manifest in manifests:
        target_arches.add(manifest.relative_to(openwrt_dir).parts[2])
        for line in manifest.read_text(
            encoding="utf-8", errors="replace"
        ).splitlines():
            if " - " in line:
                selected_pkgs.add(line.split(" - ", 1)[0].strip())

    config_path = openwrt_dir / ".config"
    if config_path.is_file():
        for line in config_path.read_text(
            encoding="utf-8", errors="replace"
        ).splitlines():
            if line.startswith("CONFIG_PACKAGE_") and line.endswith("=y"):
                selected_pkgs.add(line[len("CONFIG_PACKAGE_") : -2])

    return selected_pkgs, target_arches, manifests


def parse_packageinfo(
    openwrt_dir: Path, selected_pkgs: set[str]
) -> tuple[set[str], set[Path]]:
    """Map installed packages to their dl/ tarballs and recipe directories."""
    pkginfo_path = openwrt_dir / "tmp/.packageinfo"
    if not pkginfo_path.is_file():
        raise RuntimeError(f"Missing package metadata file: {pkginfo_path}")

    source_names: set[str] = set()
    recipe_dirs: set[Path] = set()
    current_makefile: Path | None = None
    in_selected = False

    for line in pkginfo_path.read_text(
        encoding="utf-8", errors="replace"
    ).splitlines():
        if line.startswith("Source-Makefile: "):
            current_makefile = Path(line.split(": ", 1)[1].strip())
            in_selected = False
        elif line.startswith("Package: "):
            in_selected = line.split(": ", 1)[1].strip() in selected_pkgs
            if in_selected and current_makefile is not None:
                recipe_dirs.add(current_makefile.parent)
        elif line.startswith("Source: ") and in_selected:
            source_name = line.split(": ", 1)[1].strip()
            if source_name:
                source_names.add(source_name)

    return source_names, recipe_dirs


def select_dl_files(openwrt_dir: Path, source_names: set[str]) -> list[Path]:
    """Return sorted dl/ source tarballs for installed packages and kernel."""
    dl_dir = openwrt_dir / "dl"
    if not dl_dir.is_dir():
        raise RuntimeError(f"Missing OpenWrt download directory: {dl_dir}")

    dl_names = set(source_names)
    for name in source_names:
        # OpenWrt's rules.mk keeps full git hashes in tmp/.packageinfo (DUMP=1)
        # but abbreviates them to 8 chars in dl/ ('-' on 23.05, '~' on 24.10+).
        match = GIT_HASH_RE.search(name)
        if match:
            prefix = name[: match.start()]
            sep, sha = match.group(1), match.group(2)
            for path in dl_dir.glob(f"{prefix}{sep}{sha[:8]}.*"):
                dl_names.add(path.name)

    for pattern in ALWAYS_INCLUDED_DL_GLOBS:
        for path in dl_dir.glob(pattern):
            dl_names.add(path.name)

    files = [
        dl_dir / name
        for name in sorted(dl_names)
        if not name.startswith(EXCLUDED_DL_PREFIXES)
        and (dl_dir / name).is_file()
    ]
    if not files:
        raise RuntimeError(f"No matching source tarballs found in {dl_dir}")
    return files


def is_excluded(rel_path: str) -> bool:
    """Return True if rel_path matches a VCS, cache, or excluded pattern."""
    if any(part in EXCLUDED_DIR_NAMES for part in Path(rel_path).parts):
        return True
    lower = rel_path.lower()
    return any(pattern in lower for pattern in EXCLUDED_PATH_PATTERNS)


def iter_tree_files(openwrt_dir: Path, rel_dir: Path) -> Iterator[Path]:
    """Yield non-excluded files under openwrt_dir / rel_dir."""
    root_dir = openwrt_dir / rel_dir
    if not root_dir.is_dir() or is_excluded(str(rel_dir)):
        return

    for dirpath, dirnames, filenames in os.walk(root_dir, followlinks=False):
        current_dir = Path(dirpath)
        rel_cur = str(current_dir.relative_to(openwrt_dir))
        dirnames[:] = [
            name
            for name in sorted(dirnames)
            if not is_excluded(f"{rel_cur}/{name}")
        ]
        for filename in sorted(filenames):
            if not is_excluded(f"{rel_cur}/{filename}"):
                yield current_dir / filename


def iter_openwrt_files(
    openwrt_dir: Path,
    device: str,
    target_arches: set[str],
    recipe_dirs: set[Path],
) -> Iterator[Path]:
    """Yield filtered OpenWrt build scripts, configs, and installed recipes."""
    for filename in CORE_FILES:
        path = openwrt_dir / filename
        if path.is_file():
            yield path

    dirs_to_walk = [Path(name) for name in CORE_DIRS]
    dirs_to_walk.append(Path("configs") / device)
    dirs_to_walk.append(Path("target/linux/generic"))
    for arch in sorted(target_arches):
        dirs_to_walk.append(Path("target/linux") / arch)
    dirs_to_walk.extend(sorted(recipe_dirs))

    for rel_dir in dirs_to_walk:
        yield from iter_tree_files(openwrt_dir, rel_dir)

    # Include shared feed helper *.mk files in parent dirs of matched recipes.
    parent_dirs: set[Path] = set()
    for recipe_dir in recipe_dirs:
        real_parent = (openwrt_dir / recipe_dir).resolve().parent
        if real_parent != openwrt_dir and real_parent.is_relative_to(
            openwrt_dir
        ):
            parent_dirs.add(real_parent)

    for parent_dir in sorted(parent_dirs):
        for mk_file in sorted(parent_dir.glob("*.mk")):
            rel_mk = str(mk_file.relative_to(openwrt_dir))
            if mk_file.is_file() and not is_excluded(rel_mk):
                yield mk_file


def write_source_archive(
    openwrt_dir: Path,
    device: str,
    metadata_dir: Path,
    metadata_files: list[str],
    output_tar_gz: Path,
) -> tuple[int, int]:
    """Create the filtered source tarball and return (dl_count, total_files)."""
    openwrt_dir = openwrt_dir.resolve()
    selected_pkgs, target_arches, manifests = load_installed_packages(
        openwrt_dir
    )
    source_names, recipe_dirs = parse_packageinfo(openwrt_dir, selected_pkgs)
    dl_files = select_dl_files(openwrt_dir, source_names)

    entries: dict[str, Path] = {}
    for filename in metadata_files:
        meta_path = metadata_dir / filename
        if meta_path.is_file():
            entries[filename] = meta_path

    for manifest in manifests:
        entries[manifest.name] = manifest

    for path in iter_openwrt_files(
        openwrt_dir, device, target_arches, recipe_dirs
    ):
        entries[f"openwrt/{path.relative_to(openwrt_dir)}"] = path

    for dl_path in dl_files:
        entries[f"openwrt/dl/{dl_path.name}"] = dl_path

    output_tar_gz.parent.mkdir(parents=True, exist_ok=True)
    with tarfile.open(output_tar_gz, mode="w:gz", compresslevel=1) as archive:
        for arcname, path in entries.items():
            archive.add(path, arcname=arcname, recursive=False)

    return len(dl_files), len(entries)


def main(argv: list[str] | None = None) -> int:
    """Generate the filtered corresponding source archive."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("openwrt_dir", type=Path, help="OpenWrt build root.")
    parser.add_argument("device", help="Device config directory name.")
    parser.add_argument(
        "metadata_dir", type=Path, help="Directory holding build metadata."
    )
    parser.add_argument(
        "output_tar_gz", type=Path, help="Output .tar.gz archive path."
    )
    parser.add_argument(
        "metadata_files", nargs="*", help="Metadata filenames to include."
    )
    args = parser.parse_args(argv)

    dl_count, total_files = write_source_archive(
        openwrt_dir=args.openwrt_dir,
        device=args.device,
        metadata_dir=args.metadata_dir,
        metadata_files=args.metadata_files,
        output_tar_gz=args.output_tar_gz,
    )
    size_mib = args.output_tar_gz.stat().st_size / (1024 * 1024)
    print(
        f"Source archive created: {args.output_tar_gz} "
        f"({dl_count} dl/ tarballs, {total_files} files, {size_mib:.1f} MiB)"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
