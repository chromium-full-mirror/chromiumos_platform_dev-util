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

const waitForPowerOff = 30 * time.Second

// RebootFromUSBState sets the DUT power off and bring it back up in rec mode.
type RebootFromUSBState struct {
	servodClient api.ServodServiceClient
	params       *CrossOverParameters
}

func NewRebootFromUSBState(servodClient api.ServodServiceClient, params *CrossOverParameters) common_utils.ServiceState {
	return &RebootFromUSBState{
		servodClient: servodClient,
		params:       params,
	}
}

func (s RebootFromUSBState) Execute(ctx context.Context, log *log.Logger) (*anypb.Any, api.InstallResponse_Status, error) {
	log.Println("Executing " + s.Name())
	servoHostPath := s.params.Dut.GetChromeos().GetServo().GetServodAddress().GetAddress() + sshPort
	servoPort := s.params.Dut.GetChromeos().GetServo().GetServodAddress().GetPort()
	servodDockerContainerName := ""
	if strings.Contains(servoHostPath, satlab) {
		servodDockerContainerName = s.params.Dut.GetChromeos().GetServo().GetServodAddress().GetAddress()
	}

	if err := callServod(ctx, log, s.servodClient, "power_state", "off", servoHostPath, servodDockerContainerName, servoPort); err != nil {
		return nil, api.InstallResponse_STATUS_PROVISIONING_FAILED, err
	}
	log.Printf("\nWaiting for the power state to turn off.")
	// TODO: Instead of fixed wait time, use a poll based status checker. Refer WaitForPowerStates and GetECSystemPowerState in firmware code.
	time.Sleep(waitForPowerOff)
	if err := callServod(ctx, log, s.servodClient, "power_state", "rec", servoHostPath, servodDockerContainerName, servoPort); err != nil {
		return nil, api.InstallResponse_STATUS_PROVISIONING_FAILED, err
	}
	// TODO: Use better way to wait than a constant time.
	log.Println("Waiting for power state rec 2 minutes.")
	time.Sleep(2 * time.Minute)
	return nil, api.InstallResponse_STATUS_SUCCESS, nil
}

func (s RebootFromUSBState) Next() common_utils.ServiceState {
	return NewFullOSImageState(s.servodClient, s.params)
}

func (s RebootFromUSBState) Name() string {
	return "Reboot from USB State"
}
