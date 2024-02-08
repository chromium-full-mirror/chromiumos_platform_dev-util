// Copyright 2022 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// First step of FirmwareService State Machine. Installs RW firmware.
package state_machine

import (
	common_utils "chromiumos/test/provision/v2/common-utils"
	firmwareservice "chromiumos/test/provision/v2/cros-fw-provision/service"
	"context"
	"fmt"
	"log"

	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/protobuf/types/known/anypb"
)

// FirmwareUpdateRwState updates firmware with write protection disabled.
type FirmwareUpdateRwState struct {
	service *firmwareservice.FirmwareService
}

// Execute flashes firmware using futility with write-protection enabled.
func (s FirmwareUpdateRwState) Execute(ctx context.Context, log *log.Logger) (*anypb.Any, api.InstallResponse_Status, error) {
	connection := s.service.GetConnectionToFlashingDevice()
	// form futility command args based on the request
	var futilityImageArgs []string
	// Detailed Request
	mainRwMetadata, ok := s.service.GetImageMetadata(s.service.GetMainRwPath())
	if !ok {
		panic(ok) // if nil, current state of the statemachine should not started
	}
	log.Printf("[FW Provisioning: Update RW] extracting AP image to flash\n")
	mainRwPath, err := firmwareservice.PickAndExtractMainImage(ctx, connection, mainRwMetadata, s.service.GetMainRwPath(), s.service)
	if err != nil {
		return nil, api.InstallResponse_STATUS_UPDATE_FIRMWARE_FAILED, firmwareservice.UpdateFirmwareFailedErr(err)
	}
	futilityImageArgs = []string{fmt.Sprint("--image=", mainRwPath)}

	log.Printf("[FW Provisioning: Update RW] flashing RW firmware with futility\n")
	err = s.service.FlashWithFutility(ctx, true /* WP */, futilityImageArgs, mainRwPath)
	if err != nil {
		return nil, api.InstallResponse_STATUS_UPDATE_FIRMWARE_FAILED, firmwareservice.UpdateFirmwareFailedErr(err)
	}
	return nil, api.InstallResponse_STATUS_OK, nil
}

func (s FirmwareUpdateRwState) Next() common_utils.ServiceState {
	return FirmwarePostInstallState(s)
}

const UpdateRwStateName = "Firmware Update RW"

func (s FirmwareUpdateRwState) Name() string {
	return UpdateRwStateName
}
