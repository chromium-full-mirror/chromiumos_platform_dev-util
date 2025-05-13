#!/bin/bash
#
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
#
# Sets up the amarisoft callbox with the 4G eNode-B configuration.

dir=$(pwd -P)
cp -r enb/* /root/enb/ || exit
cp -r mme/* /root/mme/ || exit

cd /root/enb/config/ || exit
# setup 4G radio
ln -sfn cros.enb.cfg enb.cfg

cd /root/mme/config/ || exit
ln -sfn cros-mme-ims.cfg mme.cfg
ln -sfn cros-ims.cfg ims.cfg

service lte stop
# find out how many SDR cards are there in the callbox, and set the value in the configuration files
cros_sdr_count="$(/root/enb/config/sdr/sdr_util version | grep "/dev/sdr" | uniq | wc -l)"
if [ "${cros_sdr_count}" -eq 0 ]; then
  echo "Cannot find any SDR cards"
  return 1
fi
echo "#define CROS_SDR_COUNT ${cros_sdr_count}" > /root/enb/config/cros_sdr_count.cfg
service lte start

# Configure log rotation for lte services.
printf 'source ots.default.cfg\nLOG_SIZE="5M"\nLOG_COUNT="10K"' > /root/ots/config/cros-ots.cfg
ln -sfn cros-ots.cfg /root/ots/config/ots.cfg

cd "${dir}"
source setupCrosServer.sh
source setupFileServer.sh
source setupDNS.sh
