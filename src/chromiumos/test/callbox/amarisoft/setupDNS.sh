#!/bin/bash -e
#
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
#
# Sets up the DNS server on the amarisoft callbox.

# If bind is not installed:
#   sudo yum install bind* -y

systemctl stop named

cp named.conf /etc/named.conf

cat <<EOT > /etc/cros_named_zones.conf
// forward zone
zone "server-callbox.cros" IN {
    type master;
    file "forward.server-callbox.cros";
    allow-update { none; };
    allow-query {any; };
};
// reverse zone
zone "168.192.in-addr.arpa" IN {
    type master;
    file "reverse.server-callbox.cros";
    allow-update { none; };
    allow-query { any; };
};
//    0       0       0       0       1       3000    0468    2001
//zone "0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.1.0.0.0.0.0.0.3.8.6.4.0.1.0.0.2.ip6.arpa" IN {
zone "3.8.6.4.0.1.0.0.2.ip6.arpa" IN {
    type master;
    file "reverse.server-callbox.cros";
    allow-update { none; };
    allow-query { any; };
};
//    0       0       0       0       1       5000    0468    2001
zone "0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.1.0.0.0.0.0.0.5.8.6.4.0.1.0.0.2.ip6.arpa" IN {
    type master;
    file "reverse.server-callbox.cros";
    allow-update { none; };
    allow-query { any; };
};

EOT


cat <<EOT > /var/named/forward.server-callbox.cros
\$TTL 8h
@ IN SOA ns1.server-callbox.cros. admin.server-callbox.cros. (
    2019061800 ;Serial
    1d ;Refresh
    3h ;Retry
    3d ;Expire
    3h ) ;Minimum TTL

;Name Server Information
@ IN NS ns1.server-callbox.cros.
;IP for Name Server
ns1 IN A    192.168.4.1
ns1 IN AAAA 2001:468:3000:1::
ns1 IN AAAA 2001:468:5000:1::
;A Record for IP address to Hostname
@ IN A 192.168.4.1
www IN A 192.168.4.1
ipv4 IN A 192.168.4.1

@ IN AAAA 2001:468:3000:1::0
@ IN AAAA 2001:468:5000:1::0
www IN AAAA 2001:468:3000:1::
ipv6 IN AAAA 2001:468:3000:1::

EOT

cat <<EOT > /var/named/reverse.server-callbox.cros
\$TTL 86400
@ IN SOA ns1.server-callbox.cros. admin.server-callbox.cros. (
    2019061800 ;Serial
    1d ;Refresh
    3h ;Retry
    3d ;Expire
    3h ) ;Minimum TTL

;Name Server Information
@ IN NS ns1.server-callbox.cros.
;Reverse lookup for Name Server
1 IN PTR ns1.server-callbox.cros.
;PTR Record IP address to HostName
1 IN PTR www.server-callbox.cros.
1 IN PTR ipv4.server-callbox.cros.
0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.1.0.0.0.0.0.0.3.8.6.4.0.1.0.0.2        PTR    www.server-callbox.cros.
0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.1.0.0.0.0.0.0.5.8.6.4.0.1.0.0.2        PTR    www.server-callbox.cros.
0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.1.0.0.0.0.0.0.3.8.6.4.0.1.0.0.2        PTR    ipv6.server-callbox.cros.
0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.1.0.0.0.0.0.0.5.8.6.4.0.1.0.0.2        PTR    ipv6.server-callbox.cros.
EOT

chown :named /var/named/forward.server-callbox.cros
chown :named /var/named/reverse.server-callbox.cros
systemctl daemon-reload
systemctl enable named
systemctl start named
