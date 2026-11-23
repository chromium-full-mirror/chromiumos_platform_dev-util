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
import re
import sys


# =============================================================================
# 1. Config & Policy Constants
# =============================================================================

# Filename & scan rules
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
    ".GZ",
)
SKIPPED_DIR_PREFIXES = (
    "root-",
    "root.orig-",
    "ipkg-",
    "apk-",
    ".pkgdir",
    "tests",
    "test",
)
HEADER_SCAN_EXTENSIONS = frozenset(
    {
        ".c",
        ".cc",
        ".cpp",
        ".h",
        ".hh",
        ".hpp",
        ".py",
        ".sh",
        ".uc",
        ".lua",
        ".pl",
    }
)
HEADER_SCAN_FILENAMES = frozenset({"Makefile", "CMakeLists.txt"})

COPYRIGHT_LINE_RE = re.compile(
    r"(?:Copyright(?:\s*\([cC]\)|\s*©)?\s+[^*/\r\n]{4,140}|"
    r"SPDX-License-Identifier:\s*[^*/\r\n]{2,80})",
    re.IGNORECASE,
)
ARCHIVE_EXT_RE = re.compile(r"\.(?:tar\.[^.]+|tgz|zip|xz|gz|bz2|zst)$")
GIT_HASH_RE = re.compile(r"([~-])([0-9a-f]{9,40})$")
ABI_SUFFIX_RE = re.compile(r"^([a-zA-Z0-9_-]*?[a-zA-Z_-])[0-9.-]+$")

# Build-tree special cases
ALLOWED_TOOLCHAIN_PREFIXES = (
    "musl-",
    "gcc-",
    "linux-",
    "fortify-headers-",
)
KERNEL_LICENSE_PATHS = (
    "COPYING",
    "LICENSES/preferred/GPL-2.0",
    "LICENSES/exceptions/Linux-syscall-note",
)
LINUX_FIRMWARE_RECIPE = Path("package/firmware/linux-firmware")
EXPLICITLY_COVERED_RECIPES = frozenset(
    {
        Path("package/kernel/linux"),
        Path("package/libs/toolchain"),
    }
)

# Scan & formatting limits
MAX_LICENSE_BYTES = 512 * 1024
MAX_HEADER_SCAN_BYTES = 8 * 1024
MAX_BUILD_DIR_DEPTH = 3
MAX_HEADER_SCAN_DEPTH = 6
MAX_HEADER_PATHS = 3


# =============================================================================
# 2. Data Models & NoticeStore
# =============================================================================


@dataclasses.dataclass
class LicenseNotice:
    """A unique license text and the relative paths where it was found."""

    text: str
    source_paths: list[str]


@dataclasses.dataclass
class InstalledRecipe:
    """Metadata for a single installed OpenWrt source recipe."""

    recipe_dir: Path
    packages: list[str] = dataclasses.field(default_factory=list)
    source: str = ""
    source_name: str = ""
    license_expr: str = ""
    license_files: list[str] = dataclasses.field(default_factory=list)


@dataclasses.dataclass(frozen=True)
class NoticeReport:
    """Collected package manifests and deduplicated license notices."""

    manifest_sections: list[str]
    notices: list[LicenseNotice]


@dataclasses.dataclass
class _PackageStanza:
    """A single Package stanza parsed from OpenWrt's tmp/.packageinfo."""

    makefile: Path
    package: str
    provides: set[str] = dataclasses.field(default_factory=set)
    source: str = ""
    source_name: str = ""
    license_expr: str = ""
    license_files: list[str] = dataclasses.field(default_factory=list)


