// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package state_machine - Pre check for State Machine. Responsible for checking image status and possibly skipping provision.
package state_machine

import (
	"context"
	"log"

	common_utils "go.chromium.org/chromiumos/test/provision/v2/common-utils"
	"go.chromium.org/chromiumos/test/provision/v2/foil-provision/service"

	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/protobuf/types/known/anypb"
)

// FoilPreInitState can be thought of as the constructor state, which initializes
// variables in FoilService
type FoilPreInitState struct {
	service    *service.FoilService
	shouldSkip bool
}

// NewFoilPreInitState provides an interface to FoilPreInitState.
func NewFoilPreInitState(service *service.FoilService) common_utils.ServiceState {
	return FoilPreInitState{
		service: service,
	}
}

// Execute executes the steps needed to support FoilPreInitState. Xheck if the chromeOS target == current, if so skip install.
func (s FoilPreInitState) Execute(ctx context.Context, log *log.Logger) (*anypb.Any, api.InstallResponse_Status, error) {
	log.Printf("State: Execute FoilPreInitState")

	log.Printf("ALL DONE")
	return nil, api.InstallResponse_STATUS_SUCCESS, nil
}

// Next provides the interface if the install flag is set.
func (s FoilPreInitState) Next() common_utils.ServiceState {
	return nil

}

// Name of the step.
func (s FoilPreInitState) Name() string {
	return "Foil Init"
}
