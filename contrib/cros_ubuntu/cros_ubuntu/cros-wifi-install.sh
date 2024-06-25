#!/bin/bash
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

cros_reboot="true"
cros_softap_dir=""

if [[ "$1" =~ (--no-reboot) ]]; then
        echo "setting argument: cros_reboot=false"
        cros_reboot="false"
        shift
fi

script_path=$(readlink -m -n "$0")
cros_softap_dir=$(dirname "${script_path}")

echo -e "Running Google ChromeOS WiFi extra steps..."

FRAMESENDER_DIR=${cros_softap_dir}/frame_sender

echo -e "Installing frame sender"

pushd "${FRAMESENDER_DIR}" ||
        status_eval $? "pushd ${FRAMESENDER_DIR}" || return 1
make
echo test0000 | sudo -S cp send_management_frame /usr/bin/

popd || exit # $FRAMESENDER_DIR

# Disable custom hostapd build to use Intel's softap bundle

# HOSTAP_DIR=${cros_softap_dir}/hostap
# HOSTAP_PACKAGE=${HOSTAP_DIR}.tar.gz

# pushd "${cros_softap_dir}" ||
#         status_eval $? "pushd ${cros_softap_dir}" || return 1
# echo "extracting hostap files..."
# tar -xzf "${HOSTAP_PACKAGE}" > /dev/null 2>&1 ||
#         die 1 "ERROR: extracting from file: ${HOSTAP_PACKAGE}. System error is $?"
# popd || exit # ${cros_softap_dir}

# pushd "${HOSTAP_DIR}/hostapd" ||
#         status_eval $? "pushd ${HOSTAP_DIR}/hostapd" || return 1
# # copy default config
# cp defconfig .config

# make -j \
#     CONFIG_WEP=1 \
#     CONFIG_IEEE80211AC=1 \
#     CONFIG_IEEE80211AX=1 \
#     CONFIG_IEEE80211BE=1 \
#     CONFIG_MBO=1 \
#     all

# sudo make BINDIR=/usr/bin install
# popd || exit # ${HOSTAP_DIR}/hostapd

echo -e "Adding a symlink from /usr/bin/hostapd_cli to /usr/local/bin/hostapd_cli"
sudo ln -s /usr/local/bin/hostapd_cli /usr/bin/hostapd_cli

# rename Ethernet interface name to eth0
echo -e "Renaming Ethernet interface name to eth0"
echo test0000 | sudo -S sed -i 's/GRUB_CMDLINE_LINUX\=.*/GRUB_CMDLINE_LINUX\="net.ifnames=0 biosdevname=0"/g' /etc/default/grub
sudo update-grub
echo -e "Ethernet interface renaming will take effect next boot"

# reboot
if [ "${cros_reboot}" = "true" ]; then
        echo -e "system Will reboot in 10 seconds...\nin case you don't want to restart press CTRL+C now."
        sleep 10
        sudo reboot
else
        echo "system reboot disabled by option."
fi
