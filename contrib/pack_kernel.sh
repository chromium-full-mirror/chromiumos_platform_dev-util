#!/bin/bash

# Copyright 2025 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# pack_kernel.sh
# Build a bundle to update ChromeOS kernel similar to update_kernel.sh, but with strict gating:
#   - NO changes are made unless rootfs verification has been removed.
#   - --ab_update also requires the TARGET rootfs to be writable (verification removed for that slot).
#
# Bundle contents:
#   - vmlinuz
#   - boot.tar.gz (root-relative /boot from /build/<BOARD>/boot)
#   - modules.tar.gz (root-relative /lib/modules from /build/<BOARD>/lib/modules)
#   - optional devkeys (kernel.keyblock, kernel_data_key.vbprivk)
#   - apply_bundle.sh (POSIX /bin/sh) that:
#       1) Determines TARGET kernel slot (in-place by default; --ab_update writes the other slot).
#       2) Reads TARGET bootargs via dump_kernel_config (no modifications).
#       3) EARLY GATE (no changes before this):
#            - Non-AB: require current root (/) mounted rw.
#            - AB: require TARGET rootfs (PARTNROFF=1 from TARGET kernel partition) mountable as rw.
#            - If gate fails, print make_dev_ssd.sh instructions and exit without changes.
#       4) vbutil_kernel pack using RAW bootargs (no macro expansion, no edits).
#       5) dd kernel.bin to TARGET kernel partition.
#       6) Sync files:
#            - Non-AB: extract boot/modules tarballs to current root (/).
#            - AB: mount TARGET rootfs rw and extract tarballs into that mount (so the other slot's rootfs is updated).
#       7) Best-effort: update EFI syslinux vmlinuz.{A,B}.
#       8) For --ab_update, set boot-once on TARGET slot.
#
# On host:
#   ./pack_kernel.sh --board <BOARD> [--out <bundle.tar.gz>] [--no-devkeys]
#   or:
#   ./pack_kernel.sh --kernel_image /build/<BOARD>/boot/vmlinuz-... --board <BOARD>
#
# On device (noexec safe):
#   tar -xzf <bundle>.tar.gz -C /tmp
#   sh /tmp/<bundle-name>/apply_bundle.sh [--reboot] [--ab_update]

set -euo pipefail

BOARD=""
KERNEL_IMAGE=""
OUT_PATH=""
BUNDLE_NAME="kernel_bundle"

INCLUDE_DEVKEYS=1
KEYBLOCK_DEFAULT="/usr/share/vboot/devkeys/kernel.keyblock"
SIGNPRIV_DEFAULT="/usr/share/vboot/devkeys/kernel_data_key.vbprivk"
KEYBLOCK="$KEYBLOCK_DEFAULT"
SIGNPRIV="$SIGNPRIV_DEFAULT"

VBVER="1"
VBFLAGS="0"

print_help() {
  cat <<'HLP'
Usage:
  ./pack_kernel.sh [options]

Options:
  --board <BOARD> | --board=<BOARD>
  --kernel_image <path> | --kernel_image=<path>
  --out <path.tar.gz> | --out=<path.tar.gz>
  --bundle-name <name> | --bundle-name=<name>
  --version <N> | --version=<N>           (vbutil_kernel --version, default 1)
  --flags <N|0xHEX> | --flags=<N|0xHEX>   (vbutil_kernel --flags, default 0)
  --no-devkeys
  --keyblock <path> | --keyblock=<path>
  --signprivate <path> | --signprivate=<path>
  -h | --help
HLP
}

need_tool() { command -v "$1" >/dev/null 2>&1 || { echo "Error: '$1' not found"; exit 1; }; }
timestamp() { date +"%Y%m%d%H%M%S"; }

detect_latest_vmlinuz_for_board() {
  local b="${1:-}"
  [[ -n "$b" ]] || { echo "Error: board not specified" >&2; return 1; }
  local d="/build/${b}/boot"
  [[ -d "$d" ]] || { echo "Error: not found: $d" >&2; return 1; }
  local latest
  latest=$(find -L "$d" -maxdepth 1 -type f -name 'vmlinuz*' -printf '%T@ %p\n' 2>/dev/null | sort -n | awk '{print $2}' | tail -n1 || true)
  [[ -n "${latest:-}" ]] || { echo "Error: no vmlinuz* found under $d" >&2; return 1; }
  echo "$latest"
}

