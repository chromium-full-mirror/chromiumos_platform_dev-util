#
# Copyright (C) 2007-2010 OpenWrt.org
#
# This is free software, licensed under the GNU General Public License v2.
# See /LICENSE for more information.
#

# Modified version of:
# https://git.openwrt.org/?p=openwrt/openwrt.git;a=blob_plain;f=package/network/utils/iperf/Makefile;hb=720b99215da295a2e7b4acdef30b58bec6acedc2

include $(TOPDIR)/rules.mk

PKG_NAME:=cros-iperf
PKG_VERSION:=2.0.9
PKG_RELEASE:=1

PKG_SOURCE:=iperf-$(PKG_VERSION).tar.gz
PKG_HASH:=db02911f35686e808ed247160dfa766e08ae3f59d1e7dcedef0ffb2a6643f0bf
PKG_SOURCE_URL:=@SF/iperf2
PKG_BUILD_DIR:=$(BUILD_DIR)/iperf-$(PKG_VERSION)

PKG_MAINTAINER:=Felix Fietkau <nbd@nbd.name>
PKG_LICENSE:=BSD-3-Clause
PKG_LICENSE_FILES:=COPYING


PKG_BUILD_PARALLEL:=1

include $(INCLUDE_DIR)/package.mk

define Package/cros-iperf
  SECTION:=net
  CATEGORY:=Network
  DEPENDS:=+libstdcpp +libpthread
  TITLE:=Internet Protocol bandwidth measuring tool
  URL:=http://sourceforge.net/projects/iperf/
endef

define Package/cros-iperf/description
 Iperf is a modern alternative for measuring TCP and UDP bandwidth
 performance, allowing the tuning of various parameters and
 characteristics.
endef

TARGET_CFLAGS += -D_GNU_SOURCE
CONFIGURE_ARGS += --disable-multicast
CONFIGURE_ARGS += --enable-threads=posix

CONFIGURE_VARS += CXXFLAGS="$$$$CXXFLAGS -fno-rtti"
CONFIGURE_VARS += LIBS="-lpthread"

define Package/cros-iperf/install
	$(INSTALL_DIR) $(1)/usr/bin
	$(INSTALL_BIN) $(PKG_BUILD_DIR)/src/iperf $(1)/usr/bin/iperf
endef

$(eval $(call BuildPackage,cros-iperf))
