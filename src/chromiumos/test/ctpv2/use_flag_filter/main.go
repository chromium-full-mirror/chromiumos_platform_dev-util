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

	"github.com/golang/protobuf/jsonpb"
	"go.chromium.org/chromiumos/config/go/test/api"
)

const (
	binName      = "test_finder_filter"
	destFilePath = "/tmp/tast_use_flags"
	fileName     = "tast_use_flags.txt" // File name with .txt extension
)

// DownloadUseFlags downloads all use flag for a given gsUrl.
func DownloadUseFlags(ctx context.Context, gsURL string, gsClient *storage.GSClient, board string, log *log.Logger) (map[string]bool, error) {

	destFilePath := filepath.Join(destFilePath, board)

	if err := gsClient.DownloadFile(ctx, gsURL, destFilePath); err != nil {
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
	return useFlagDict, nil
}

// CreateBoardGsURLMapFromRequest creates a map with board value as key and gspath for image location as value
func createBoardGsURLMapFromRequest(req *api.InternalTestplan) (map[string]string, error) {

	boardGsMap := make(map[string]string)
	targetRequirements := req.GetSuiteInfo().GetSuiteMetadata().GetTargetRequirements()
	for _, tr := range targetRequirements {
		buildTarget, err := retrieveChromesOSBuildTarget(tr.GetHwRequirements().GetHwDefinition())
		if err != nil {
			return nil, fmt.Errorf("xyz: %s", err)
		}
		boardGsMap[buildTarget] = tr.GetSwRequirement().GetGcsPath()
	}
	return boardGsMap, nil
}

// RetrieveChromesOSBuildTarget retrieves ChromeOS build target from the list of HW Definition
func retrieveChromesOSBuildTarget(req []*api.SwarmingDefinition) (string, error) {
	for _, hwDef := range req {
		if hwDef.GetDutInfo().GetChromeos() != nil {
			return hwDef.GetDutInfo().GetChromeos().GetDutModel().GetBuildTarget(), nil
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

func filterHwRequirementsBasedOnUseFlag(hwReqs []*api.HWRequirements, useFlagDict map[string]map[string]bool, testHwDeps []*api.TestCase_BuildDeps) error {

	for _, hwReq := range hwReqs {
		var filteredHwDefs []*api.SwarmingDefinition
		hwDefs := hwReq.GetHwDefinition()
		for _, hwDef := range hwDefs {
			board := hwDef.GetDutInfo().GetChromeos().GetDutModel().GetBuildTarget()
			useFlagSet := useFlagDict[board]
			if hwDef.GetDutInfo().GetChromeos() != nil && useFlagDict[board] != nil {
				present := checkIfAnyUseFlagPresent(useFlagSet, testHwDeps)
				if present {
					filteredHwDefs = append(filteredHwDefs, hwDef)
				}
			} else {
				filteredHwDefs = append(filteredHwDefs, hwDef)
			}
		}
		hwReq.HwDefinition = filteredHwDefs
	}
	return nil
}

func updateTestCases(req *api.InternalTestplan, useFlagDict map[string]map[string]bool) error {

	for _, testCase := range req.GetTestCases() {
		if len(testCase.GetMetadata().GetTestCase().GetBuildDependencies()) > 0 {
			err := filterHwRequirementsBasedOnUseFlag(testCase.GetHwRequirements(), useFlagDict, testCase.GetMetadata().GetTestCase().GetBuildDependencies())
			if err != nil {
				return fmt.Errorf("Error filtering HwReqs: %s", err)
			}
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
	fmt.Println(useFlagDict)

	err = updateTestCases(req, useFlagDict)

	if err != nil {
		return nil, fmt.Errorf("Error during use flag based test cases pruning: %s", err)
	}

	marshaler := jsonpb.Marshaler{}
	jsonString, err := marshaler.MarshalToString(req)
	if err != nil {
		log.Fatalf("Error marshaling proto to JSON: %v", err)
	}

	// Print the JSON string
	fmt.Println(jsonString)

	return req, nil
}

func main() {
	err := server.Server(executor, binName)
	if err != nil {
		os.Exit(2)
	}

	os.Exit(0)
}