class NoticeStore:
    """Deduplicates collected license texts by SHA-256 content hash."""

    def __init__(self, openwrt_dir: Path) -> None:
        self._openwrt_dir = openwrt_dir
        self._notices_by_hash: dict[str, LicenseNotice] = {}

    def add_text(self, text: str, rel_path: str) -> None:
        """Insert or deduplicate notice text under rel_path."""
        stripped = text.strip()
        if not stripped:
            return
        digest = hashlib.sha256(stripped.encode("utf-8")).hexdigest()
        notice = self._notices_by_hash.get(digest)
        if notice is None:
            self._notices_by_hash[digest] = LicenseNotice(
                text=stripped,
                source_paths=[rel_path],
            )
        elif rel_path not in notice.source_paths:
            notice.source_paths.append(rel_path)

    def add_file(self, path: Path) -> bool:
        """Read a plain-text license file once and record it if readable."""
        text = _read_text_file(path, max_bytes=MAX_LICENSE_BYTES)
        if text is None:
            return False
        rel_path = str(path.relative_to(self._openwrt_dir))
        self.add_text(text, rel_path)
        return True

    def sorted_notices(self) -> list[LicenseNotice]:
        """Return deduplicated notices ordered by their primary source path."""
        return sorted(
            self._notices_by_hash.values(),
            key=lambda notice: notice.source_paths[0],
        )


# =============================================================================
# 3. Shared Filesystem Utilities
# =============================================================================


def _read_text_file(
    path: Path,
    *,
    max_bytes: int | None = None,
    read_limit: int | None = None,
) -> str | None:
    """Read a plain-text file once; return None if binary or unreadable."""
    if not path.is_file():
        return None
    try:
        file_size = path.stat().st_size
        if file_size == 0:
            return None
        if max_bytes is not None and file_size > max_bytes:
            return None
        with path.open("rb") as handle:
            raw = (
                handle.read(read_limit)
                if read_limit is not None
                else handle.read()
            )
    except OSError:
        return None

    if not raw or b"\x00" in raw[:4096]:
        return None
    return raw.decode("utf-8", errors="replace")


def is_readable_text_file(path: Path) -> bool:
    """Return True if path is a non-empty plain-text file under the size cap."""
    return (
        _read_text_file(path, max_bytes=MAX_LICENSE_BYTES, read_limit=4096)
        is not None
    )


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


def should_descend(
    dirname: str,
    depth: int,
    *,
    skip_nested_linux: bool = True,
) -> bool:
    """Return True if os.walk should enter dirname at the given depth."""
    if dirname.startswith(SKIPPED_DIR_PREFIXES):
        return False
    # Kernel license files are collected explicitly to avoid walking the full
    # kernel source tree.
    if skip_nested_linux and depth >= 1 and dirname.startswith("linux-"):
        return False
    return True


def _walk_files(
    root_dir: Path,
    *,
    max_depth: int,
    skip_nested_linux: bool = False,
) -> Iterator[Path]:
    """Yield files under root_dir up to max_depth, pruning skipped subdirs."""
    if not root_dir.is_dir():
        return

    base_depth = len(root_dir.parts)
    for dirpath, dirnames, filenames in os.walk(root_dir):
        current_dir = Path(dirpath)
        depth = len(current_dir.parts) - base_depth

        if depth >= max_depth:
            dirnames.clear()
        else:
            dirnames[:] = [
                name
                for name in sorted(dirnames)
                if should_descend(
                    name, depth, skip_nested_linux=skip_nested_linux
                )
            ]

        for filename in sorted(filenames):
            yield current_dir / filename


def iter_dir_license_files(
    root_dir: Path, max_depth: int = MAX_BUILD_DIR_DEPTH
) -> Iterator[Path]:
    """Yield candidate license files within a package build directory."""
    for path in _walk_files(
        root_dir, max_depth=max_depth, skip_nested_linux=True
    ):
        if has_license_filename(path) and is_readable_text_file(path):
            yield path


# =============================================================================
# 4. OpenWrt Metadata Parsing
# =============================================================================


