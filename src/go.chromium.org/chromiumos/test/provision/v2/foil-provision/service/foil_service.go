// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Container for the FoilProvision state machine
package service

import (
	common_utils "go.chromium.org/chromiumos/test/provision/v2/common-utils"
	"go.chromium.org/chromiumos/test/provision/v2/common-utils/metadata"

	conf "go.chromium.org/chromiumos/config/go"
	"go.chromium.org/chromiumos/config/go/test/api"
	lab_api "go.chromium.org/chromiumos/config/go/test/lab/api"
)

// FoilService inherits ServiceInterface
type FoilService struct {
	Connection       common_utils.ServiceAdapterInterface
	MachineMetadata  metadata.MachineMetadata
	ImagePath        *conf.StoragePath
	OverwritePayload *conf.StoragePath
	SkipUpdate       bool
	QuickResetDevice bool
	DutIp            string
	UpdateEnginePid  string
	CurrentBuild     string
	TargetBuild      string
}

func NewFoilService(dut *lab_api.Dut, dutClient api.DutServiceClient, req *api.InstallRequest) (*FoilService, error) {

	return &FoilService{
		DutIp:       dut.GetChromeos().GetSsh().GetAddress(),
		TargetBuild: "P78687640",
	}, nil
}

// CleanupOnFailure is called if one of service's states fails to Execute() and
// should clean up the temporary files, and undo the execution, if feasible.
func (c *FoilService) CleanupOnFailure(states []common_utils.ServiceState, executionErr error) error {
	// TODO: evaluate whether cleanup is needed.
	return nil
}
