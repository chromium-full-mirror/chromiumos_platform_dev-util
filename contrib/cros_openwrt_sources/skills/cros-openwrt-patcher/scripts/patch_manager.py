#!/usr/bin/env python3
# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""ChromeOS OpenWrt Patch Manager (quilt automation).

Manages quilt patch creation, editing, compilation testing, and exporting
inside OpenWrt debug Docker containers.
"""

import argparse
import os
import re
import subprocess
import sys
from typing import List, Optional


SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))
REPO_ROOT = os.path.realpath(os.path.join(SCRIPT_DIR, "../../../.."))
IMAGE_GEN_DIR = os.path.join(REPO_ROOT, "contrib/cros_openwrt_sources/image_generator")

PACKAGE_PATH_MAP = {
    "hostapd": "package/network/services/hostapd",
    "mt76": "package/kernel/mt76",
    "mac80211": "package/kernel/mac80211",
    "uboot-mediatek": "package/boot/uboot-mediatek",
}


def get_container_name(device: str) -> str:
    """Returns the Docker container name for a device."""
    return f"openwrt_debug_{device}"


def run_docker_cmd(
    container: str,
    cmd: str,
    cwd: Optional[str] = None,
    capture_output: bool = False,
    check: bool = True,
) -> subprocess.CompletedProcess:
    """Executes a command inside the specified Docker container."""
    docker_cmd = ["docker", "exec", "-u", "1001"]
    if cwd:
        docker_cmd += ["-w", cwd]
    docker_cmd += [container, "bash", "-c", f"export PATH=/build/openwrt/staging_dir/host/bin:$PATH; {cmd}"]

    print(f"[Docker: {container}] $ {cmd}")
    return subprocess.run(
        docker_cmd,
        capture_output=capture_output,
        text=True,
        check=check,
    )


def find_package_build_dir(container: str, package: str) -> str:
    """Finds the build directory for a package in the container."""
    find_cmd = f"find /build/openwrt/build_dir -maxdepth 2 -type d -name '{package}*' | head -n 1"
    res = run_docker_cmd(container, find_cmd, capture_output=True, check=True)
    build_dir = res.stdout.strip()
    if not build_dir:
        raise RuntimeError(
            f"Build directory for package '{package}' not found. Run 'prepare' first."
        )
    return build_dir


def prepare_package(device: str, package: str):
    """Prepares a package for quilt patch management."""
    container = get_container_name(device)
    pkg_rel_path = PACKAGE_PATH_MAP.get(package, f"package/{package}")
    print(f"Preparing package '{package}' with QUILT=1 in {container}...")
    run_docker_cmd(
        container,
        f"make {pkg_rel_path}/clean {pkg_rel_path}/prepare V=s QUILT=1",
        cwd="/build/openwrt",
    )
    build_dir = find_package_build_dir(container, package)
    print(f"Package prepared! Build directory: {build_dir}")


def get_next_patch_prefix(device: str, package: str) -> str:
    """Calculates the next 9000-series patch number."""
    pkg_patches_dir = os.path.join(
        IMAGE_GEN_DIR, "configs", device, "patches-package", package
    )
    if not os.path.isdir(pkg_patches_dir):
        return "9001"

    highest = 9000
    for fname in os.listdir(pkg_patches_dir):
        match = re.match(r"^(\d{4})-.*\.patch$", fname)
        if match:
            num = int(match.group(1))
            if num >= 9000 and num > highest:
                highest = num

    return f"{highest + 1:04d}"


def new_patch(device: str, package: str, patch_name: str):
    """Creates a new quilt patch for a package."""
    container = get_container_name(device)
    build_dir = find_package_build_dir(container, package)

    if not re.match(r"^\d{4}-", patch_name):
        prefix = get_next_patch_prefix(device, package)
        patch_name = f"{prefix}-{patch_name}"
    if not patch_name.endswith(".patch"):
        patch_name = f"{patch_name}.patch"

    print(f"Creating quilt patch '{patch_name}' in {build_dir}...")
    run_docker_cmd(
        container,
        f"quilt new {patch_name}",
        cwd=build_dir,
    )
    print(f"Quilt patch '{patch_name}' created!")


def add_file(device: str, package: str, filepath: str):
    """Adds a file to the active quilt patch."""
    container = get_container_name(device)
    build_dir = find_package_build_dir(container, package)
    print(f"Adding '{filepath}' to active quilt patch...")
    run_docker_cmd(
        container,
        f"quilt add {filepath}",
        cwd=build_dir,
    )
    print(f"File '{filepath}' added to quilt.")


def refresh_patch(device: str, package: str):
    """Refreshes the active quilt patch."""
    container = get_container_name(device)
    build_dir = find_package_build_dir(container, package)
    print(f"Refreshing active quilt patch in {build_dir}...")
    run_docker_cmd(
        container,
        "quilt refresh",
        cwd=build_dir,
    )
    print("Quilt patch refreshed successfully!")


def compile_package(device: str, package: str):
    """Tests compilation of a package inside the container."""
    container = get_container_name(device)
    pkg_rel_path = PACKAGE_PATH_MAP.get(package, f"package/{package}")
    print(f"Testing compilation of '{package}' in {container}...")
    run_docker_cmd(
        container,
        f"make {pkg_rel_path}/compile V=s -j$(nproc)",
        cwd="/build/openwrt",
    )
    print(f"Compilation of '{package}' succeeded!")


def export_patch(device: str, package: str, validate: bool = True):
    """Updates OpenWrt package patches and copies to repo configs."""
    container = get_container_name(device)
    pkg_rel_path = PACKAGE_PATH_MAP.get(package, f"package/{package}")
    print(f"Updating package patches via make {pkg_rel_path}/update...")
    run_docker_cmd(
        container,
        f"make {pkg_rel_path}/update V=s",
        cwd="/build/openwrt",
    )

    dest_dir = os.path.join(IMAGE_GEN_DIR, "configs", device, "patches-package", package)
    os.makedirs(dest_dir, exist_ok=True)

    print(f"Copying package patches from container to {dest_dir}...")
    cp_cmd = [
        "docker",
        "cp",
        f"{container}:/build/openwrt/{pkg_rel_path}/patches/.",
        dest_dir,
    ]
    subprocess.run(cp_cmd, check=True)

    print("Patches exported successfully!")

    if validate:
        val_script = os.path.join(IMAGE_GEN_DIR, "validate_patches.sh")
        print(f"Running pre-flight patch validation: {val_script} {device}...")
        subprocess.run([val_script, device], check=True)


def main():
    parser = argparse.ArgumentParser(
        description="Manage OpenWrt package patches using quilt inside Docker."
    )
    subparsers = parser.add_subparsers(dest="action", required=True)

    # prepare
    p_prep = subparsers.add_parser("prepare", help="Prepare a package with QUILT=1")
    p_prep.add_argument("device", help="Target device (e.g. BananaPi-R4)")
    p_prep.add_argument("package", help="Package name (e.g. hostapd, mt76)")

    # new
    p_new = subparsers.add_parser("new", help="Create a new quilt patch")
    p_new.add_argument("device", help="Target device")
    p_new.add_argument("package", help="Package name")
    p_new.add_argument("patch_name", help="Patch name (e.g. mlo-fix-beacon)")

    # add
    p_add = subparsers.add_parser("add", help="Add file to active quilt patch")
    p_add.add_argument("device", help="Target device")
    p_add.add_argument("package", help="Package name")
    p_add.add_argument("filepath", help="Relative file path within package src")

    # refresh
    p_ref = subparsers.add_parser("refresh", help="Refresh active quilt patch")
    p_ref.add_argument("device", help="Target device")
    p_ref.add_argument("package", help="Package name")

    # compile
    p_cmp = subparsers.add_parser("compile", help="Test compile package")
    p_cmp.add_argument("device", help="Target device")
    p_cmp.add_argument("package", help="Package name")

    # export
    p_exp = subparsers.add_parser("export", help="Export patches to repo and validate")
    p_exp.add_argument("device", help="Target device")
    p_exp.add_argument("package", help="Package name")
    p_exp.add_argument(
        "--no-validate",
        action="store_true",
        help="Skip running validate_patches.sh after export",
    )

    args = parser.parse_args()

    if args.action == "prepare":
        prepare_package(args.device, args.package)
    elif args.action == "new":
        new_patch(args.device, args.package, args.patch_name)
    elif args.action == "add":
        add_file(args.device, args.package, args.filepath)
    elif args.action == "refresh":
        refresh_patch(args.device, args.package)
    elif args.action == "compile":
        compile_package(args.device, args.package)
    elif args.action == "export":
        export_patch(args.device, args.package, validate=not args.no_validate)


if __name__ == "__main__":
    main()