def load_installed_packages(
    openwrt_dir: Path,
) -> tuple[set[str], set[str], list[Path]]:
    """Return manifest packages, .config =y packages, and manifest paths."""
    manifests = sorted(openwrt_dir.glob("bin/targets/*/*/*.manifest"))
    if not manifests:
        raise RuntimeError(f"No target .manifest found under {openwrt_dir}")

    manifest_pkgs: set[str] = set()
    for manifest in manifests:
        for line in manifest.read_text(
            encoding="utf-8", errors="replace"
        ).splitlines():
            if " - " in line:
                manifest_pkgs.add(line.split(" - ", 1)[0].strip())

    config_pkgs: set[str] = set()
    config_path = openwrt_dir / ".config"
    if config_path.is_file():
        for line in config_path.read_text(
            encoding="utf-8", errors="replace"
        ).splitlines():
            if line.startswith("CONFIG_PACKAGE_") and line.endswith("=y"):
                config_pkgs.add(line[len("CONFIG_PACKAGE_") : -2])

    return manifest_pkgs, config_pkgs, manifests


def _iter_packageinfo_stanzas(pkginfo_path: Path) -> Iterator[_PackageStanza]:
    """Yield parsed Package stanzas from OpenWrt's tmp/.packageinfo."""
    current_makefile: Path | None = None
    stanza: _PackageStanza | None = None

    for line in pkginfo_path.read_text(
        encoding="utf-8", errors="replace"
    ).splitlines():
        if line.startswith("Source-Makefile: "):
            if stanza is not None:
                yield stanza
                stanza = None
            current_makefile = Path(line.split(": ", 1)[1].strip())
        elif line.startswith("Package: "):
            if stanza is not None:
                yield stanza
                stanza = None
            if current_makefile is not None:
                stanza = _PackageStanza(
                    makefile=current_makefile,
                    package=line.split(": ", 1)[1].strip(),
                )
        elif stanza is not None:
            if line.startswith("Provides: "):
                stanza.provides.update(
                    tok
                    for tok in line.split(": ", 1)[1].split()
                    if not tok.startswith("@")
                )
            elif line.startswith("Source: "):
                stanza.source = line.split(": ", 1)[1].strip()
            elif line.startswith("SourceName: "):
                stanza.source_name = line.split(": ", 1)[1].strip()
            elif line.startswith("License: "):
                stanza.license_expr = line.split(": ", 1)[1].strip()
            elif line.startswith(("LicenseFiles: ", "License-Files: ")):
                stanza.license_files.extend(line.split(": ", 1)[1].split())

    if stanza is not None:
        yield stanza


def _matches_selected(
    stanza: _PackageStanza,
    selected_pkgs: set[str],
    stripped_selected: set[str],
) -> bool:
    """Return True if a tmp/.packageinfo stanza matches an installed package."""
    if stanza.package in selected_pkgs or stanza.package in stripped_selected:
        return True
    return bool(stanza.provides & selected_pkgs)


def parse_installed_recipes(
    openwrt_dir: Path,
    manifest_pkgs: set[str],
    config_pkgs: set[str],
) -> list[InstalledRecipe]:
    """Parse tmp/.packageinfo into installed source recipes."""
    pkginfo_path = openwrt_dir / "tmp/.packageinfo"
    if not pkginfo_path.is_file():
        return []

    selected_pkgs = manifest_pkgs | config_pkgs
    stripped_selected = {
        match.group(1).rstrip("-")
        for name in manifest_pkgs
        if (match := ABI_SUFFIX_RE.match(name))
    }

    recipes_by_dir: dict[Path, InstalledRecipe] = {}
    for stanza in _iter_packageinfo_stanzas(pkginfo_path):
        if not _matches_selected(stanza, selected_pkgs, stripped_selected):
            continue
        recipe_dir = stanza.makefile.parent
        recipe = recipes_by_dir.get(recipe_dir)
        if recipe is None:
            recipe = InstalledRecipe(recipe_dir=recipe_dir)
            recipes_by_dir[recipe_dir] = recipe
        if stanza.package not in recipe.packages:
            recipe.packages.append(stanza.package)
        if stanza.source and not recipe.source:
            recipe.source = stanza.source
        if stanza.source_name and not recipe.source_name:
            recipe.source_name = stanza.source_name
        if stanza.license_expr and not recipe.license_expr:
            recipe.license_expr = stanza.license_expr
        for lic_file in stanza.license_files:
            if lic_file not in recipe.license_files:
                recipe.license_files.append(lic_file)

    return [recipes_by_dir[k] for k in sorted(recipes_by_dir)]


