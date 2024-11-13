#!/bin/bash

# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# This is a script to generate capacitor tables from all the firmware
# manifests textprotos.
# The script parses the firmware manifests, combine them into one table and uploads
# it to colossus.
# The script should be run outside the chroot because gqui is not available
# inside it.

src_path="$(pwd | sed  's/chromiumos\/src.*/chromiumos\/src/')"
proto_path="${src_path}"/platform2/modemfwd/proto/firmware_manifest_v2.proto

if [ ! -f "${proto_path}" ]; then
  echo "Cannot locate firmware_manifest_v2.proto path. Try running inside " \
    "the chromium OS src checkout"
  exit 1
fi

tmpfile=$(mktemp /tmp/parsed_fw_manifests.XXXXXX)
for fm_path in "${src_path}"/private-overlays/chromeos-partner-overlay/chromeos-base/modemfwd-helpers-*/files/firmware_manifest.* \
    "${src_path}"/private-overlays/overlay-*-private/chromeos-base/modemfwd-helpers/files/firmware_manifest.* \
    "${src_path}"/overlays/overlay-*/chromeos-base/modemfwd-helpers/files/firmware_manifest.*; do
  board=''
  # Extract the value of the `*` from the globs to set the board name
  #match first glob
  [[ "${fm_path}" =~ modemfwd-helpers-([a-z]*)/files/firmware_manifest ]]
  if [ -n "${BASH_REMATCH[1]}" ]; then
    board="${BASH_REMATCH[1]}"
  else
    #match second glob
    [[ "${fm_path}" =~ private-overlays/overlay-([a-z]*)-private/chromeos-base/modemfwd-helpers ]]
    if [ -n "${BASH_REMATCH[1]}" ]; then
      board="${BASH_REMATCH[1]}"
    else
      #match third glob
      [[ "${fm_path}" =~ overlays/overlay-([a-z]*)/chromeos-base/modemfwd-helpers ]]
      if [ -n "${BASH_REMATCH[1]}" ]; then
        board="${BASH_REMATCH[1]}"
      fi
    fi
  fi
  ( \
  gqui from "flatten(textproto:${fm_path}, device.carrier_firmware.carrier_id)" \
    proto "${proto_path}":FirmwareManifestV2 \
    "select device.device_id, device.variant, " \
      "device.carrier_firmware.carrier_id, " \
      "device.carrier_firmware.main_firmware_version != NULL ? device.carrier_firmware.main_firmware_version : device.default_main_firmware_version " \
      "format '%r,%r,%r,%r\n'" \
    --quiet &&
  gqui from "flatten(textproto:${fm_path}, device.oem_firmware.main_firmware_version)" \
    proto "${proto_path}":FirmwareManifestV2 \
    "select device.device_id, device.variant, 'generic' as carrier_id, " \
      "device.oem_firmware.main_firmware_version != NULL ? device.oem_firmware.main_firmware_version : device.default_main_firmware_version " \
      "where all(device.carrier_firmware[*].filename == NULL) " \
      "format '%r,%r,%r,%r\n'" \
    --quiet \
  ) \
    | sed -e 's/usb:2cb7:0007/L850GL/g' \
      -e 's/pci:14c3:4d75 (External)/FM350/g' -e 's/usb:2cb7:01a0/NL668AM/g' \
      -e 's/usb:2cb7:01a2/FM101/g' -e 's/usb:2c7c:0128/EM060/g' \
      -e 's/usb:33f8:0115/RW135/g' -e 's/usb:33f8:01a2/RW101/g' \
      -e 's/usb:3731:0100/LCUK54/g' \
    | while read -r line; do echo "${line},${board}"; done >> "${tmpfile}";
done

fileutil cp -f "${tmpfile}" \
  /cns/ok-d/home/croscellular/modemfw/firmware_manifests.csv

while test $# -gt 0
do
    case "$1" in
        --v|-v) cat "${tmpfile}"
            ;;
    esac
    shift
done

rm "${tmpfile}"
