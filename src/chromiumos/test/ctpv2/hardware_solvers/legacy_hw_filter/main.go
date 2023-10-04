// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	server "chromiumos/test/ctpv2/common/server_template"
	"log"
	"os"

	dut_api "go.chromium.org/chromiumos/config/go/test/lab/api"
	"google.golang.org/protobuf/proto"

	"go.chromium.org/chromiumos/config/go/test/api"
)

const (
	address = "localhost"
	binName = "legacy_hw_filter"
)

// hwTargetsFromReq returns the list of hwTargets from the suiteMetadata definition.
func hwTargetsFromReq(req *api.InternalTestplan) (hwTargets []*api.SwarmingDefinition) {
	for _, target := range req.GetSuiteInfo().GetSuiteMetadata().GetTargetRequirements() {
		for _, hwTarget := range target.GetHwRequirements().GetHwDefinition() {
			hwTargets = append(hwTargets, hwTarget)
			// Placeholder code for now.
			// TODO: We might need to swap this to UUIDs
			// Or something else due to proto size limitations on larger suites.
			// This switch/case shows how to pull the board out.
			// Though we will also likely need the model, and possibly other markers as well.
			switch hw := hwTarget.GetDutInfo().GetDutType().(type) {
			case *dut_api.Dut_Chromeos:
				_ = hw.Chromeos.GetDutModel().GetBuildTarget()
			default:
				continue
			}

		}
	}
	return hwTargets

}

// tcDeps returns a list of tc deps as a string.
func tcDeps(tc *api.CTPTestCase) (deps []string) {
	for _, dep := range tc.GetMetadata().GetTestCase().GetDependencies() {
		deps = append(deps, dep.GetValue())
	}
	return deps
}

// generateTcHwReqs will return a list that represents all HW the test
// is to be scheduled on. Each testcase will need its own unique list;
// which will later sharded/organized based on HW/SW requirements.
func generateTcHwReqs(hwTargets []*api.SwarmingDefinition, tcDeps []string) (tcHwReqs []*api.HWRequirements) {

	for _, target := range hwTargets {
		// We need to clone the target as different tests can set different hw requirements.
		// If we just set the field directly then it would change for all tests every time we change
		// the field.
		t := proto.Clone(target).(*api.SwarmingDefinition)
		t.SwarmingLabels = tcDeps
		targ := &api.HWRequirements{
			// For legacy solver, each HwDefinition is a list of 1
			// This is because for a test dut matching will follow this logic:
			// HwDefinition = [[a], [b]]; must run the test on a AND b
			// HwDefinition = [[a, b], [c]]; must run on A OR B, AND C
			// Legacy always lists of 1 item. Others, eg 3D; will give OR options.
			HwDefinition: []*api.SwarmingDefinition{t},
			State:        api.HWRequirements_REQUIRED,
		}
		tcHwReqs = append(tcHwReqs, targ)
	}

	return tcHwReqs
}

func executor(req *api.InternalTestplan, log *log.Logger) (*api.InternalTestplan, error) {
	// Step 1. Get all the HWTargets from the suite metadata
	// these might contain provision info.
	var hwTargets = hwTargetsFromReq(req)

	// Step 2.  Iterate through the tests, adding the HW to each test
	// Include the deps from the test metadata into the HW.
	for _, tc := range req.GetTestCases() {
		tcDeps := tcDeps(tc)
		tc.HwRequirements = generateTcHwReqs(hwTargets, tcDeps)
	}

	return req, nil
}

func main() {
	err := server.Server(executor, binName)
	if err != nil {
		os.Exit(2)
	}

	os.Exit(0)
}
