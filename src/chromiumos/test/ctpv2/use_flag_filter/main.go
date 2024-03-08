// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	server "chromiumos/test/ctpv2/common/server_template"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"chromiumos/test/publish/cmd/publishserver/storage"

	"go.chromium.org/chromiumos/config/go/test/api"
)

const (
	binName      = "use_flag_filter"
	destFilePath = "/tmp/tast_use_flags"
	fileName     = "tast_use_flags.txt" // File name with .txt extension
)

// DownloadUseFlags downloads all use flag for a given gsUrl.
func DownloadUseFlags(ctx context.Context, gsURL string, gsClient *storage.GSClient, board string, log *log.Logger) (map[string]bool, error) {

	destFilePath := filepath.Join(destFilePath, board)

	if err := gsClient.DownloadFile(ctx, gsURL, destFilePath); err != nil {
		log.Printf("Error while reading use flag from gs: %s", err)
		return nil, err
	}

	content, err := os.ReadFile(destFilePath)
	if err != nil {
		return nil, err
	}

	useFlagList := strings.Split(string(content), "\n")

	useFlagSet := make(map[string]bool)
	for _, item := range useFlagList {
		useFlagSet[item] = true
	}

	return useFlagSet, nil
}

func createDirectory(destFilePath string) error {

	if _, err := os.Stat(destFilePath); err == nil {
		if err := os.RemoveAll(destFilePath); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(destFilePath, 0755); err != nil {
		return err
	}
	return nil
}

// GenerateUseFlagDict creates a dictionary for boards and respective use flag list.
func generateUseFlagDict(ctx context.Context, req *api.InternalTestplan, log *log.Logger) (map[string]map[string]bool, error) {

	boardGsMap, err := createBoardGsURLMapFromRequest(req)
	if err != nil {
		return nil, fmt.Errorf("Error generating board-gcsPath map: %s", err)
	}

	gsClient, err := storage.NewGSClient(ctx, "")
	if err != nil {
		return nil, err
	}

	err = createDirectory(destFilePath)
	if err != nil {
		return nil, fmt.Errorf("Error setting up parent directory for downloading files from gs: %s", err)
	}

	// useFlagDict will look like after collecting use flags for all board-variant combination in suite info.
	// map[
	// 	"nissa-variant": map[
	// 		"amd64" : true,
	// 		"vulkan" : true,
	// 		"wpa3_sae" : true
	// 	],
	// 	"octopus-variant": map[
	// 		"arc" : true,
	// 		"arc-camera3" : true,
	// 		"arcvm" : true,
	// 		"cheets_user_64" : true
	// 	]
	// ]
	useFlagDict := make(map[string]map[string]bool)

	for board, gcsPath := range boardGsMap {
		gcsPath = gcsPath + "/" + fileName
		useFlagSet, err := DownloadUseFlags(ctx, gcsPath, gsClient, board, log)
		if err != nil {
			return nil, err
		}

		useFlagDict[board] = useFlagSet
	}

	defer gsClient.Close()
	fmt.Println(useFlagDict)
	log.Printf("useFlagDict: %s", boardGsMap)
	return useFlagDict, nil
}

// CreateBoardGsURLMapFromRequest creates a map with board value as key and gspath for image location as value
func createBoardGsURLMapFromRequest(req *api.InternalTestplan) (map[string]string, error) {

	boardGsMap := make(map[string]string)
	targetRequirements := req.GetSuiteInfo().GetSuiteMetadata().GetTargetRequirements()
	for _, tr := range targetRequirements {
		buildTarget, err := retrieveChromesOSHwUseFlagKey(tr.GetHwRequirements().GetHwDefinition())
		if err != nil {
			return nil, fmt.Errorf("xyz: %s", err)
		}
		boardGsMap[buildTarget] = tr.GetSwRequirement().GetGcsPath()
	}
	log.Printf("BoardGsURLMap: %s", boardGsMap)
	return boardGsMap, nil
}

// RetrieveChromesOSHwUseFlagKey retrieves ChromeOS buildTarget+variant from the list of HW Definition
func retrieveChromesOSHwUseFlagKey(req []*api.SwarmingDefinition) (string, error) {
	for _, hwDef := range req {
		if hwDef.GetDutInfo().GetChromeos() != nil {
			return hwDef.GetDutInfo().GetChromeos().GetDutModel().GetBuildTarget() + "-" + hwDef.GetVariant(), nil
		}
	}
	return "", nil
}

func checkIfAnyUseFlagPresent(useFlagSet map[string]bool, testHwDeps []*api.TestCase_BuildDeps) bool {

	for _, buildDep := range testHwDeps {
		if useFlagSet[buildDep.GetValue()] {
			return true
		}
	}
	return false
}

func filterHwRequirementsBasedOnUseFlag(hwReqs []*api.HWRequirements, useFlagDict map[string]map[string]bool, testHwDeps []*api.TestCase_BuildDeps, log *log.Logger) error {

	for _, hwReq := range hwReqs {
		// hwDef will be removed if no buildDeps in test metadata is present in hwDef's use flag set in useFlagDict
		var filteredHwDefs []*api.SwarmingDefinition
		hwDefs := hwReq.GetHwDefinition()

		for _, hwDef := range hwDefs {

			// fetch use flag set for hwDef. Key is BuildTarget+variant.
			board := hwDef.GetDutInfo().GetChromeos().GetDutModel().GetBuildTarget()
			variant := hwDef.GetVariant()
			useFlagSet := useFlagDict[board+"-"+variant]

			if hwDef.GetDutInfo().GetChromeos() != nil && useFlagDict[board+"-"+variant] != nil {
				// check if any test buildDeps is present in the set.
				present := checkIfAnyUseFlagPresent(useFlagSet, testHwDeps)
				if present {
					filteredHwDefs = append(filteredHwDefs, hwDef)
				} else {
					log.Printf("DUT with build target : %s removed", board)
				}
			} else {
				log.Printf("DUT with build target : %s skipped check, due to missing use flag list or DUT is not of ChromeOS type", board)
				filteredHwDefs = append(filteredHwDefs, hwDef)
			}
		}
		// update hwDef list
		hwReq.HwDefinition = filteredHwDefs
	}
	return nil
}

func updateTestCases(req *api.InternalTestplan, useFlagDict map[string]map[string]bool, log *log.Logger) error {

	for _, testCase := range req.GetTestCases() {
		if len(testCase.GetMetadata().GetTestCase().GetBuildDependencies()) > 0 {
			log.Printf("Pruning HwDef for test : %s", testCase.GetName())
			err := filterHwRequirementsBasedOnUseFlag(testCase.GetHwRequirements(), useFlagDict, testCase.GetMetadata().GetTestCase().GetBuildDependencies(), log)
			if err != nil {
				return fmt.Errorf("Error filtering HwReqs: %s", err)
			}
		} else {
			log.Printf("Skippping as test : %s, doesn't  have any build deps", testCase.GetName())
		}
	}
	return nil
}

func executor(req *api.InternalTestplan, log *log.Logger) (*api.InternalTestplan, error) {
	ctx := context.Background()

	useFlagDict, err := generateUseFlagDict(ctx, req, log)
	if err != nil {
		return nil, fmt.Errorf("Error generating board-use flags map: %s", err)
	}

	err = updateTestCases(req, useFlagDict, log)
	if err != nil {
		return nil, fmt.Errorf("Error during use flag based test cases pruning: %s", err)
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
