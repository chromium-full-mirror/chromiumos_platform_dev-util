// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// DLC constants and helpers
package cross_over

import (
	"context"
	"log"
	"time"

	lab_api "go.chromium.org/chromiumos/config/go/test/lab/api"
	common_utils "go.chromium.org/chromiumos/test/provision/v2/common-utils"

	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/protobuf/types/known/anypb"
)

type CrossOverParameters struct {
	Dut                       *lab_api.Dut
	ServodDockerContainerName string
	ServodDockerImagePath     string
}

// CrossOverInitState starts the servod process.
type CrossOverInitState struct {
	crosServodClient api.ServodServiceClient
	params           *CrossOverParameters
}

func NewCrossOverInitState(servodClient api.ServodServiceClient, params *CrossOverParameters) common_utils.ServiceState {
	return &CrossOverInitState{
		crosServodClient: servodClient,
		params:           params,
	}
}

func (s CrossOverInitState) Execute(ctx context.Context, log *log.Logger) (*anypb.Any, api.InstallResponse_Status, error) {
	req := &api.StartServodRequest{}
	req.Board = s.params.Dut.GetChromeos().GetDutModel().GetBuildTarget()
	req.Model = s.params.Dut.GetChromeos().GetDutModel().GetModelName()
	req.SerialName = s.params.Dut.GetChromeos().GetServo().GetSerial()
	req.ServodPort = s.params.Dut.GetChromeos().GetServo().GetServodAddress().GetPort()
	req.ServoHostPath = s.params.Dut.GetChromeos().GetServo().GetServodAddress().GetAddress()
	req.ServodDockerContainerName = s.params.ServodDockerContainerName
	req.ServodDockerImagePath = s.params.ServodDockerImagePath
	log.Printf("\nCalling start servod with request %v.", req)
	op, err := s.crosServodClient.StartServod(ctx, req)
	if err != nil {
		return nil, api.InstallResponse_STATUS_PRE_PROVISION_SETUP_FAILED, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = common_utils.WaitLongRunningOp(ctx, log, op)
	if err != nil {
		return nil, api.InstallResponse_STATUS_PRE_PROVISION_SETUP_FAILED, err
	}
	log.Printf("\nStarted servod process")
	return nil, api.InstallResponse_STATUS_SUCCESS, nil
}

func (s CrossOverInitState) Next() common_utils.ServiceState {
	return NewSetupUSBState(s.crosServodClient, s.params)
}

func (s CrossOverInitState) Name() string {
	return "Cross Over Init State"
}
