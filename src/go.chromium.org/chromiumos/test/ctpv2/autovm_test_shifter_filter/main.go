// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"fmt"
	"log"
	"os"

	server "go.chromium.org/chromiumos/test/ctpv2/common/server_template"

	"go.chromium.org/chromiumos/config/go/test/api"
	testapi "go.chromium.org/chromiumos/config/go/test/lab/api"
)

const (
	binName         = "autovm_test_shifter_filter"
	installPath     = "installPath"
	hwAgnostic      = "hw_agnostic"
	release         = "release/"
	gsBucketPath    = "gs://chromeos-image-archive/"
	buildReportJSON = "build_report.json"
	success         = "SUCCESS"
)

var vmBoards = []string{"betty", "reven-vmtest", "amd64-generic"}

// isTestVMCompatible returns true if the test can be safely executed on a VM.
func isTestVMCompatible(testCase *api.CTPTestCase) bool {
	compatibilityCriteria := []func(*api.CTPTestCase) bool{
		isHwAgnostic,
		// Add more criteria functions here in the future
	}

	for _, criterion := range compatibilityCriteria {
		if criterion(testCase) {
			return true
		}
	}
	return false
}

// isHwAgnostic checks if the test case has the hwAgnostic tag.
func isHwAgnostic(testCase *api.CTPTestCase) bool {
	for _, testCaseTag := range testCase.GetMetadata().GetTestCase().GetTags() {
		if testCaseTag.GetValue() == hwAgnostic {
			return true
		}
	}
	return false
}

func generateVMSchedulingUnit(board string, version string) *api.SchedulingUnit {
	return &api.SchedulingUnit{
		PrimaryTarget: &api.Target{
			SwarmingDef: &api.SwarmingDefinition{
				DutInfo: &testapi.Dut{
					DutType: &testapi.Dut_Chromeos{
						Chromeos: &testapi.Dut_ChromeOS{
							DutModel: &testapi.DutModel{BuildTarget: board},
						},
					},
				},
			}},
		DynamicUpdateLookupTable: map[string]string{
			"board":       board,
			"installPath": "gs://chromeos-image-archive/" + board + "-" + release + version,
		},
	}
}

// updateSchedulingUnitOption updates the list of Scheduling units. Returns nil if no VMlab Scheduling unit found.
func updateSchedulingUnitOption(schedulingUnitOption *api.SchedulingUnitOptions, board string, version string) *api.SchedulingUnitOptions {
	var filteredSchedulingUnits []*api.SchedulingUnit
	for _, schedulingUnit := range schedulingUnitOption.GetSchedulingUnits() {
		// if VM board in scheduling unit then do not remove. This will trigger VMlab flow
		if checkBoardInVMBoardsList(getBoard(schedulingUnit)) {
			filteredSchedulingUnits = append(filteredSchedulingUnits, schedulingUnit)
		}
	}
	if len(filteredSchedulingUnits) == 0 {
		return nil
	}
	schedulingUnitOption.SchedulingUnits = filteredSchedulingUnits
	return schedulingUnitOption
}

// updateSchedulingUnitOptions updates the schedulingUnitOptions for each test case in internal test plan request.
func updateSchedulingUnitOptions(req *api.InternalTestplan, board string, version string, log *log.Logger) error {

	for _, testCase := range req.GetTestCases() {
		if isTestVMCompatible(testCase) {
			var updatedSchedulingUnitOptions []*api.SchedulingUnitOptions
			for _, schedulingUnitOption := range testCase.GetSchedulingUnitOptions() {
				updatedSchedulingUnitOption := updateSchedulingUnitOption(schedulingUnitOption, board, version)
				if updatedSchedulingUnitOption == nil {
					continue
				}
				updatedSchedulingUnitOptions = append(updatedSchedulingUnitOptions, updatedSchedulingUnitOption)
			}
			// if no schedulign unit found from intended scheduling units, then create a VM schedulign unit with available board/version
			if len(updatedSchedulingUnitOptions) == 0 {
				testCase.SchedulingUnitOptions = []*api.SchedulingUnitOptions{
					{
						SchedulingUnits: []*api.SchedulingUnit{
							generateVMSchedulingUnit(board, version),
						},
					},
				}
			} else {
				testCase.SchedulingUnitOptions = updatedSchedulingUnitOptions
			}

		} else {
			log.Printf("Skippping as test : %s, isn't VM compatible", testCase.GetName())
		}
	}

	return nil
}

func executor(req *api.InternalTestplan, log *log.Logger) (*api.InternalTestplan, error) {

	available, board, version := isAnyVMImageAvailable(req, log)

	if available {
		// iterates each test case and updates the scheduling units based on if test is hw agnostic.
		err := updateSchedulingUnitOptions(req, board, version, log)
		if err != nil {
			return nil, fmt.Errorf("Error during AutoVM test shifter filter execution: %s", err)
		}
		log.Printf("Image is available: %t", available)
	} else {
		log.Printf("Skipping AutoVM test shifter filter execution. VM Image is not available.")
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
