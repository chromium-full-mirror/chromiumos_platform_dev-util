// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cross_over

import (
	"context"
	"log"

	common_utils "go.chromium.org/chromiumos/test/provision/v2/common-utils"

	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/protobuf/types/known/anypb"
)

// RebootFromDiskState boots the DUT from image on its disk.
type RebootFromDiskState struct {
	servodClient api.ServodServiceClient
	params       *CrossOverParameters
}

func NewRebootFromDiskState(servodClient api.ServodServiceClient, params *CrossOverParameters) common_utils.ServiceState {
	return &RebootFromDiskState{
		servodClient: servodClient,
		params:       params,
	}
}

func (s RebootFromDiskState) Execute(ctx context.Context, log *log.Logger) (*anypb.Any, api.InstallResponse_Status, error) {
	return nil, api.InstallResponse_STATUS_SUCCESS, nil
}

func (s RebootFromDiskState) Next() common_utils.ServiceState {
	return NewCrossOverTerminateState(s.servodClient, s.params)
}

func (s RebootFromDiskState) Name() string {
	return "Reboot From Disk State"
}
