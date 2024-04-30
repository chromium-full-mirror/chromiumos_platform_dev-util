// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"go.chromium.org/chromiumos/config/go/test/api"

	dut_api "go.chromium.org/chromiumos/config/go/test/lab/api"
)

// GenerateDynamicInfo creates dynamic updates for provision
// requests, and adds their relevant information to each
// scheduling unit's dynamic lookup table.
func GenerateDynamicInfo(req *api.InternalTestplan, specs *FirmwareSpecs) error {
	// Create Dynamic Updates.
	if err := generateProvisionRequests(req, specs); err != nil {
		return err
	}

	// Add provision/DUT related information to dynamic
	// lookup table for resolving placeholders
	// found within generated Dynamic Updates.
	generateDynamicUpdateLookupTables(req, specs)

	return nil
}

// generateProvisionRequests loops through each swarming definition and
// builds out the firmware provision request.
func generateProvisionRequests(req *api.InternalTestplan, specs *FirmwareSpecs) error {
	dynamicHelper := NewDynamicFirmwareProvisionHelper(specs)

	// Legacy primary
	suiteMetadata := req.GetSuiteInfo().GetSuiteMetadata()
	if len(suiteMetadata.GetTargetRequirements()) > 0 {
		hwDef := suiteMetadata.GetTargetRequirements()[0].GetHwRequirements().GetHwDefinition()
		if len(hwDef) > 0 {
			swarmingDef := hwDef[0]
			dynamicHelper.GenerateProvisionRequest(req, swarmingDef)
		}
	}

	if len(suiteMetadata.GetSchedulingUnits()) > 0 {
		schedulingUnit := suiteMetadata.GetSchedulingUnits()[0]
		swarmingDef := schedulingUnit.GetPrimaryTarget().GetSwarmingDef()
		dynamicHelper.GenerateProvisionRequest(req, swarmingDef)

		for _, companion := range schedulingUnit.GetCompanionTargets() {
			swarmingDef := companion.GetSwarmingDef()
			dynamicHelper.GenerateProvisionRequest(req, swarmingDef)
		}
	}

	return nil
}

// generateDynamicUpdateLookupTables populates the lookup table for the primary
// and companion devices.
func generateDynamicUpdateLookupTables(req *api.InternalTestplan, specs *FirmwareSpecs) {
	suiteMetadata := req.GetSuiteInfo().GetSuiteMetadata()

	for _, target := range suiteMetadata.GetSchedulingUnits() {
		dynamicHelper := NewDynamicFirmwareProvisionHelper(specs)
		if target.DynamicUpdateLookupTable == nil {
			target.DynamicUpdateLookupTable = map[string]string{}
		}
		lookup := target.DynamicUpdateLookupTable

		// Do primary
		primarySwarming := target.PrimaryTarget.GetSwarmingDef()
		addFwProvisionValuesToLookup(lookup, primarySwarming, dynamicHelper, specs)

		// Do companions
		for _, companion := range target.GetCompanionTargets() {
			swarmingDef := companion.GetSwarmingDef()
			addFwProvisionValuesToLookup(lookup, swarmingDef, dynamicHelper, specs)
		}
	}

	// Support legacy.
	for _, targetReq := range suiteMetadata.GetTargetRequirements() {
		for _, hwDef := range targetReq.GetHwRequirements().GetHwDefinition() {
			dynamicHelper := NewDynamicFirmwareProvisionHelper(specs)
			if hwDef.DynamicUpdateLookupTable == nil {
				hwDef.DynamicUpdateLookupTable = map[string]string{}
			}
			lookup := hwDef.DynamicUpdateLookupTable
			addFwProvisionValuesToLookup(lookup, hwDef, dynamicHelper, specs)
		}
	}
}

// addFwProvisionValuesToLookup provides the actual values that will be
// stored within the lookup table for firmware provisioning.
func addFwProvisionValuesToLookup(
	lookup map[string]string,
	swarmingDef *api.SwarmingDefinition,
	dynamicHelper *DynamicFirmwareProvisionHelper,
	specs *FirmwareSpecs) {

	switch swarmingDef.GetDutInfo().GetDutType().(type) {
	case *dut_api.Dut_Chromeos:
		lookupValues := &FirmwareProvisionLookupValues{}

		// TODO(jbettis): Populate these fields,
		// Modify the function parameters as needed if you need any
		// other values to fetch these.
		// If Ro and Rw are not enough, extend the struct and corresponding
		// helper functions by following the pattern present.
		// lookupValues.Ro =
		// lookupValues.Rw =

		dynamicHelper.ApplyFirmwareProvisionToLookup(lookup, lookupValues)
	}
}
