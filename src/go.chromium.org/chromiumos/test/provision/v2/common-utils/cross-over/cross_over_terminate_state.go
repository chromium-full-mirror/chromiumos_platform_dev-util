// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// DLC constants and helpers
package cross_over

import (
	"context"
	"log"
	"strings"
	"time"

	common_utils "go.chromium.org/chromiumos/test/provision/v2/common-utils"

	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/protobuf/types/known/anypb"
)

// CrossOverTerminateState stops the servod process and downloads the servod logs.
type CrossOverTerminateState struct {
	servodClient api.ServodServiceClient
	params       *CrossOverParameters
}

func NewCrossOverTerminateState(servodClient api.ServodServiceClient, params *CrossOverParameters) common_utils.ServiceState {
	return &CrossOverTerminateState{
		servodClient: servodClient,
		params:       params,
	}
}

func (s CrossOverTerminateState) Execute(ctx context.Context, log *log.Logger) (*anypb.Any, api.InstallResponse_Status, error) {
	log.Println("Executing " + s.Name())
	req := &api.StopServodRequest{}
	req.ServodPort = s.params.Dut.GetChromeos().GetServo().GetServodAddress().GetPort()
	req.ServoHostPath = s.params.Dut.GetChromeos().GetServo().GetServodAddress().GetAddress() + sshPort
	if strings.Contains(req.ServoHostPath, satlab) {
		req.ServodDockerContainerName = s.params.Dut.GetChromeos().GetServo().GetServodAddress().GetAddress()
	}
	log.Printf("\nCalling stop servod with request %v.", req)
	op, err := s.servodClient.StopServod(ctx, req)
	if err != nil {
		return nil, api.InstallResponse_STATUS_POST_PROVISION_SETUP_FAILED, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = common_utils.WaitLongRunningOp(ctx, log, op)
	if err != nil {
		return nil, api.InstallResponse_STATUS_POST_PROVISION_SETUP_FAILED, err
	}
	log.Printf("\nStopped servod process")
	// TODO: Code change to download the logs.
	return nil, api.InstallResponse_STATUS_SUCCESS, nil
}

func (s CrossOverTerminateState) Next() common_utils.ServiceState {
	return s.params.PostProvisionState
}

func (s CrossOverTerminateState) Name() string {
	return "Cross Over Terminate State"
}
