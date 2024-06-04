// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"log"
	"os"

	server "go.chromium.org/chromiumos/test/ctpv2/common/server_template"

	"go.chromium.org/chromiumos/config/go/test/api"
)

const (
	stubPass = "tauto.stub_PassServer"
)

func executor(req *api.InternalTestplan, log *log.Logger) (*api.InternalTestplan, error) {
	filteredTests := make([]*api.CTPTestCase, 0)
	for _, testCase := range req.GetTestCases() {
		testID := testCase.GetMetadata().GetTestCase().GetId().GetValue()
		if testID != stubPass {
			filteredTests = append(filteredTests, testCase)
		}
	}

	req.TestCases = filteredTests
	return req, nil
}

func main() {
	err := server.Server(executor, "")
	if err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}
