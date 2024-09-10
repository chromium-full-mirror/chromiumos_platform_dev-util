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

// FullOSImageState copies the full Android/Cros image onto DUT disk.
type FullOSImageState struct {
	servodClient api.ServodServiceClient
	params       *CrossOverParameters
}

func NewFullOSImageState(servodClient api.ServodServiceClient, params *CrossOverParameters) common_utils.ServiceState {
	return &FullOSImageState{
		servodClient: servodClient,
		params:       params,
	}
}

func (s FullOSImageState) Execute(ctx context.Context, log *log.Logger) (*anypb.Any, api.InstallResponse_Status, error) {
	return nil, api.InstallResponse_STATUS_SUCCESS, nil
}

func (s FullOSImageState) Next() common_utils.ServiceState {
	return NewRebootFromDiskState(s.servodClient, s.params)
}

func (s FullOSImageState) Name() string {
	return "Full OS Image State"
}
