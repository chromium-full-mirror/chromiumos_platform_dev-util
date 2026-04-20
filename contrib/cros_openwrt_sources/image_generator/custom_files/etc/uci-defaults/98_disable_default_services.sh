#!/bin/sh
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
. /lib/functions/system.sh

if [ "$(board_name)" != 'bananapi,bpi-r4' ]; then
    [ -x /etc/init.d/wpad ] && /etc/init.d/wpad disable
    [ -x /etc/init.d/dnsmasq ] && /etc/init.d/dnsmasq disable
    [ -x /etc/init.d/odhcpd ] && /etc/init.d/odhcpd disable
    [ -x /etc/init.d/sysntpd ] && /etc/init.d/sysntpd disable
else
    [ -x /etc/init.d/uhttpd ] && /etc/init.d/uhttpd disable
fi
