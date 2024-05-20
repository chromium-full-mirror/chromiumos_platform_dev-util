// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package commands

import (
	"log"

	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetAvlInfo packages AVL qualification info from AVL log files
func GetAvlInfo(logger *log.Logger, dutClient api.DutServiceClient) (*api.GetAvlInfoResponse, error) {
	return nil, status.Error(codes.Unimplemented, "GetAvlInfo unimplemented")
}
