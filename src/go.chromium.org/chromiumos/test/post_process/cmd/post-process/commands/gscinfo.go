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
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	gscInfoFileName = "gsc_info.json"
)

// GscInfo packages GSC devboard info from GSC log files
func GscInfo(logger *log.Logger, testResult *artifact.TestResult) *api.GetGscInfoResponse {
	if testResult == nil {
		return &api.GetGscInfoResponse{}
	}

	ctx := context.Background()
	gscFiles := gscFiles(logger, testResult)
	gscInfos := gscInfos(ctx, logger, gscFiles)
	return &api.GetGscInfoResponse{GscInfos: gscInfos}
}

// gscFiles gets the GSC devboard info JSON file for each test result and
// returns a map from test name to GSC devboard info JSON file path.
func gscFiles(logger *log.Logger, testResult *artifact.TestResult) map[string]string {
	gscFiles := map[string]string{}
	for _, testRun := range testResult.GetTestRuns() {
		testCaseResult := testRun.GetTestCaseInfo().GetTestCaseResult()
		if testCaseResult == nil {
			continue
		}

		testName := testCaseResult.GetTestCaseId().GetValue()
		if testName == "" {
			continue
		}

		// Constructs the GSC devboard info JSON file path.
		resultDirPath := testCaseResult.GetResultDirPath().GetPath()
		gscFile := ""
		if strings.HasPrefix(testName, common.TastTestPrefix) {
			// For Tast test
			baseTestName, _ := strings.CutPrefix(testName, common.TastTestPrefix)
			gscFile = resultDirPath + "/" + common.TastTestsDir + "/" + baseTestName + "/" + gscInfoFileName
		} else if strings.HasPrefix(testName, common.TautoTestPrefix) {
			// For Tauto test
			baseTestName, _ := strings.CutPrefix(testName, common.TautoTestPrefix)
			gscFile = resultDirPath + "/" + baseTestName + "/" + gscInfoFileName
		}

		gscFiles[testName] = gscFile
	}

	logger.Printf("Successfully fetched GSC devboard info JSON files: %q", gscFiles)
	return gscFiles
}

// gscInfos ingests the GSC devboard info from JSON file.
func gscInfos(ctx context.Context, logger *log.Logger, gscFiles map[string]string) map[string]*anypb.Any {
	gscInfos := map[string]*anypb.Any{}
	for test, gscFile := range gscFiles {
		logger.Printf("Fetch: %q for test: %q\n", gscFile, test)

		// Converts the GSC devboard info JSON file into AVL info proto.
		gscInfo := &artifact.GscInfo{}
		err := util.ReadProtoJSONFile(ctx, gscFile, gscInfo)
		if err != nil {
			logger.Printf("Failed to read GSC devboard info from file: %q for test: %q with err: %v\n", gscFile, test, err)
			gscInfos[test], _ = anypb.New(&artifact.GscInfo{})
			continue
		}

		gscInfos[test], _ = anypb.New(gscInfo)
	}

	return gscInfos
}