detect_kernel_version_from_build_boot() {
  local bootdir="$1"
  if [[ -L "${bootdir}/vmlinuz" ]]; then
    readlink "${bootdir}/vmlinuz" | cut -d- -f2- || true
  else
    local vf
    vf=$(find "${bootdir}" -maxdepth 1 -type f -name 'vmlinuz-*' 2>/dev/null | head -n1 || true)
    [[ -n "$vf" ]] && basename "$vf" | cut -d- -f2- || true
  fi
}

embed_apply_script() {
  local outdir="$1"
  cat > "${outdir}/apply_bundle.sh" <<"EOF_APPLY_BUNDLE"
#!/bin/sh
set -eu

REBOOT=0
VBFLAGS="${VBFLAGS:-0}"
VBVER="${VBVER:-1}"
DO_AB=0   # default: in-place overwrite current kernel slot

while [ $# -gt 0 ]; do
  case "$1" in
    --reboot) REBOOT=1; shift ;;
    --flags) VBFLAGS="${2:-0}"; shift 2 ;;
    --version) VBVER="${2:-1}"; shift 2 ;;
    --ab_update) DO_AB=1; shift ;;
    *) echo "Unknown option: $1" >&2; exit 2 ;;
  esac
done

need_tool() { command -v "$1" >/dev/null 2>&1 || { echo "Error: '$1' not found"; exit 1; }; }
require_root() { [ "$(id -u)" -eq 0 ] || { echo "Run as root"; exit 1; }; }

find_base_disk() {
  # Find base disk device (e.g. /dev/sda)
  if command -v rootdev >/dev/null 2>&1; then rootdev -d -s; else
    dev="$(findmnt -n -o SOURCE / 2>/dev/null || true)"; echo "${dev%%[0-9]*}"
  fi
}

partition_path() {
  # Given disk and partition number, return device path (e.g. /dev/sda3)
  disk="$1"; pn="$2"; case "$disk" in *[0-9]) sep="p";; *) sep="";; esac; printf '%s%s%s\n' "$disk" "$sep" "$pn"
}

map_arch() {
  # Map uname -m to vbutil_kernel arch
  case "$(uname -m)" in
    x86_64|i*86) echo x86 ;;
    aarch64|arm64) echo arm64 ;;
    arm* ) echo arm ;;
    mips* ) echo mips ;;
    * ) echo arm64 ;;
  esac
}

is_root_mount_rw() {
  # Return 0 if / is mounted rw, else 1
  awk '$2=="/"{print $4}' /proc/mounts | grep -q '\<rw\>'
}

mount_target_rootfs_rw_and_echo_mountpoint() {
  # Mount target rootfs partition as rw, echo mountpoint if success
  disk="$1"; target_part="$2"
  # ChromeOS layout uses PARTNROFF=1 from kernel -> rootfs
  rootfs_part=$((target_part + 1))
  rootfs_dev="$(partition_path "$disk" "$rootfs_part")"
  [ -b "$rootfs_dev" ] || { echo "Invalid target rootfs device: $rootfs_dev" >&2; return 2; }
  mnt="/tmp/ukp-target-root.$$"
  mkdir -p "$mnt"
  # Try mount rw; if verification not removed, kernel will mount ro or fail
  if mount -o rw "$rootfs_dev" "$mnt" 2>/dev/null; then
    # Check it's actually rw
    if awk -v m="$mnt" '$2==m{print $4}' /proc/mounts | grep -q '\<rw\>'; then
      echo "$mnt"
      return 0
    else
      umount "$mnt" >/dev/null 2>&1 || true
      rmdir "$mnt" >/dev/null 2>&1 || true
      echo "Target rootfs mounted read-only (verification likely still enabled) on $rootfs_dev" >&2
      return 3
    fi
  else
    rmdir "$mnt" >/dev/null 2>&1 || true
    echo "Failed to mount target rootfs $rootfs_dev (verification likely still enabled)" >&2
    return 4
  fi
}

require_root
need_tool vbutil_kernel
need_tool cgpt
need_tool dd
need_tool dump_kernel_config
need_tool tar
need_tool mount
need_tool awk
need_tool grep
need_tool rm

SCRIPT_DIR="$(CDPATH= ; cd -- "$(dirname -- "$0")" && pwd)"
cd "$SCRIPT_DIR" || exit 1

