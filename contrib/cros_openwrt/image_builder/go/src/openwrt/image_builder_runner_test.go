// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package openwrt

import (
	"reflect"
	"testing"

	labapi "go.chromium.org/chromiumos/config/go/test/lab/api"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func prettyPrintProtoMessage(message proto.Message) string {
	if message == nil {
		return "nil"
	}
	marshaller := protojson.MarshalOptions{Multiline: true, Indent: "  "}
	result, err := marshaller.Marshal(message)
	if err != nil {
		panic("failed to marshal message")
	}
	return string(result)
}

func TestImageBuilderRunner_parseStandardBuildInfo(t *testing.T) {
	sampleOutput := `
Current Target: "ramips/mt7621"
Current Revision: "r16688-fa9a932fdb"
Default Packages: base-files ca-bundle dropbear
Available Profiles:

ubnt_edgerouter-x-sfp:
    Ubiquiti EdgeRouter X SFP
    Packages: -wpad-basic-wolfssl kmod-i2c-algo-pca kmod-gpio-pca953x kmod-sfp
    hasImageMetadata: 1
    SupportedDevices: ubnt,edgerouter-x-sfp ubnt-erx-sfp ubiquiti,edgerouterx-sfp
ubnt_unifi-6-lite:
    Ubiquiti UniFi 6 Lite
    Packages: kmod-mt7603 kmod-mt7915e
    hasImageMetadata: 1
    SupportedDevices: ubnt,unifi-6-lite
ubnt_unifi-nanohd:
    Ubiquiti UniFi nanoHD
    Packages: kmod-mt7603 kmod-mt7615e kmod-mt7615-firmware
    hasImageMetadata: 1
    SupportedDevices: ubnt,unifi-nanohd
`
	type args struct {
		makeInfoOutput string
		profileName    string
	}
	tests := []struct {
		name    string
		args    args
		want    *labapi.CrosOpenWrtImageBuildInfo_StandardBuildConfig
		wantErr bool
	}{
		{
			"empty output",
			args{
				"",
				"",
			},
			nil,
			true,
		},
		{
			"bad profile name",
			args{
				sampleOutput,
				"foo_bar",
			},
			nil,
			true,
		},
		{
			"can parse ubnt_unifi-6-lite",
			args{
				sampleOutput,
				"ubnt_unifi-6-lite",
			},
			&labapi.CrosOpenWrtImageBuildInfo_StandardBuildConfig{
				OpenwrtBuildTarget: "ramips/mt7621",
				OpenwrtRevision:    "r16688-fa9a932fdb",
				BuildTargetPackages: []string{
					"base-files",
					"ca-bundle",
					"dropbear",
				},
				BuildProfile:     "ubnt_unifi-6-lite",
				DeviceName:       "Ubiquiti UniFi 6 Lite",
				ProfilePackages:  []string{"kmod-mt7603", "kmod-mt7915e"},
				SupportedDevices: []string{"ubnt,unifi-6-lite"},
			},
			false,
		},
		{
			"can parse ubnt_edgerouter-x-sfp",
			args{
				sampleOutput,
				"ubnt_edgerouter-x-sfp",
			},
			&labapi.CrosOpenWrtImageBuildInfo_StandardBuildConfig{
				OpenwrtBuildTarget: "ramips/mt7621",
				OpenwrtRevision:    "r16688-fa9a932fdb",
				BuildTargetPackages: []string{
					"base-files",
					"ca-bundle",
					"dropbear",
				},
				BuildProfile: "ubnt_edgerouter-x-sfp",
				DeviceName:   "Ubiquiti EdgeRouter X SFP",
				ProfilePackages: []string{
					"-wpad-basic-wolfssl",
					"kmod-i2c-algo-pca",
					"kmod-gpio-pca953x",
					"kmod-sfp",
				},
				SupportedDevices: []string{
					"ubnt,edgerouter-x-sfp",
					"ubnt-erx-sfp",
					"ubiquiti,edgerouterx-sfp",
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ib := &ImageBuilderRunner{}
			got, err := ib.parseStandardBuildInfo(tt.args.makeInfoOutput, tt.args.profileName)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseStandardBuildInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseStandardBuildInfo() got = %v, want %v", prettyPrintProtoMessage(got), prettyPrintProtoMessage(tt.want))
			}
		})
	}
}

func TestImageBuilderRunner_parseOSReleaseFile(t *testing.T) {
	type args struct {
		osReleaseFileContents string
	}
	tests := []struct {
		name    string
		args    args
		want    *labapi.CrosOpenWrtImageBuildInfo_OSRelease
		wantErr bool
	}{
		{
			"empty content",
			args{""},
			nil,
			true,
		},
		{
			"sample content",
			args{`NAME="OpenWrt"
VERSION="21.02.5"
ID="openwrt"
ID_LIKE="lede openwrt"
PRETTY_NAME="OpenWrt 21.02.5"
VERSION_ID="21.02.5"
HOME_URL="https://openwrt.org/"
BUG_URL="https://bugs.openwrt.org/"
SUPPORT_URL="https://forum.openwrt.org/"
BUILD_ID="r16688-fa9a932fdb"
OPENWRT_BOARD="ramips/mt7621"
OPENWRT_ARCH="mipsel_24kc"
OPENWRT_TAINTS=""
OPENWRT_DEVICE_MANUFACTURER="OpenWrt"
OPENWRT_DEVICE_MANUFACTURER_URL="https://openwrt.org/"
OPENWRT_DEVICE_PRODUCT="Generic"
OPENWRT_DEVICE_REVISION="v0"
OPENWRT_RELEASE="OpenWrt 21.02.5 r16688-fa9a932fdb"

`},
			&labapi.CrosOpenWrtImageBuildInfo_OSRelease{
				Version:        "21.02.5",
				BuildId:        "r16688-fa9a932fdb",
				OpenwrtBoard:   "ramips/mt7621",
				OpenwrtArch:    "mipsel_24kc",
				OpenwrtRelease: "OpenWrt 21.02.5 r16688-fa9a932fdb",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ib := &ImageBuilderRunner{}
			got, err := ib.parseOSReleaseFile(tt.args.osReleaseFileContents)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseOSReleaseFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseOSReleaseFile() got = %v, want %v", prettyPrintProtoMessage(got), prettyPrintProtoMessage(tt.want))
			}
		})
	}
}
