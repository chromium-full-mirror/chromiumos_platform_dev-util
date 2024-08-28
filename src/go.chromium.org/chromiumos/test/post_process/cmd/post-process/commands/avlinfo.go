// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package commands

import (
	"context"
	"log"

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
	avlFiles := common.TestLevelFiles(logger, testResult, avlInfoFileName)
	avlInfos := avlInfos(ctx, logger, avlFiles)
	return &api.GetAvlInfoResponse{AvlInfos: avlInfos}, nil
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
