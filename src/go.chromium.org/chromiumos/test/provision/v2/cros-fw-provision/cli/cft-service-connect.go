// Copyright 2022 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Provides functionality to connect to other CFT services.
package cli

import (
	common_utils "go.chromium.org/chromiumos/test/provision/v2/common-utils"

	"github.com/pkg/errors"
	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/grpc"
)

func connectToDutServer(dutServAddr string) (*common_utils.ServiceAdapter, error) {
	dutConn, err := grpc.Dial(dutServAddr, grpc.WithInsecure())
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to Dut Server")
	}
	dutServerAdapter := common_utils.NewServiceAdapter(
		api.NewDutServiceClient(dutConn), false /*noReboot*/)
	return &dutServerAdapter, nil
}

func connectToCrosServod(crosServodAddr string) (api.ServodServiceClient, error) {
	servodConn, err := grpc.Dial(crosServodAddr, grpc.WithInsecure())
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to cros-servod")
	}
	servodServiceClient := api.NewServodServiceClient(servodConn)
	return servodServiceClient, nil
}
