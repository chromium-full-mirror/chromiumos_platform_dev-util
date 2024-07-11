#!/bin/bash
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# The template for integrating SoftAP on top of Ubuntu.
# Replace WIFI_PKG_NAME with the package name provided by Intel.
# Generate a new iteration of script:
# $ cp cros-install-wifi-to-host-template.sh cros-install-wifi-to-host-TIMESTAMP.sh
# $ base64 WiFi-pkg.tar >> cros-install-wifi-to-host-TIMESTAMP.sh

# directory where a tarball is to be extracted
WORK_DIR=/tmp

# Directory name
DIR_NAME=WiFi-pkg

# tar file name in host
TAR_NAME=${DIR_NAME}.tar

# The jenkins package name provided by Intel
WIFI_PKG_NAME=jenkins-STUB

if [ $# -eq 0 ]; then
      >&2 echo "No host argument provided"
      exit 1
fi

HOST=$1

install_dependencies() {
  ssh -C "${HOST}" 'echo test0000 | sudo -S apt-get install iw net-tools build-essential checkinstall linux-headers-$(uname -r) libreadline-dev  gcc-multilib pkg-config qtcreator qtbase5-dev qt5-qmake cmake libnl-3-dev libssl-dev libnl-genl-3-dev isc-dhcp-server libpcap-dev iputils-arping gcc-12 iperf -y'
}

# We disable autoupdates on ubuntu because it will overwrite the unsigned kernel modules
# during stealth updates.
disable_autoupdate() {
  ssh -C "${HOST}" 'echo test0000 | sudo -S bash -c "cat > /etc/apt/apt.conf.d/20auto-upgrades << EOF
APT::Periodic::Update-Package-Lists "0";
APT::Periodic::Unattended-Upgrade "0";
EOF"'
}
echo "Installing dependencies on host ${HOST}"
install_dependencies
echo "Completed dependency installation on host ${HOST}"

echo "Disabling auto update on host ${HOST}"
disable_autoupdate
echo "Completed disabling auto update on host ${HOST}"

echo "Uploading ${TAR_NAME} to ${DIR_NAME} on host ${HOST}"
# line number where payload starts
PAYLOAD_LINE=$(awk '/^__INTEL_SOFTAP_TAR_PAYLOAD_BEGINS__/ { print NR + 1; exit 0; }' "$0")

# extract the embedded tar file encoded in base64
tail -n +"${PAYLOAD_LINE}" "$0" | base64 -d | ssh "${HOST}" "cat > ${WORK_DIR}/${TAR_NAME}"

echo "${TAR_NAME} uploaded to ${DIR_NAME} on host ${HOST}"

echo "Start extraction to ${WORK_DIR}/${DIR_NAME} on host ${HOST}"
# process_tar
ssh -C "${HOST}" "tar -pxf ${WORK_DIR}/${TAR_NAME} -C ${WORK_DIR}"
echo "Completed extraction to ${WORK_DIR}/${DIR_NAME} on host ${HOST}"

echo "Start building and installing SoftAP on host ${HOST}"
# build_and_install_wifi
ssh -C "${HOST}" "chmod +x ${WORK_DIR}/${DIR_NAME}/wifi-install.sh"
ssh -C "${HOST}" "chmod +x ${WORK_DIR}/${DIR_NAME}/cros-wifi-install.sh"
ssh -C "${HOST}" "${WORK_DIR}/${DIR_NAME}/wifi-install.sh ${WORK_DIR}/${DIR_NAME}/${WIFI_PKG_NAME} --no-reboot"
ssh -C "${HOST}" "${WORK_DIR}/${DIR_NAME}/cros-wifi-install.sh"

echo "Completed building and installing SoftAP on host ${HOST}"

exit 0
__INTEL_SOFTAP_TAR_PAYLOAD_BEGINS__

