#!/bin/sh
[ -x /etc/init.d/wpad ] && /etc/init.d/wpad disable
[ -x /etc/init.d/dnsmasq ] && /etc/init.d/dnsmasq disable
[ -x /etc/init.d/odhcpd ] && /etc/init.d/odhcpd disable
[ -x /etc/init.d/sysntpd ] && /etc/init.d/sysntpd disable
