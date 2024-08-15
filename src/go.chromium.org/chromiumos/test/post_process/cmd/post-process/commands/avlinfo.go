// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package commands

import (
	"context"
	"log"
	"strings"

	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/config/go/test/artifact"
	"go.chromium.org/chromiumos/test/post_process/cmd/post-process/common"
	util "go.chromium.org/chromiumos/test/util/common"
)

const (
	avlInfoFileName = "avl_info.json"
)

// GetAvlInfo packages AVL qualification info from AVL log files
func GetAvlInfo(logger *log.Logger, testResult *artifact.TestResult) (*api.GetAvlInfoResponse, error) {
	if testResult == nil {
		return &api.GetAvlInfoResponse{}, nil
	}

	ctx := context.Background()
	avlFiles := avlFiles(logger, testResult)
	avlInfos := avlInfos(ctx, logger, avlFiles)
	return &api.GetAvlInfoResponse{AvlInfos: avlInfos}, nil
}

// avlFiles gets the AVL info JSON file for each test result and returns a
// map from test name to AVL info JSON file path.
func avlFiles(logger *log.Logger, testResult *artifact.TestResult) map[string]string {
	avlFiles := map[string]string{}
	for _, testRun := range testResult.GetTestRuns() {
		testCaseResult := testRun.GetTestCaseInfo().GetTestCaseResult()
		if testCaseResult == nil {
			continue
		}

		// Constructs the AVL info JSON file path.
		testName := testCaseResult.GetTestCaseId().GetValue()
		if testName == "" {
			continue
		}

		resultDirPath := testCaseResult.GetResultDirPath().GetPath()
		avlFile := ""
		if strings.HasPrefix(testName, common.TastTestPrefix) {
			// For Tast test
			baseTestName, _ := strings.CutPrefix(testName, common.TastTestPrefix)
			avlFile = resultDirPath + "/" + common.TastTestsDir + "/" + baseTestName + "/" + avlInfoFileName
		} else if strings.HasPrefix(testName, common.TautoTestPrefix) {
			// For Tauto test
			baseTestName, _ := strings.CutPrefix(testName, common.TautoTestPrefix)
			avlFile = resultDirPath + "/" + baseTestName + "/" + avlInfoFileName
		}

		avlFiles[testName] = avlFile
	}

	logger.Printf("Successfully fetched AVL info JSON files: %v", avlFiles)
	return avlFiles
}

// avlInfos ingests the AVL info from JSON file.
func avlInfos(ctx context.Context, logger *log.Logger, avlFiles map[string]string) map[string]*api.AvlInfo {
	avlInfos := map[string]*api.AvlInfo{}
	for test, avlFile := range avlFiles {
		logger.Printf("Fetch: %q for test: %q\n", avlFile, test)

		// Converts the AVL info JSON file into AVL info proto.
		avlInfo := &api.AvlInfo{}
		err := util.ReadProtoJSONFile(ctx, avlFile, avlInfo)
		if err != nil {
			logger.Printf("Failed to read AVL info from file: %q for test: %q with err: %v\n", avlFile, test, err)
			avlInfos[test] = &api.AvlInfo{}
			continue
		}

		avlInfos[test] = avlInfo
	}

	return avlInfos
}
