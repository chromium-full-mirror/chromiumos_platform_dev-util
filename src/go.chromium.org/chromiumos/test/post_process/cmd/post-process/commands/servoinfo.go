// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package commands

import (
	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/config/go/test/artifact"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
)

// ServoInfo retrieves servo info from sysinfo/servo file.
func ServoInfo(logger *log.Logger, testResult *artifact.TestResult) (*api.GetServoInfoResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ServoInfo unimplemented")
}