[ -f vmlinuz ] || { echo "vmlinuz not found"; exit 1; }
[ -f boot.tar.gz ] || { echo "boot.tar.gz not found"; exit 1; }
[ -f modules.tar.gz ] || { echo "modules.tar.gz not found"; exit 1; }

# devkeys (read-only)
KEYBLOCK="devkeys/kernel.keyblock"
SIGNPRIV="devkeys/kernel_data_key.vbprivk"
[ -f "$KEYBLOCK" ] || KEYBLOCK="/usr/share/vboot/devkeys/kernel.keyblock"
[ -f "$SIGNPRIV" ] || SIGNPRIV="/usr/share/vboot/devkeys/kernel_data_key.vbprivk"
[ -f "$KEYBLOCK" ] || { echo "keyblock not found: $KEYBLOCK"; exit 1; }
[ -f "$SIGNPRIV" ] || { echo "signprivate not found: $SIGNPRIV"; exit 1; }

# Identify TARGET slot (read-only)
DISK="$(find_base_disk)"
[ -b "$DISK" ] || { echo "Invalid base disk: $DISK"; exit 1; }

KPARTS="$(cgpt find -t kernel -n "$DISK" 2>/dev/null || true)"
[ -n "$KPARTS" ] || { echo "No kernel partitions on $DISK"; exit 1; }

CUR_GUID="$(grep -o 'kern_guid=[^ ]*' /proc/cmdline 2>/dev/null | head -n1 | cut -d= -f2 || true)"
CUR_PART=""
if [ -n "$CUR_GUID" ]; then
  CUR_GUID_LC="$(echo "$CUR_GUID" | tr 'A-Z' 'a-z')"
  for pn in $KPARTS; do
    GUID="$(cgpt show -i "$pn" -u "$DISK" 2>/dev/null | tr 'A-Z' 'a-z' || true)"
    [ -n "$GUID" ] && [ "$GUID" = "$CUR_GUID_LC" ] && CUR_PART="$pn"
  done
fi
[ -n "$CUR_PART" ] || CUR_PART="$(printf '%s\n' "$KPARTS" | awk 'NR==1{print; exit}')"

TARGET_PART="$CUR_PART"
if [ "$DO_AB" -eq 1 ]; then
  for pn in $KPARTS; do [ "$pn" != "$CUR_PART" ] && { TARGET_PART="$pn"; break; }; done
fi
TARGET_DEV="$(partition_path "$DISK" "$TARGET_PART")"
[ -b "$TARGET_DEV" ] || { echo "Target is not a block device: $TARGET_DEV"; exit 1; }

# Read TARGET bootargs (no changes, no side effects)
RAW_BOOTARGS="$(dump_kernel_config "$TARGET_DEV")"
[ -n "$RAW_BOOTARGS" ] || { echo "dump_kernel_config failed on $TARGET_DEV"; exit 1; }
echo "Bootargs from $TARGET_DEV:"
echo "  $RAW_BOOTARGS"

# EARLY GATE: no changes until verification is confirmed removed.
if [ "$DO_AB" -eq 0 ]; then
  # In-place update: require current root to be rw
  if ! is_root_mount_rw; then
    echo "Root filesystem (/) is not mounted rw. Remove rootfs verification and reboot, then retry."
    echo "  /usr/share/vboot/bin/make_dev_ssd.sh --remove_rootfs_verification -f"
    exit 20
  fi
else
  # AB update: require TARGET rootfs to be mountable as rw (verification removed for that slot)
  if ! mnt="$(mount_target_rootfs_rw_and_echo_mountpoint "$DISK" "$TARGET_PART")"; then
    echo "Cannot mount TARGET rootfs as rw. Remove rootfs verification for that slot and reboot."
    echo "  /usr/share/vboot/bin/make_dev_ssd.sh --remove_rootfs_verification -f"
    exit 21
  fi
  # Clean up the test mount; will remount later for file sync.
  umount "$mnt" >/dev/null 2>&1 || true
  rmdir "$mnt" >/dev/null 2>&1 || true
fi

# From here on, we can modify the system.

ARCH="$(map_arch)"
echo "vbutil_kernel arch=$ARCH flags=$VBFLAGS version=$VBVER"

CMDLINE_FILE="$(mktemp)"; trap 'rm -f "$CMDLINE_FILE"' EXIT
# Use RAW_BOOTARGS as-is (no macro expansion, no edits) — matches update_kernel.sh
printf '%s\n' "$RAW_BOOTARGS" > "$CMDLINE_FILE"