def index_pkgdirs(openwrt_dir: Path) -> dict[str, list[Path]]:
    """Map binary package names to their PKG_BUILD_DIR via .pkgdir markers."""
    pkg_to_dirs: dict[str, list[Path]] = {}
    for target_dir in sorted(openwrt_dir.glob("build_dir/target-*")):
        if not target_dir.is_dir():
            continue
        for pattern in ("*/.pkgdir/*", "*/*/.pkgdir/*", "*/*/*/.pkgdir/*"):
            for marker in sorted(target_dir.glob(pattern)):
                if not marker.is_dir():
                    continue
                build_root = marker.parent.parent
                dirs = pkg_to_dirs.setdefault(marker.name, [])
                if build_root not in dirs:
                    dirs.append(build_root)
    return pkg_to_dirs


def _match_recipe_build_dirs(
    openwrt_dir: Path,
    recipe: InstalledRecipe,
    pkg_to_dirs: dict[str, list[Path]],
) -> list[Path]:
    """Resolve the unpacked build directory(ies) for an installed recipe."""
    matched: list[Path] = []
    for pkg in recipe.packages:
        for bdir in pkg_to_dirs.get(pkg, ()):
            if bdir not in matched:
                matched.append(bdir)
    if matched:
        return matched

    # Fallback when .pkgdir markers are absent: match by source stem or name.
    candidates: set[str] = {recipe.recipe_dir.name}
    if recipe.source_name:
        candidates.add(recipe.source_name)
    if recipe.source:
        stem = ARCHIVE_EXT_RE.sub("", recipe.source)
        candidates.add(stem)
        candidates.add(
            GIT_HASH_RE.sub(lambda m: f"{m.group(1)}{m.group(2)[:8]}", stem)
        )

    for target_dir in sorted(openwrt_dir.glob("build_dir/target-*")):
        if not target_dir.is_dir():
            continue
        search_roots = [target_dir] + [
            sub for sub in sorted(target_dir.glob("linux-*")) if sub.is_dir()
        ]
        for root in search_roots:
            for entry in sorted(root.iterdir()):
                if not entry.is_dir() or entry.name.startswith(
                    ("root-", "root.orig-", "linux-")
                ):
                    continue
                if entry.name in candidates or any(
                    entry.name.startswith(f"{name}-")
                    for name in candidates
                    if name
                ):
                    if entry not in matched:
                        matched.append(entry)
    return matched


# =============================================================================
# 5. License Discovery, Rendering & CLI
# =============================================================================


def _collect_explicit_license_files(
    build_dirs: list[Path],
    license_files: list[str],
) -> list[Path]:
    """Resolve PKG_LICENSE_FILES relative paths/globs inside build_dirs."""
    found: list[Path] = []
    for bdir in build_dirs:
        for rel_spec in license_files:
            matches = sorted(bdir.glob(rel_spec))
            if not matches:
                candidate = bdir / rel_spec
                if candidate.is_file():
                    matches = [candidate]
            for path in matches:
                if path.is_file() and path not in found:
                    found.append(path)
    return found


def _extract_copyright_headers(search_dirs: list[Path]) -> list[str]:
    """Extract deduplicated Copyright / SPDX header lines from source files."""
    notices: list[str] = []
    seen: set[str] = set()

    for root_dir in search_dirs:
        for path in _walk_files(root_dir, max_depth=MAX_HEADER_SCAN_DEPTH):
            if (
                path.suffix.lower() not in HEADER_SCAN_EXTENSIONS
                and path.name not in HEADER_SCAN_FILENAMES
            ):
                continue
            text = _read_text_file(path, read_limit=MAX_HEADER_SCAN_BYTES)
            if text is None:
                continue
            for match in COPYRIGHT_LINE_RE.finditer(text):
                cleaned = " ".join(match.group(0).strip(" \t#*;/").split())
                if cleaned and cleaned not in seen:
                    seen.add(cleaned)
                    notices.append(cleaned)
    return notices


