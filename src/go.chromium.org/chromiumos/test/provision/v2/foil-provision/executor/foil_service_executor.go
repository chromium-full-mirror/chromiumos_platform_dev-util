// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Executor defines a state initializer for each state. Usable by server to start.
// Under normal conditions this would be part of service, but go being go, it
// would create an impossible cycle
package executor

import (
	common_utils "go.chromium.org/chromiumos/test/provision/v2/common-utils"

	"go.chromium.org/chromiumos/test/provision/v2/foil-provision/service"
	state_machine "go.chromium.org/chromiumos/test/provision/v2/foil-provision/state-machine"

	"go.chromium.org/chromiumos/config/go/test/api"
	lab_api "go.chromium.org/chromiumos/config/go/test/lab/api"
)

type FoilProvisionExecutor struct {
}

func (c *FoilProvisionExecutor) GetFirstState(dut *lab_api.Dut, dutClient api.DutServiceClient, req *api.InstallRequest) (common_utils.ServiceState, error) {
	cs, err := service.NewFoilService(dut, dutClient, req)
	if err != nil {
		return nil, err
	}
	return state_machine.NewFoilPreInitState(cs), nil
}

// Validate ensures the ProvisionStartupRequest meets specified requirements.
func (c *FoilProvisionExecutor) Validate(req *api.ProvisionStartupRequest) error {

	return nil
}