OUT_BIN="kernel.bin"
vbutil_kernel --pack "$OUT_BIN" \
  --keyblock "$KEYBLOCK" \
  --signprivate "$SIGNPRIV" \
  --version "$VBVER" \
  --config "$CMDLINE_FILE" \
  --vmlinuz "vmlinuz" \
  --arch "$ARCH" \
  --flags "$VBFLAGS"

echo "Writing kernel.bin to $TARGET_DEV (partition index $TARGET_PART) ..."
dd if="$OUT_BIN" of="$TARGET_DEV" bs=4K conv=fsync

# Helper: get new kernel version string from modules.tar.gz
get_new_kver() {
  # Extract the first-level directory name from modules.tar.gz (should be the kernel version)
  tar -tzf modules.tar.gz | sed -n '2p' | cut -d/ -f3
}

# Sync files:
NEW_KVER="$(get_new_kver)"
if [ -z "$NEW_KVER" ]; then
  echo "Failed to detect new kernel version for modules cleanup."
  exit 30
fi
OLD_KVER=$(uname -r)
DO_RENAME=0
if [ "${OLD_KVER}" != "${NEW_KVER}" ]; then
  DO_RENAME=1
fi

if [ "$DO_AB" -eq 0 ]; then
  # In-place: update current rootfs
  echo "Updating /boot from boot.tar.gz ..."
  if [ "$DO_RENAME" -eq 1 ]; then
    rename "${OLD_KVER}" "${NEW_KVER}" /boot/* 2>/dev/null || true
  fi
  tar -xzf boot.tar.gz -C /
  echo "Updating /lib/modules from modules.tar.gz ..."
  # Remove old /lib/modules directories except the new one
  if [ "$DO_RENAME" -eq 1 ] && [ -d "/lib/modules/${OLD_KVER}" ]; then
    mv "/lib/modules/${OLD_KVER}" "/lib/modules/${NEW_KVER}"
  fi
  tar -xzf modules.tar.gz -C /
else
  # AB: update TARGET rootfs by mounting it rw and extracting there
  root_mnt="/tmp/ukp-target-root.$$"
  mkdir -p "$root_mnt"
  rootfs_part=$((TARGET_PART + 1))
  rootfs_dev="$(partition_path "$DISK" "$rootfs_part")"
  echo "Mounting TARGET rootfs $rootfs_dev to $root_mnt (rw) ..."
  mount -o rw "$rootfs_dev" "$root_mnt"
  if ! awk -v m="$root_mnt" '$2==m{print $4}' /proc/mounts | grep -q '\<rw\>'; then
    umount "$root_mnt" || true
    rmdir "$root_mnt" || true
    echo "TARGET rootfs unexpectedly not rw after mount. Aborting to avoid partial update."
    exit 22
  fi
  echo "Updating $root_mnt/boot from boot.tar.gz ..."
  if [ "$DO_RENAME" -eq 1 ]; then
    rename "${OLD_KVER}" "${NEW_KVER}" "$root_mnt"/boot/* 2>/dev/null || true
  fi
  tar -xzf boot.tar.gz -C "$root_mnt"
  echo "Updating $root_mnt/lib/modules from modules.tar.gz ..."
  # Remove old $root_mnt/lib/modules directories except the new one
  if [ "$DO_RENAME" -eq 1 ] && [ -d "${root_mnt}/lib/modules/${OLD_KVER}" ]; then
    mv "${root_mnt}/lib/modules/${OLD_KVER}" "${root_mnt}/lib/modules/${NEW_KVER}"
  fi
  tar -xzf modules.tar.gz -C "$root_mnt"
  sync
  umount "$root_mnt" || true
  rmdir "$root_mnt" || true
fi

# Best-effort: update syslinux vmlinuz.{A,B} if EFI present
EFI_PN=""
if cgpt show "$DISK" 2>/dev/null | grep -q ' EFI system'; then
  EFI_PN="$(cgpt find -t efi -n "$DISK" 2>/dev/null | head -n1 || true)"
fi
if [ -n "$EFI_PN" ]; then
  EFI_DEV="$(partition_path "$DISK" "$EFI_PN")"
  if [ -b "$EFI_DEV" ]; then
    mkdir -p /tmp/efi-mnt
    if mount "$EFI_DEV" /tmp/efi-mnt 2>/dev/null; then
      case "$TARGET_PART" in
        2) TARGET_SYS_VML=/tmp/efi-mnt/syslinux/vmlinuz.A ;;
        4) TARGET_SYS_VML=/tmp/efi-mnt/syslinux/vmlinuz.B ;;
        *) TARGET_SYS_VML="" ;;
      esac
      if [ -n "${TARGET_SYS_VML:-}" ] && [ -f "$TARGET_SYS_VML" ]; then
        # Source vmlinuz is inside the rootfs; for AB we just laid one into target rootfs/boot/vmlinuz
        SRC_VML="/boot/vmlinuz"
        if [ "$DO_AB" -eq 1 ]; then
          # Prefer the vmlinuz we just installed into target rootfs if accessible (requires mounting again temporarily ro)
          root_mnt="/tmp/ukp-target-root.$$"
          mkdir -p "$root_mnt"
          rootfs_part=$((TARGET_PART + 1))
          rootfs_dev="$(partition_path "$DISK" "$rootfs_part")"
          if mount -o ro "$rootfs_dev" "$root_mnt" 2>/dev/null && [ -f "$root_mnt/boot/vmlinuz" ]; then
            SRC_VML="$root_mnt/boot/vmlinuz"
            cp "$SRC_VML" "$TARGET_SYS_VML" || true
            umount "$root_mnt" || true
            rmdir "$root_mnt" || true
            SRC_VML=""
          else
            umount "$root_mnt" >/dev/null 2>&1 || true
            rmdir "$root_mnt" >/dev/null 2>&1 || true
          fi
        fi
        if [ -n "${SRC_VML:-}" ] && [ -f "$SRC_VML" ]; then
          cp "$SRC_VML" "$TARGET_SYS_VML" || true
        fi
      fi
      umount /tmp/efi-mnt || true
      rmdir /tmp/efi-mnt || true
    fi
  fi
fi

# For AB, set boot-once on the TARGET slot
if [ "$DO_AB" -eq 1 ]; then
  echo "Setting boot-once for partition $TARGET_PART ..."
  cgpt add -i "$TARGET_PART" -P 15 -S 1 -T 0 "$DISK" || true
else
  echo "In-place kernel update completed (no GPT changes)."
fi

sync
[ "$REBOOT" -eq 1 ] && { echo "Rebooting..."; reboot; }
EOF_APPLY_BUNDLE
  chmod +x "${outdir}/apply_bundle.sh"
}

write_metadata() {
  local outdir="$1" board="$2" kernel_image="$3" flags="$4" version="$5" kver="$6"
  {
    echo "bundle_created_at=$(date -Is)"
    echo "board=${board}"
    echo "kernel_image_source=${kernel_image}"
    echo "flags=${flags}"
    echo "version=${version}"
    echo "kernel_version=${kver}"
    echo "host=$(hostname)"
    echo "user=${USER:-}"
  } > "${outdir}/metadata.txt"
}

main () {
  local BUILD_BOOT_DIR=""
  local BUILD_MODULES_DIR=""
  local KVER=""

  # Parse args
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --board=*) BOARD="${1#*=}"; shift ;;
      --board) BOARD="${2:-}"; shift 2 ;;
      --kernel_image=*) KERNEL_IMAGE="${1#*=}"; shift ;;
      --kernel_image) KERNEL_IMAGE="${2:-}"; shift 2 ;;
      --out=*) OUT_PATH="${1#*=}"; shift ;;
      --out) OUT_PATH="${2:-}"; shift 2 ;;
      --bundle-name=*) BUNDLE_NAME="${1#*=}"; shift ;;
      --bundle-name) BUNDLE_NAME="${2:-}"; shift 2 ;;
      --version=*) VBVER="${1#*=}"; shift ;;
      --version) VBVER="${2:-}"; shift 2 ;;
      --flags=*) VBFLAGS="${1#*=}"; shift ;;
      --flags) VBFLAGS="${2:-}"; shift 2 ;;
      --no-devkeys) INCLUDE_DEVKEYS=0; shift ;;
      --keyblock=*) KEYBLOCK="${1#*=}"; shift ;;
      --keyblock) KEYBLOCK="${2:-}"; shift 2 ;;
      --signprivate=*) SIGNPRIV="${1#*=}"; shift ;;
      --signprivate) SIGNPRIV="${2:-}"; shift 2 ;;
      -h|--help) print_help; exit 0 ;;
      --) shift; break ;;
      *) echo "Unknown option: $1" >&2; exit 2 ;;
    esac
  done

  need_tool tar
  need_tool cp
  need_tool mkdir
  need_tool date

  if [[ -z "$KERNEL_IMAGE" ]]; then
    if [[ -z "$BOARD" ]]; then
      echo "Error: provide --board or --kernel_image" >&2
      exit 1
    fi
    KERNEL_IMAGE=$(detect_latest_vmlinuz_for_board "$BOARD")
  fi
  [[ -f "$KERNEL_IMAGE" ]] || { echo "Error: kernel image not found: $KERNEL_IMAGE" >&2; exit 1; }

  if [[ -n "$BOARD" ]]; then
    BUILD_BOOT_DIR="/build/${BOARD}/boot"
    BUILD_MODULES_DIR="/build/${BOARD}/lib/modules"
    [[ -d "$BUILD_BOOT_DIR" ]] || { echo "Error: not found: $BUILD_BOOT_DIR" >&2; exit 1; }
    [[ -d "$BUILD_MODULES_DIR" ]] || { echo "Error: not found: $BUILD_MODULES_DIR" >&2; exit 1; }
    KVER=$(detect_kernel_version_from_build_boot "$BUILD_BOOT_DIR")
  else
    BOOT_DIR_CANDIDATE="$(dirname "$KERNEL_IMAGE")"
    [[ -d "$BOOT_DIR_CANDIDATE" ]] && BUILD_BOOT_DIR="$BOOT_DIR_CANDIDATE"
    echo "Warning: --board not provided; tar contents rely on non-standard paths." >&2
  fi

  if [[ "$INCLUDE_DEVKEYS" -eq 1 ]]; then
    [[ -f "$KEYBLOCK" ]] || { echo "Error: keyblock not found: $KEYBLOCK" >&2; exit 1; }
    [[ -f "$SIGNPRIV" ]] || { echo "Error: signprivate key not found: $SIGNPRIV" >&2; exit 1; }
  fi

  if [[ -n "$OUT_PATH" ]]; then
    OUT_PATH="${OUT_PATH}/$(timestamp)_kernel.tar.gz"
  else
    OUT_PATH="${HOME}/$(timestamp)_kernel.tar.gz"
  fi

  WORKDIR="$(mktemp -d -t kernel_bundle.XXXXXXXX)"
  trap 'rm -rf "$WORKDIR"' EXIT
  BUNDLE_DIR="${WORKDIR}/${BUNDLE_NAME}"
  mkdir -p "$BUNDLE_DIR"

  # vmlinuz
  cp -f "$KERNEL_IMAGE" "${BUNDLE_DIR}/vmlinuz"

  # boot.tar.gz (root-relative /boot)
  if [[ -n "$BUILD_BOOT_DIR" && -d "$BUILD_BOOT_DIR" ]]; then
    tar -C / -czf "${BUNDLE_DIR}/boot.tar.gz" --transform='s#^build/'"${BOARD}"'/##' "build/${BOARD}/boot"
  else
    tar -czf "${BUNDLE_DIR}/boot.tar.gz" --files-from /dev/null
  fi

  # modules.tar.gz (root-relative /lib/modules)
  if [[ -n "$BUILD_MODULES_DIR" && -d "$BUILD_MODULES_DIR" ]]; then
    tar -C / -czf "${BUNDLE_DIR}/modules.tar.gz" --transform='s#^build/'"${BOARD}"'/##' "build/${BOARD}/lib/modules"
  else
    tar -czf "${BUNDLE_DIR}/modules.tar.gz" --files-from /dev/null
  fi

  # devkeys (optional)
  if [[ "$INCLUDE_DEVKEYS" -eq 1 ]]; then
    mkdir -p "${BUNDLE_DIR}/devkeys"
    cp -f "$KEYBLOCK" "${BUNDLE_DIR}/devkeys/kernel.keyblock"
    cp -f "$SIGNPRIV" "${BUNDLE_DIR}/devkeys/kernel_data_key.vbprivk"
  fi

  # apply script
  embed_apply_script "$BUNDLE_DIR"

  # metadata
  write_metadata "$BUNDLE_DIR" "$BOARD" "$KERNEL_IMAGE" "$VBFLAGS" "$VBVER" "$KVER"

  # pack
  ( cd "$WORKDIR" && tar -czf "$OUT_PATH" "$BUNDLE_NAME" )
  echo "Bundle created: $OUT_PATH"
  echo "Done."
}

main "$@"
