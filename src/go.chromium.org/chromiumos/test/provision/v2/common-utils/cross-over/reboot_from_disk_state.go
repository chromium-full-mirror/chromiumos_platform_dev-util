// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cross_over

import (
	"context"
	"log"
	"time"

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
	log.Println("Executing " + s.Name())
	servoHostPath := s.params.Dut.GetChromeos().GetServo().GetServodAddress().GetAddress() + sshPort
	servoPort := s.params.Dut.GetChromeos().GetServo().GetServodAddress().GetPort()
	if err := callServod(ctx, log, s.servodClient, "power_state", "off", servoHostPath, s.params.ServodDockerContainerName, servoPort); err != nil {
		return nil, api.InstallResponse_STATUS_PROVISIONING_FAILED, err
	}
	log.Printf("\nWaiting for the power state to turn off.")
	// TODO: Instead of fixed wait time, use a poll based status checker. Refer WaitForPowerStates and GetECSystemPowerState in firmware code.
	time.Sleep(waitForPowerOff)

	if err := callServod(ctx, log, s.servodClient, "image_usbkey_direction", "servo_sees_usbkey", servoHostPath, s.params.ServodDockerContainerName, servoPort); err != nil {
		return nil, api.InstallResponse_STATUS_PROVISIONING_FAILED, err
	}
	log.Printf("\nWaiting for the image_usbkey_direction.")
	// TODO: Instead of fixed wait time, use a poll based status checker. Refer WaitForPowerStates and GetECSystemPowerState in firmware code.
	time.Sleep(10 * time.Second)

	if err := callServod(ctx, log, s.servodClient, "power_state", "on", servoHostPath, s.params.ServodDockerContainerName, servoPort); err != nil {
		return nil, api.InstallResponse_STATUS_PROVISIONING_FAILED, err
	}
	log.Println("Waiting for power on.")
	time.Sleep(waitForPowerOff)
	return nil, api.InstallResponse_STATUS_SUCCESS, nil
}

func (s RebootFromDiskState) Next() common_utils.ServiceState {
	return NewCrossOverTerminateState(s.servodClient, s.params)
}

func (s RebootFromDiskState) Name() string {
	return "Reboot From Disk State"
}
