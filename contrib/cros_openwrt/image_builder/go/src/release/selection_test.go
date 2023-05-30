// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package release

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
	labapi "go.chromium.org/chromiumos/config/go/test/lab/api"
)

func randomUUID() string {
	return uuid.New().String()
}

func TestSelectImageByUUID(t *testing.T) {
	imageConfig1 := &labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage{
		ImageUuid: randomUUID(),
	}
	imageConfig2 := &labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage{
		ImageUuid: randomUUID(),
	}
	imageConfig3 := &labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage{
		ImageUuid: randomUUID(),
	}
	type args struct {
		deviceConfig *labapi.OpenWrtWifiRouterDeviceConfig
		imageUUID    string
	}
	tests := []struct {
		name    string
		args    args
		want    *labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage
		wantErr bool
	}{
		{
			"no available images",
			args{
				&labapi.OpenWrtWifiRouterDeviceConfig{},
				imageConfig1.ImageUuid,
			},
			nil,
			true,
		},
		{
			"no matching images",
			args{
				&labapi.OpenWrtWifiRouterDeviceConfig{
					Images: []*labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage{
						imageConfig2,
						imageConfig3,
					},
				},
				imageConfig1.ImageUuid,
			},
			nil,
			true,
		},
		{
			"match found",
			args{
				&labapi.OpenWrtWifiRouterDeviceConfig{
					Images: []*labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage{
						imageConfig2,
						imageConfig1,
						imageConfig3,
					},
				},
				imageConfig1.ImageUuid,
			},
			imageConfig1,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SelectImageByUUID(tt.args.deviceConfig, tt.args.imageUUID)
			if (err != nil) != tt.wantErr {
				t.Errorf("SelectImageByUUID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SelectImageByUUID() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSelectCurrentImage(t *testing.T) {
	imageConfig1 := &labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage{
		ImageUuid: randomUUID(),
	}
	type args struct {
		deviceConfig *labapi.OpenWrtWifiRouterDeviceConfig
	}
	tests := []struct {
		name    string
		args    args
		want    *labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage
		wantErr bool
	}{
		{
			"no current image configured",
			args{
				deviceConfig: &labapi.OpenWrtWifiRouterDeviceConfig{
					Images: []*labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage{
						imageConfig1,
					},
					CurrentImageUuid: "",
				},
			},
			nil,
			true,
		},
		{
			"current image configured",
			args{
				deviceConfig: &labapi.OpenWrtWifiRouterDeviceConfig{
					Images: []*labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage{
						imageConfig1,
					},
					CurrentImageUuid: imageConfig1.ImageUuid,
				},
			},
			imageConfig1,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SelectCurrentImage(tt.args.deviceConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("SelectCurrentImage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SelectCurrentImage() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSelectNextImage(t *testing.T) {
	imageConfig1 := &labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage{
		ImageUuid: randomUUID(),
	}
	type args struct {
		deviceConfig *labapi.OpenWrtWifiRouterDeviceConfig
	}
	tests := []struct {
		name    string
		args    args
		want    *labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage
		wantErr bool
	}{
		{
			"no next image configured",
			args{
				deviceConfig: &labapi.OpenWrtWifiRouterDeviceConfig{
					Images: []*labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage{
						imageConfig1,
					},
					NextImageUuid: "",
				},
			},
			nil,
			true,
		},
		{
			"next image configured",
			args{
				deviceConfig: &labapi.OpenWrtWifiRouterDeviceConfig{
					Images: []*labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage{
						imageConfig1,
					},
					NextImageUuid: imageConfig1.ImageUuid,
				},
			},
			imageConfig1,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SelectNextImage(tt.args.deviceConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("SelectNextImage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SelectNextImage() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSelectImageByCrosReleaseVersion(t *testing.T) {
	imageConfig1 := &labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage{
		ImageUuid:            randomUUID(),
		MinDutReleaseVersion: "10",
	}
	imageConfig2 := &labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage{
		ImageUuid:            randomUUID(),
		MinDutReleaseVersion: "20",
	}
	imageConfig3A := &labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage{
		ImageUuid:            randomUUID(),
		MinDutReleaseVersion: "30",
	}
	imageConfig3B := &labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage{
		ImageUuid:            randomUUID(),
		MinDutReleaseVersion: "30",
	}
	imageConfig4 := &labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage{
		ImageUuid:            randomUUID(),
		MinDutReleaseVersion: "40",
	}
	deviceConfig1 := &labapi.OpenWrtWifiRouterDeviceConfig{
		Images: []*labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage{
			imageConfig1,
			imageConfig2,
			imageConfig3A,
			imageConfig3B,
			imageConfig4,
		},
		CurrentImageUuid: imageConfig4.ImageUuid,
	}
	type args struct {
		deviceConfig          *labapi.OpenWrtWifiRouterDeviceConfig
		dutCrosReleaseVersion string
		useCurrentIfNoMatches bool
	}
	tests := []struct {
		name    string
		args    args
		want    *labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage
		wantErr bool
	}{
		{
			"no available images",
			args{
				&labapi.OpenWrtWifiRouterDeviceConfig{},
				"1",
				false,
			},
			nil,
			true,
		},
		{
			"bad version",
			args{
				deviceConfig1,
				"abc",
				false,
			},
			nil,
			true,
		},
		{
			"no matching images",
			args{
				deviceConfig1,
				"1",
				false,
			},
			nil,
			true,
		},
		{
			"no matching images, default to current",
			args{
				deviceConfig1,
				"1",
				true,
			},
			imageConfig4,
			false,
		},
		{
			"single matching image; version greater than min",
			args{
				deviceConfig1,
				"25",
				false,
			},
			imageConfig2,
			false,
		},
		{
			"single matching image; version same as min",
			args{
				deviceConfig1,
				"20",
				false,
			},
			imageConfig2,
			false,
		},
		{
			"multiple matching images with same min version",
			args{
				deviceConfig1,
				"35",
				false,
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SelectImageByCrosReleaseVersion(tt.args.deviceConfig, tt.args.dutCrosReleaseVersion, tt.args.useCurrentIfNoMatches)
			if (err != nil) != tt.wantErr {
				t.Errorf("SelectImageByCrosReleaseVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SelectImageByCrosReleaseVersion() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSelectImageForDut(t *testing.T) {
	dut1 := "dut1"
	dut2 := "dut2"
	imageConfig1 := &labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage{
		ImageUuid:            randomUUID(),
		MinDutReleaseVersion: "10",
	}
	imageConfig2 := &labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage{
		ImageUuid:            randomUUID(),
		MinDutReleaseVersion: "20",
	}
	imageConfig3 := &labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage{
		ImageUuid:            randomUUID(),
		MinDutReleaseVersion: "30",
	}
	deviceConfig1 := &labapi.OpenWrtWifiRouterDeviceConfig{
		Images: []*labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage{
			imageConfig1,
			imageConfig2,
			imageConfig3,
		},
		CurrentImageUuid: imageConfig2.ImageUuid,
		NextImageUuid:    imageConfig3.ImageUuid,
		NextImageVerificationDutPool: []string{
			dut1,
		},
	}
	type args struct {
		deviceConfig          *labapi.OpenWrtWifiRouterDeviceConfig
		dutHostname           string
		dutCrosReleaseVersion string
	}
	tests := []struct {
		name    string
		args    args
		want    *labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage
		wantErr bool
	}{
		{
			"bad version",
			args{
				deviceConfig1,
				dut1,
				"abc",
			},
			nil,
			true,
		},
		{
			"dut in next pool should return next image and no version specified",
			args{
				deviceConfig1,
				dut1,
				"",
			},
			imageConfig3,
			false,
		},
		{
			"dut in next pool should return next image and version matches image",
			args{
				deviceConfig1,
				dut1,
				"35",
			},
			imageConfig3,
			false,
		},
		{
			"dut in next pool should return next image and version does not match image",
			args{
				deviceConfig1,
				dut1,
				"1",
			},
			imageConfig3,
			false,
		},
		{
			"use current when no version specified",
			args{
				deviceConfig1,
				dut2,
				"",
			},
			imageConfig2,
			false,
		},
		{
			"use current when version matches current image",
			args{
				deviceConfig1,
				dut2,
				"25",
			},
			imageConfig2,
			false,
		},
		{
			"use next highest image when version does not match current image",
			args{
				deviceConfig1,
				dut2,
				"15",
			},
			imageConfig1,
			false,
		},
		{
			"use current when version does not match current image but no other images match either",
			args{
				deviceConfig1,
				dut2,
				"5",
			},
			imageConfig2,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SelectImageForDut(tt.args.deviceConfig, tt.args.dutHostname, tt.args.dutCrosReleaseVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("SelectImageForDut() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SelectImageForDut() got = %v, want %v", got, tt.want)
			}
		})
	}
}
