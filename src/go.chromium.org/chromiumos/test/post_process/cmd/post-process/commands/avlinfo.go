// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package commands

import (
	"context"
	"log"

	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/test/util/common"
)

// GetAvlInfo packages AVL qualification info from AVL log files
func GetAvlInfo(logger *log.Logger, req *api.GetAvlInfoRequest) (*api.GetAvlInfoResponse, error) {
	ctx := context.Background()
	avlInfos := map[string]*api.AvlInfo{}
	for test, file := range req.GetAvlFiles() {
		logger.Printf("Fetch: %q for test: %q\n", file, test)

		avlInfo := &api.AvlInfo{}
		err := common.ReadProtoJSONFile(ctx, file, avlInfo)
		if err != nil {
			logger.Printf("Failed to read AVL info from file: %q for test: %q with err: %v\n", file, test, err)
			avlInfos[test] = &api.AvlInfo{}
			continue
		}

		avlInfos[test] = avlInfo
	}
	return &api.GetAvlInfoResponse{AvlInfos: avlInfos}, nil
}