def _synthesize_recipe_notice(
    openwrt_dir: Path,
    recipe: InstalledRecipe,
    build_dirs: list[Path],
) -> LicenseNotice | None:
    """Synthesize a notice from source headers for packages lacking LICENSE."""
    if recipe.recipe_dir in EXPLICITLY_COVERED_RECIPES:
        return None

    search_dirs = list(build_dirs)
    recipe_path = openwrt_dir / recipe.recipe_dir
    if recipe_path.is_dir():
        search_dirs.append(recipe_path)

    headers = _extract_copyright_headers(search_dirs)
    if not headers and not recipe.license_expr:
        return None

    lines = [
        f"Package(s): {', '.join(sorted(recipe.packages))}",
        f"Recipe: {recipe.recipe_dir}",
    ]
    if recipe.license_expr:
        lines.append(f"Declared License: {recipe.license_expr}")
    if headers:
        lines.append("")
        lines.append("Extracted Copyright / License Headers:")
        for header in headers:
            lines.append(f"  {header}")
    if recipe.license_expr:
        lines.append("")
        lines.append(
            f"(Full license text for {recipe.license_expr} is included in the "
            "OpenWrt COPYING / LICENSES sections of this document.)"
        )
    return LicenseNotice(
        text="\n".join(lines),
        source_paths=[str(recipe.recipe_dir)],
    )


def _iter_toolchain_license_files(openwrt_dir: Path) -> Iterator[Path]:
    """Yield license files from runtime-relevant toolchain build dirs only."""
    for base_dir in sorted(openwrt_dir.glob("build_dir/toolchain-*")):
        if not base_dir.is_dir():
            continue
        for entry in sorted(base_dir.iterdir()):
            if entry.is_dir() and entry.name.startswith(
                ALLOWED_TOOLCHAIN_PREFIXES
            ):
                yield from _walk_files(
                    entry, max_depth=MAX_BUILD_DIR_DEPTH, skip_nested_linux=True
                )


def _iter_known_license_files(openwrt_dir: Path) -> Iterator[Path]:
    """Yield top-level OpenWrt, Linux kernel, and custom_files license paths."""
    known_paths = [openwrt_dir / "COPYING"]

    licenses_dir = openwrt_dir / "LICENSES"
    if licenses_dir.is_dir():
        known_paths.extend(sorted(licenses_dir.rglob("*")))

    custom_licenses_dir = openwrt_dir / "files/usr/share/licenses"
    if custom_licenses_dir.is_dir():
        for path in sorted(custom_licenses_dir.rglob("*")):
            if not path.name.startswith("THIRD_PARTY_NOTICES"):
                known_paths.append(path)

    for kernel_dir in sorted(
        openwrt_dir.glob("build_dir/target-*/linux-*/linux-*")
    ):
        for rel_path in KERNEL_LICENSE_PATHS:
            known_paths.append(kernel_dir / rel_path)

    yield from known_paths


