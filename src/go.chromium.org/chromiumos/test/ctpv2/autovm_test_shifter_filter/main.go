// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"

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

var vmBoards = []string{"betty", "betty-arc-t", "reven-vmtest", "amd64-generic", "betty-kernelnext", "betty-arc-r", "betty-arc-v"}

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

func generateVMSchedulingUnitOptions(board string, version string) []*api.SchedulingUnitOptions {
	schedulingUnit := &api.SchedulingUnit{
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
	var schedulingUnits []*api.SchedulingUnit
	schedulingUnits = append(schedulingUnits, schedulingUnit)
	schedulingUnitOption := &api.SchedulingUnitOptions{
		SchedulingUnits: schedulingUnits,
	}

	var schedulingUnitOptions []*api.SchedulingUnitOptions
	schedulingUnitOptions = append(schedulingUnitOptions, schedulingUnitOption)

	return schedulingUnitOptions
}

// updateSchedulingUnitOptions updates the schedulingUnitOptions for each test case in internal test plan request.
func updateSchedulingUnitOptions(req *api.InternalTestplan, board string, version string, log *log.Logger) error {

	for _, testCase := range req.GetTestCases() {
		if isTestVMCompatible(testCase) {
			updatedSchedulingUnitOptions := generateVMSchedulingUnitOptions(board, version)
			testCase.SchedulingUnitOptions = updatedSchedulingUnitOptions
		} else {
			log.Printf("Skippping as test : %s, isn't VM compatible", testCase.GetName())
		}
	}

	return nil
}

func getBuildReportJSON(gcsPath string) ([]byte, error) {
	// Execute the gsutil ls command
	log.Printf("Checking image %s status...", gcsPath)
	cmd := exec.Command("gsutil", "cat", gcsPath+"/"+buildReportJSON)
	log.Printf("Executing command: %s", cmd.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error executing gsutil command: %v, output: %s", err, string(output))
	}
	// Check if the stdout is empty. No build report found
	if len(output) == 0 {
		return nil, fmt.Errorf("No build report found")
	}
	return output, nil
}

// isVMImageAvailable reads build report in gs bucket and infers if the builder has successfully completed.
func isVMImageAvailable(gcsPath string) (bool, error) {
	buildReportJSON, err := getBuildReportJSON(gcsPath)
	if err != nil {
		log.Printf("Error while getting build report: %s", err)
	}
	// Unmarshal the JSON string into the map
	var result map[string]interface{}
	err = json.Unmarshal(buildReportJSON, &result)
	if err != nil {
		return false, fmt.Errorf("error parsing JSON: %v", err)
	}

	// safely parse the nested structure
	status, ok := result["status"].(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("status key not found or is not in expected format")
	}

	value, ok := status["value"].(string)
	if !ok {
		return false, fmt.Errorf("value key not found or is not in expected format")
	}

	// check the status value from build report
	fmt.Printf("Status is %s\n", value)
	if value == success {
		log.Printf("Image %s is available", gcsPath)
		return true, nil
	}
	log.Printf("Image %s is not available", gcsPath)
	return false, nil
}

// getImageVersion returns the image version from a given image gs url(installPath) ex-`R128-15964.4.0`.
func getImageVersionFromInstallPath(installPath string) (string, error) {
	if installPath == "" {
		return "", errors.New("Empty Installpath")
	}
	// Define a regex pattern to match "R" followed by digits and a hyphen
	re := regexp.MustCompile(`/R(.+)$`)

	match := re.FindStringSubmatch(installPath)
	if len(match) < 2 {
		return "", errors.New("milestone version not found in installPath")
	}

	milestone := match[1]
	return "R" + milestone, nil
}

// getTargetedImageVersion fetches the chromeos image version of boards on which tests are to be executed.
// Note: Is same for all boards for a single CTP request.
func getTargetedImageVersion(req *api.InternalTestplan, log *log.Logger) string {
	// find the image version from the first valid gcs installPath
	for _, tc := range req.TestCases {
		for _, units := range tc.GetSchedulingUnitOptions() {
			for _, unit := range units.GetSchedulingUnits() {
				gcsInstallPath := unit.GetDynamicUpdateLookupTable()[installPath]
				imageVersion, err := getImageVersionFromInstallPath(gcsInstallPath)
				if err != nil {
					// log the error and try for other install paths.
					log.Printf("Error while getting image version for gcsInstallPath %s: %s", gcsInstallPath, err)
					continue
				}
				log.Printf("Found image version for the request: %s", imageVersion)
				return imageVersion
			}
		}
	}
	return ""
}

// isAnyVMImageAvailable returns true if any corresponding VM image for same version is available. Availability
// means that the builder has successfully completed building image and should be available to lease a VM with it.
func isAnyVMImageAvailable(req *api.InternalTestplan, log *log.Logger) (bool, string, string) {
	// fetch image version of intended boards. Same version of VM boards will be used, if available.
	targetedVersion := getTargetedImageVersion(req, log)
	if targetedVersion == "" {
		log.Printf("Failed to find image version from the request. Skipping AutoVM test shifter filter execution.")
		return false, "", ""
	}

	// check for all valid VM boards for image availability. Stop as soon as any is found available.
	log.Printf("Checking availability for VM board images ...")
	for _, board := range vmBoards {
		// form the gcsImagePath for VM board to check image availability.
		gcsImagePath := gsBucketPath + board + "-" + release + targetedVersion
		log.Printf("GCS image path for board %s: %s", board, gcsImagePath)

		exists, err := isVMImageAvailable(gcsImagePath)
		if err != nil {
			log.Printf("Error while checking image availability: %s", err)
			continue
		}
		if exists {
			return exists, board, targetedVersion
		}
	}
	log.Printf("No images found available.")
	return false, "", ""
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
