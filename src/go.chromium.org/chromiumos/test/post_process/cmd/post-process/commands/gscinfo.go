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

// GscInfo packages GSC devboard info from GSC log files
func GscInfo(logger *log.Logger, req *api.GetGscInfoRequest) (*api.GetGscInfoResponse, error) {
	return nil, status.Error(codes.Unimplemented, "GetGscInfo unimplemented")
}