def _collect_recipe_notices(
    store: NoticeStore,
    openwrt_dir: Path,
    recipe: InstalledRecipe,
    pkg_to_dirs: dict[str, list[Path]],
) -> None:
    """Collect license files or synthesized header notice for one recipe."""
    build_dirs = _match_recipe_build_dirs(openwrt_dir, recipe, pkg_to_dirs)
    explicit_files = _collect_explicit_license_files(
        build_dirs, recipe.license_files
    )

    added_any = False
    for path in explicit_files:
        if store.add_file(path):
            added_any = True

    # For linux-firmware, collect only declared PKG_LICENSE_FILES (e.g.
    # LICENCE.mediatek) to avoid ingesting ~120 uninstalled third-party vendor
    # firmware EULAs from the full archive.
    is_linux_firmware = recipe.recipe_dir == LINUX_FIRMWARE_RECIPE or any(
        d.name.startswith("linux-firmware-") for d in build_dirs
    )
    if not (is_linux_firmware and added_any):
        seen_explicit = set(explicit_files)
        for bdir in build_dirs:
            for path in _walk_files(
                bdir, max_depth=MAX_BUILD_DIR_DEPTH, skip_nested_linux=True
            ):
                if (
                    path not in seen_explicit
                    and has_license_filename(path)
                    and store.add_file(path)
                ):
                    added_any = True

    if not added_any:
        synthesized = _synthesize_recipe_notice(openwrt_dir, recipe, build_dirs)
        if synthesized is not None:
            store.add_text(synthesized.text, synthesized.source_paths[0])


def load_unique_notices(
    openwrt_dir: Path,
    manifest_pkgs: set[str],
    config_pkgs: set[str],
) -> list[LicenseNotice]:
    """Read and deduplicate license texts for installed packages."""
    store = NoticeStore(openwrt_dir)

    for path in _iter_known_license_files(openwrt_dir):
        store.add_file(path)

    for path in _iter_toolchain_license_files(openwrt_dir):
        if has_license_filename(path):
            store.add_file(path)

    installed_recipes = parse_installed_recipes(
        openwrt_dir, manifest_pkgs, config_pkgs
    )
    if not installed_recipes:
        # Fallback for minimal trees without tmp/.packageinfo.
        for base_dir in sorted(openwrt_dir.glob("build_dir/target-*")):
            for path in _walk_files(
                base_dir, max_depth=MAX_BUILD_DIR_DEPTH, skip_nested_linux=True
            ):
                if has_license_filename(path):
                    store.add_file(path)
    else:
        pkg_to_dirs = index_pkgdirs(openwrt_dir)
        for recipe in installed_recipes:
            _collect_recipe_notices(store, openwrt_dir, recipe, pkg_to_dirs)

    return store.sorted_notices()


def format_source_header(source_paths: list[str]) -> str:
    """Format the source path summary line for a deduplicated notice."""
    shown = ", ".join(source_paths[:MAX_HEADER_PATHS])
    remaining = len(source_paths) - MAX_HEADER_PATHS
    if remaining > 0:
        return f"{shown} (+{remaining} more)"
    return shown


def build_notice_report(openwrt_dir: Path) -> NoticeReport:
    """Collect installed manifests and deduplicated license notices."""
    manifest_pkgs, config_pkgs, manifests = load_installed_packages(openwrt_dir)
    manifest_sections = [
        manifest.read_text(encoding="utf-8", errors="replace").strip()
        for manifest in manifests
    ]
    notices = load_unique_notices(openwrt_dir, manifest_pkgs, config_pkgs)
    if not notices:
        raise RuntimeError(f"No license files found under {openwrt_dir}")
    return NoticeReport(manifest_sections=manifest_sections, notices=notices)


def render_notices(report: NoticeReport) -> str:
    """Render a NoticeReport into the plain-text THIRD_PARTY_NOTICES format."""
    sections = [
        "THIRD-PARTY SOFTWARE NOTICES AND INFORMATION\n\n"
        "Corresponding source archives are distributed in the companion "
        "*-sources.tar.gz bundle.\n\n"
        "INSTALLED PACKAGES\n"
    ]
    for manifest_text in report.manifest_sections:
        sections.append(f"{manifest_text}\n")

    divider = "=" * 72
    for notice in report.notices:
        header = format_source_header(notice.source_paths)
        sections.append(
            f"\n{divider}\nSource files: {header}\n{divider}\n\n"
            f"{notice.text}\n"
        )
    return "".join(sections)


def collect_notices(openwrt_dir: Path) -> str:
    """Build the full THIRD_PARTY_NOTICES document for an OpenWrt build."""
    return render_notices(build_notice_report(openwrt_dir))


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
