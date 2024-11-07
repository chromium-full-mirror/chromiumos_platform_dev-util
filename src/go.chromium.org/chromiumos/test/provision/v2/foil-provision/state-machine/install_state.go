// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package state_machine

import (
	"context"
	"fmt"
	"log"

	common_utils "go.chromium.org/chromiumos/test/provision/v2/common-utils"
	"go.chromium.org/chromiumos/test/provision/v2/foil-provision/service"
	"go.chromium.org/chromiumos/test/provision/v2/foil-provision/state-machine/commands"

	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/protobuf/types/known/anypb"
)

// FoilInstallState can be thought of as the constructor state, which initializes
// variables in FoilService
type FoilInstallState struct {
	service *service.FoilService
}

func NewFoilInstallState(service *service.FoilService) common_utils.ServiceState {
	return FoilInstallState{
		service: service,
	}
}

func (s FoilInstallState) Execute(ctx context.Context, log *log.Logger) (*anypb.Any, api.InstallResponse_Status, error) {
	log.Printf("State: Execute FoilInstallState")
	comms := []common_utils.CommandInterface{

		commands.NewSetSelinuxCommandSetup(ctx, s.service),
		commands.NewInstall(ctx, s.service),
		commands.NewRebootCommand(ctx, s.service),
	}

	for _, comm := range comms {
		err := comm.Execute(log)
		if err != nil {
			return common_utils.WrapStringInAny(comm.GetErrorMessage()), comm.GetStatus(), fmt.Errorf("%s, %s", comm.GetErrorMessage(), err)
		}
	}
	log.Printf("State: FoilInstallState Completed")
	return nil, api.InstallResponse_STATUS_SUCCESS, nil
}

func (s FoilInstallState) Next() common_utils.ServiceState {
	return FoilPostState{
		service: s.service,
	}
}

func (s FoilInstallState) Name() string {
	return "Foil Install"
}
