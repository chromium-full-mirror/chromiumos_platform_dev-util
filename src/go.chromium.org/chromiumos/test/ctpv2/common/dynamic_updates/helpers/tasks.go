// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package helpers

import (
	_go "go.chromium.org/chromiumos/config/go"
	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/test/ctpv2/common/dynamic_updates/interfaces"
	"google.golang.org/protobuf/types/known/anypb"
)

// DefaultProvisionStartUpRequest outlines the default values
// typically required for starting up a provision task.
func DefaultProvisionStartUpRequest(deviceId *DeviceIdentifier) *interfaces.ProvisionTaskStartUpRequest {
	return &interfaces.ProvisionTaskStartUpRequest{
		DynamicInputs: map[string]string{
			"dut":       deviceId.GetDevice("dut"),
			"dutServer": deviceId.GetCrosDutServer(),
		},
	}
}

// DefaultCrosProvisionInstallRequest sets up a cros-provision request
// with an empty metadata field, solely passing along the installPath.
func DefaultCrosProvisionInstallRequest(installPath string) *interfaces.ProvisionTaskInstallRequest {
	crosProvisionMetadata, _ := anypb.New(&api.CrOSProvisionMetadata{})
	return &interfaces.ProvisionTaskInstallRequest{
		StaticImagePath: &_go.StoragePath{
			HostType: _go.StoragePath_GS,
			Path:     installPath,
		},
		StaticMetadata: crosProvisionMetadata,
	}
}
