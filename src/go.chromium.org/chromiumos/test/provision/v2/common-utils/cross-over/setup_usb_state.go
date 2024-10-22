// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// DLC constants and helpers
package cross_over

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	common_utils "go.chromium.org/chromiumos/test/provision/v2/common-utils"

	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/protobuf/types/known/anypb"
)

// TODO: Optimize so that we dont have to write to USB every single time.
// TODO: Set provisionOSImagePath when its ready. Currently using a cros test image as a placeholder.
// TODO: Post 9/23 deadline, remove provisionOSImagePath hardcoding here and input from provision request.
const (
	dutSeesUsbkeyWait = time.Second * 5
)

// SetupUSBState downloads the bootable provision image onto USB.
type SetupUSBState struct {
	servodClient api.ServodServiceClient
	params       *CrossOverParameters
}

func NewSetupUSBState(servodClient api.ServodServiceClient, params *CrossOverParameters) common_utils.ServiceState {
	return &SetupUSBState{
		servodClient: servodClient,
		params:       params,
	}
}

func (s SetupUSBState) Execute(ctx context.Context, log *log.Logger) (*anypb.Any, api.InstallResponse_Status, error) {
	log.Println("Executing " + s.Name())
	servoHostPath := s.params.Dut.GetChromeos().GetServo().GetServodAddress().GetAddress() + sshPort
	servoPort := s.params.Dut.GetChromeos().GetServo().GetServodAddress().GetPort()
	servodDockerContainerName := ""
	if strings.Contains(servoHostPath, satlab) {
		servodDockerContainerName = s.params.Dut.GetChromeos().GetServo().GetServodAddress().GetAddress()
	}
	board := s.params.Dut.GetChromeos().GetDutModel().GetBuildTarget()
	provisionOSImagePath := fmt.Sprintf("gs://chromeos-throw-away-bucket/kimjae/%s-provision-v2.bin", board)

	// TODO: Instead of passing cache server url, we should pass gs path and cros-servod should build cache url from it.
	provisionImgCacheServerPath := fmt.Sprintf("http://%s:%v/download/%s", s.params.Dut.GetCacheServer().GetAddress().GetAddress(), s.params.Dut.GetCacheServer().GetAddress().GetPort(), provisionOSImagePath[5:])
	// TODO: Move these two calls to servod into cros-servod as a composite API SetupUSBKey.
	// TODO: Deprecate primitive callServod and Exec API.
	if err := callServod(ctx, log, s.servodClient, "download_image_to_usb_dev", provisionImgCacheServerPath, servoHostPath, servodDockerContainerName, servoPort); err != nil {
		return nil, api.InstallResponse_STATUS_DOWNLOADING_IMAGE_FAILED, err
	}
	if err := callServod(ctx, log, s.servodClient, "image_usbkey_mux", "dut_sees_usbkey", servoHostPath, servodDockerContainerName, servoPort); err != nil {
		return nil, api.InstallResponse_STATUS_DOWNLOADING_IMAGE_FAILED, err
	}
	// Wait for some time for the dut_sees_usbkey to take affect.
	time.Sleep(dutSeesUsbkeyWait)
	return nil, api.InstallResponse_STATUS_SUCCESS, nil
}

func (s SetupUSBState) Next() common_utils.ServiceState {
	return NewRebootFromUSBState(s.servodClient, s.params)
}

func (s SetupUSBState) Name() string {
	return "Setup USB State"
}
