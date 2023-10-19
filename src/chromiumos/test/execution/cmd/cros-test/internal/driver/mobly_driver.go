// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package driver implements drivers to execute tests.
package driver

import (
	"context"
	"log"
	"time"

	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MoblyTestDriver runs mobly and report its results.
type MoblyTestDriver struct {
	// logger provides logging service.
	logger *log.Logger
}

// NewMoblyTestDriver creates a new driver to run tests.
func NewMoblyDriver(logger *log.Logger) *MoblyTestDriver {
	return &MoblyTestDriver{
		logger: logger,
	}
}

// Name returns the name of the driver.
func (td *MoblyTestDriver) Name() string {
	return "gtest"
}

type moblyTestCase struct {
	Name      string
	Status    string
	Result    string
	Timestamp string
	Time      string
}

// // logCmd logs a remote command run through a DUT server
// func logCmd(logger *log.Logger, cmd *api.ExecCommandRequest, resp *api.ExecCommandResponse) {
// 	logger.Printf("cmd '%v', args '%v'", cmd.Command, cmd.Args)
// 	logger.Printf("[status]:\n%v", resp.ExitInfo.Status)
// 	logger.Printf("[stdout]:\n%v", string(resp.Stdout))
// 	logger.Printf("[stderr]:\n%v", string(resp.Stderr))
// 	logger.Printf("[error]:\n%v", string(resp.ExitInfo.ErrorMessage))
// }

func runMoblyCmd(ctx context.Context, logger *log.Logger, test *api.TestCaseMetadata) string {
	// CALL THE MOBLY ENDPOINT HERE :D
	return "pass"
}

// RunTests drives a test framework to execute tests.
func (td *MoblyTestDriver) RunTests(ctx context.Context, resultsDir string, req *api.CrosTestRequest, tlwAddr string, tests []*api.TestCaseMetadata) (*api.CrosTestResponse, error) {
	var testCaseResults []*api.TestCaseResult

	// Setup dut connection to be able to run the tests and get results.
	// Not currently needed for remote remote Mobly, but might need to bring this back for skylab mobly.
	// var dutInfo *device.DutInfo
	// if dutInfo, err = device.FillDUTInfo(req.Primary, ""); err != nil {
	// 	return nil, errors.NewStatusError(errors.InvalidArgument,
	// 		fmt.Errorf("cannot get address from primary device: %v", dutInfo))
	// }

	// var primaryDutConn *grpc.ClientConn
	// if primaryDutConn, err = grpc.Dial(dutInfo.DutServer, grpc.WithInsecure()); err != nil {
	// 	return nil, errors.NewStatusError(errors.InvalidArgument,
	// 		fmt.Errorf("cannot create connection with primary device: %v, address: %v", req.Primary, dutInfo.DutServer))
	// }
	// defer primaryDutConn.Close()

	// dut := api.NewDutServiceClient(primaryDutConn)

	for _, test := range tests {
		results := runMoblyCmd(ctx, td.logger, test)

		tcResult := buildMoblyResult(test.TestCase.Id.Value, results)
		testCaseResults = append(testCaseResults, tcResult)
	}
	return &api.CrosTestResponse{TestCaseResults: testCaseResults}, nil
}

// If err is populated, and ERROR status will be returned.
func buildMoblyResult(tcID string, results string) *api.TestCaseResult {
	tcResult := new(api.TestCaseResult)

	tcResult.TestHarness = &api.TestHarness{TestHarnessType: &api.TestHarness_Gtest_{Gtest: &api.TestHarness_Gtest{}}}
	tcResult.TestCaseId = &api.TestCase_Id{Value: tcID}
	tcResult.StartTime = timestamppb.New(time.Now())
	tcResult.Duration = &durationpb.Duration{Seconds: 100}

	tcResult.Verdict = &api.TestCaseResult_Pass_{Pass: &api.TestCaseResult_Pass{}}

	return tcResult
}
