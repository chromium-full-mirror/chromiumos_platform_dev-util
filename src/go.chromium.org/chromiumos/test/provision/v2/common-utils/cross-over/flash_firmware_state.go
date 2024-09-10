// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// DLC constants and helpers
package cross_over

import (
	"context"
	"log"

	common_utils "go.chromium.org/chromiumos/test/provision/v2/common-utils"

	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/protobuf/types/known/anypb"
)

// FlashFirmwareState installs a new firmware on DUT.
type FlashFirmwareState struct {
	servodClient api.ServodServiceClient
	params       *CrossOverParameters
}

func NewFlashFirmwareState(servodClient api.ServodServiceClient, params *CrossOverParameters) common_utils.ServiceState {
	return &FlashFirmwareState{
		servodClient: servodClient,
		params:       params,
	}
}

func (s FlashFirmwareState) Execute(ctx context.Context, log *log.Logger) (*anypb.Any, api.InstallResponse_Status, error) {
	return nil, api.InstallResponse_STATUS_SUCCESS, nil
}

func (s FlashFirmwareState) Next() common_utils.ServiceState {
	return NewFullOSImageState(s.servodClient, s.params)
}

func (s FlashFirmwareState) Name() string {
	return "Flash Firmware State"
}
