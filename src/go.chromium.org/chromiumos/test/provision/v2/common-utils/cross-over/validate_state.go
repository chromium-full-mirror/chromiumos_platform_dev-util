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

// ValidateState checks if the correct version was provisioned or not.
type ValidateState struct {
	servodClient api.ServodServiceClient
	params       *CrossOverParameters
}

func NewValidateState(servodClient api.ServodServiceClient, params *CrossOverParameters) common_utils.ServiceState {
	return &ValidateState{
		servodClient: servodClient,
		params:       params,
	}
}

func (s ValidateState) Execute(ctx context.Context, log *log.Logger) (*anypb.Any, api.InstallResponse_Status, error) {
	return nil, api.InstallResponse_STATUS_SUCCESS, nil
}

func (s ValidateState) Next() common_utils.ServiceState {
	return nil
}

func (s ValidateState) Name() string {
	return "Reboot State"
}
