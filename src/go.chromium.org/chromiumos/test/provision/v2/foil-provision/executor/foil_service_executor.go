// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Executor defines a state initializer for each state. Usable by server to start.
// Under normal conditions this would be part of service, but go being go, it
// would create an impossible cycle
package executor

import (
	"fmt"
	"log"

	common_utils "go.chromium.org/chromiumos/test/provision/v2/common-utils"
	cross_over "go.chromium.org/chromiumos/test/provision/v2/common-utils/cross-over"
	"go.chromium.org/chromiumos/test/provision/v2/foil-provision/service"
	state_machine "go.chromium.org/chromiumos/test/provision/v2/foil-provision/state-machine"
	"google.golang.org/grpc"

	"go.chromium.org/chromiumos/config/go/test/api"
	lab_api "go.chromium.org/chromiumos/config/go/test/lab/api"
)

type FoilProvisionExecutor struct {
	Logger *log.Logger
}

func NewFoilProvisionExecutor(logger *log.Logger) (*FoilProvisionExecutor, error) {
	return &FoilProvisionExecutor{
		Logger: logger,
	}, nil
}

func (c *FoilProvisionExecutor) GetFirstState(dut *lab_api.Dut, dutClient api.DutServiceClient, servoNexusAddr string, req *api.InstallRequest) (common_utils.ServiceState, error) {
	osType, err := cross_over.DetectOS(c.Logger, fmt.Sprintf("%v:%v", dut.GetChromeos().GetSsh().GetAddress(), dut.GetChromeos().GetSsh().GetPort()))
	if err != nil {
		c.Logger.Println("Could not detect the OS. Will perform CrossOver provision.")
	} else if osType == cross_over.ANDROID {
		c.Logger.Println("Android Detected")
	} else {
		c.Logger.Println("CROS Detected")
	}
	if err != nil || osType == cross_over.CROS {
		if servoNexusAddr == "" {
			return nil, fmt.Errorf("servoNexusAdd required for crossover provision")
		}
		params := &cross_over.CrossOverParameters{
			DutClient:       dutClient,
			Dut:             dut,
			TargetImagePath: req.GetImagePath(),
		}
		conn, err := grpc.Dial(servoNexusAddr, grpc.WithInsecure())
		if err != nil {
			return nil, err
		}
		crosServodClient := api.NewServodServiceClient(conn)
		return cross_over.NewCrossOverInitState(crosServodClient, params), nil
	}
	cs, err := service.NewFoilService(dut, req)
	if err != nil {
		return nil, err
	}
	return state_machine.NewFoilPreInitState(cs), nil
}

// Validate ensures the ProvisionStartupRequest meets specified requirements.
func (c *FoilProvisionExecutor) Validate(req *api.ProvisionStartupRequest) error {
	return nil
}
