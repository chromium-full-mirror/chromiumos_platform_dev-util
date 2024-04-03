#!/bin/bash
# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Abort on error.
set -eu -o pipefail

# Extract program name for usage instructions.
PROG="$(basename "$0")"

usage() {
  cat <<EOF
Usage: ${PROG} <board_name> <milestone> <fw_version> <signature_path> <output_path>
       Ex. ${PROG} zork R125 15836.0.0 ${HOME}/Downloads/psp_verstage_signature/
           \${CROS_ROOT}/src/third_party/coreboot/3rdparty/blobs/mainboard/google/zork/

For detail, reference the AMD documentation titled "OEM PSP VERSTAGE BL FW Signing
Key Pair Generation and Certificate Request Process" -
http://dr/corp/drive/folders/1ySJyDgbH73W1lqrhxMvM9UYl5TtJt_mw. This document
is Google internal only and is under NDA. This document is loosely based on the
"AMD BIOS Signing Key Pair Generation and Certificate Request Process Document
(id: 56535)" from AMD devhub.

EOF

  if [[ $# -ne 0 ]]; then
    echo "$*" >&2
    exit 1
  else
    exit 0
  fi
}

die() {
  echo "$*" >&2
  exit 1
}

extract_signature() {
  local input_file="$1"
  local output_file="$2"

  if [[ ! -r "${input_file}" ]]; then
    die "Signature file ${input_file} is not readable."
  fi

  if ! grep -q "END AUTH SIGNATURE" "${input_file}"; then
    die "Signature file ${input_file} does not contain signature pattern."
  fi
  # Signature file looks like as follows:
  # -----BEGIN AUTH SIGNATURE-----
  # Algorithm line1
  # <snip>
  # Algorithm lineM
  #
  # Signature line1
  # <snip>
  # Signature lineN
  # -----END AUTH SIGNATURE-----

  # Extracting the signature between the empty line and END AUTH SIGNATURE patterns
  awk '/^$/,/^-----END AUTH SIGNATURE-----$/ \
        { if ($0 != "" && $0 != "-----END AUTH SIGNATURE-----") print }' "${input_file}" > \
                "${output_file}" || \
                die "Unable to parse the signature file"
  if [[ ! -s "${output_file}" ]]; then
    die "${output_file} size is 0. Probably ${input_file} does not contain signature."
  fi
}

# Check the arguments to make sure we have the correct number.
if [[ $# -ne 5 ]]; then
  usage "Error: Incorrect number of arguments"
fi

TMPDIR="$(mktemp -d)"

main() {
  local board_name="$1"
  local milestone="$2"
  local fw_ver="$3"
  local sig_path="$4"
  local output_path="$5"
  local fw_archive="ChromeOS-firmware-${milestone}-${fw_ver}-${board_name}.tar.bz2"

  if [[ ! -d "${sig_path}" ]]; then
    die "Path ${sig_path} containing the signature files does not exist."
  fi

  if [[ ! -d "${output_path}" ]]; then
    die "Output path ${output_path} to copy the signed PSP verstage binaries does not exist."
  fi

  pushd "${TMPDIR}" > /dev/null

  echo "Downloading firmware archive ${fw_archive}..."
  gsutil cp "gs://chromeos-releases/canary-channel/${board_name}/${fw_ver}/${fw_archive}" \
          . 1> /dev/null || die "Unable to download firmware archive"

  echo "Extracting unsigned_psp_verstage and psp_verstage_to_sign from firmware archive..."
  tar -xf "${fw_archive}" -C ./ unsigned_psp_verstage psp_verstage_to_sign || \
          die "Unable to extract unsigned_psp_verstage or psp_verstage_to_sign from firmware archive"

  mkdir -p signed_psp_verstage || die "Unable to create signed_psp_verstage directory"

  while read -r f; do
    fw_target="$(basename "${f}" | cut -d'_' -f1)"
    if [[ -z "${fw_target}" ]]; then
      die "Unexpected empty firmware target. File name incompatible with the script."
    fi

    echo "Appending the signature for ${fw_target}..."
    extract_signature "${f}" "signed_psp_verstage/${fw_target}_sig_base64"

    unsigned_fw="unsigned_psp_verstage/${fw_target}_psp_verstage.bin"
    if [[ ! -e "${unsigned_fw}" ]]; then
      die "${unsigned_fw} does not exist in the firmware archive."
    fi
    fw_size="$(stat -c %s -- "${unsigned_fw}")"
    # Parse PSP_FOOTER_DATA i.e. 64 bytes of 0x99 that marks the end of signable PSP Verstage
    # and hence its size.
    signed_fw_size="$(od -v --address-radix=d -t x4 --width=64 "${unsigned_fw}" | \
                           awk '/[[:digit:]]+( 9{8}){16}$/ {print ($1 + 64)}')"
    if (( signed_fw_size <= 0 || signed_fw_size > fw_size )); then
      die "Unexpected signed_fw_size ${signed_fw_size}"
    fi
    unsigned_fw_size="$(( fw_size - signed_fw_size ))"

    # To support CBFS RO verification, hash of the metadata of all CBFS files are kept in
    # PSP verstage - at the end of the unsigned PSP verstage binary. But that part is excluded
    # from the signing because that can change anytime. Hence the binary in psp_verstage_to_sign
    # excludes that part. So we copy the binary in psp_verstage_to_sign first, followed by the
    # unsigned part. In summary the file format of the signed PSP verstage is:
    #  * 256 bytes of AMD FW Header
    #  * Signable part of PSP Verstage binary ending with 64 bytes of 0x99
    #  * Optional Metadata hash anchor to support CBFS RO verification
    #  * 256 bytes of signature
    cp "psp_verstage_to_sign/${fw_target}_psp_verstage.bin" \
            "signed_psp_verstage/${fw_target}_psp_verstage.signed.bin"
    dd if="unsigned_psp_verstage/${fw_target}_psp_verstage.bin" \
       of="signed_psp_verstage/${fw_target}_psp_verstage.signed.bin" \
       seek="${signed_fw_size}" skip="${signed_fw_size}" bs=1 count="${unsigned_fw_size}" \
       conv=notrunc status=none || die "Unable to copy unsigned part of the binary"
    base64 -d "signed_psp_verstage/${fw_target}_sig_base64" >> \
            "signed_psp_verstage/${fw_target}_psp_verstage.signed.bin"

  done < <(find "${sig_path}" -iname "*psp_verstage.bin.sig")

  cp signed_psp_verstage/*.signed.bin "${output_path}"/.
  echo "Signed PSP verstage binaries are in ${output_path}"
  popd > /dev/null
}

trap '{ rm -rf "${TMPDIR}" ; }' EXIT

main "$@"
