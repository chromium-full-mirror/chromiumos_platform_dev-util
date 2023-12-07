// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package driver implements drivers to execute tests.
package driver

import (
	"bufio"
	"chromiumos/test/execution/cmd/cros-test/internal/common"
	"chromiumos/test/execution/errors"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/encoding/prototext"

	"go.chromium.org/chromiumos/config/go/test/api"
)

const omnilabGatewayOutputMaxCapacity = 4096 * 1024 * 10 // 40MB
const moblyCmd = "gateway_go"

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
	return "mobly"
}

// Execute Mobly command against Omnilab Gateway Client and process test results
func runMoblyCmd(ctx context.Context, logger *log.Logger, test *api.TestCaseMetadata) (*api.TestCaseResult, error) {
	startTime := time.Now()

	moblyArgs, err := genMoblyArgs(test)
	if err != nil {
		return buildTestCaseResult(test,
			startTime,
			time.Since(startTime),
			"FAIL",
			err.Error()), nil
	}

	// Run mobly command.
	cmd := exec.Command(moblyCmd, moblyArgs...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return buildTestCaseResult(test,
			startTime,
			time.Since(startTime),
			"FAIL",
			fmt.Errorf("failed to capture mobly stderr: %v", err).Error()), nil
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return buildTestCaseResult(test,
			startTime,
			time.Since(startTime),
			"FAIL",
			fmt.Errorf("failed to capture mobly stdout: %v", err).Error()), nil
	}
	logger.Println("Running Mobly: ", cmd.String())
	if err := cmd.Start(); err != nil {
		return buildTestCaseResult(test,
			startTime,
			time.Since(startTime),
			"FAIL",
			fmt.Errorf("failed to run mobly: %v", err).Error()), nil
	}
	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine to handle stderr logging
	go func() {
		defer wg.Done()
		common.TestScanner(stderr, logger, "mobly")
	}()

	type result struct {
		sessionDetailResponse *sessionDetailResponse
		err                   error
	}
	stdoutResultChannel := make(chan result)

	// Goroutine to process GetSessionDetailResponse from Omnilab Gateway Client.
	go func() {
		defer wg.Done()

		scanner := bufio.NewScanner(stdout)
		buf := make([]byte, omnilabGatewayOutputMaxCapacity)
		scanner.Buffer(buf, omnilabGatewayOutputMaxCapacity)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			respJsonText := scanner.Text()
			// Process response from Omnilab Gateway Client
			// Each line of output in stdout is a json marshalled representation of
			// Omnilab Gateway gRPC proto definition.
			var response sessionDetailResponse
			err := json.Unmarshal([]byte(respJsonText), &response)
			if err != nil {
				logger.Printf("[%v] failed to deserialize Omnilab Gateway Client output (%v): %v",
					"mobly", respJsonText, err)
				stdoutResultChannel <- result{sessionDetailResponse: nil, err: err}
				break
			}
			//Check sessionDetailResponse status and exit conditions
			if response.SessionDetail.SessionSummary.Status == "DONE" {
				logger.Printf("[%v] Omnilab Gateway Client output: %v", "mobly", respJsonText)
				stdoutResultChannel <- result{sessionDetailResponse: &response, err: nil}
				break
			}

		}
		if scanner.Err() != nil {
			logger.Printf("[%v] failed to read stdout pipe: %v", "mobly", scanner.Err())
			stdoutResultChannel <- result{sessionDetailResponse: nil, err: scanner.Err()}
		}
	}()

	select {
	case <-ctx.Done():
		return buildTestCaseResult(test,
			startTime,
			time.Since(startTime),
			"FAIL",
			ctx.Err().Error()), nil

	case res := <-stdoutResultChannel:
		// wait for the stderr goroutine to finish.
		// The unlock statement is moved here to ensure there is no deadlock.
		wg.Wait()
		if res.err != nil {
			return buildTestCaseResult(test,
				startTime,
				time.Since(startTime),
				"FAIL",
				res.err.Error()), nil
		} else if res.sessionDetailResponse != nil {
			tcResult := buildTestCaseResultFromSessionResponse(test,
				startTime,
				time.Since(startTime),
				res.sessionDetailResponse)
			logger.Printf("[%v] Test case result: %v", "mobly", prototext.Format(tcResult))
			return tcResult, nil
		}
		return buildTestCaseResult(test,
			startTime,
			time.Since(startTime),
			"FAIL",
			"Unknown error"), nil
	}
}

// Generate Mobly command line arguments for Omnilab Gateway Client
func genMoblyArgs(test *api.TestCaseMetadata) ([]string, error) {
	argList := make([]string, 0)
	tagsMap := getTestCaseMetadataTags(test)

	arr, ok := tagsMap["mobly_mh_target"]
	if ok && len(arr) > 0 {
		target := arr[0]
		argList = append(argList, "-target_name")
		argList = append(argList, target)
		argList = append(argList, "-test_flags")
		//TODO (jonfan): The execution is hardcoded against a fixed omnilab setup.
		//Remove the hardcoded once the mechanism is in place to inject scheduling dimensions.
		argList = append(argList, "--notest_loasd --test_arg=--run_as=chromeos-engprod --test_arg=--dimension_label=chromeos-engprod-demo")
		return argList, nil
	} else {
		return nil, errors.NewStatusError(errors.InvalidArgument, fmt.Errorf("%v", "could not find mobly_mh_target"))
	}
}

// RunTests drives a test framework to execute tests.
func (td *MoblyTestDriver) RunTests(ctx context.Context, resultsDir string, req *api.CrosTestRequest, tlwAddr string, tests []*api.TestCaseMetadata) (*api.CrosTestResponse, error) {
	var testCaseResults []*api.TestCaseResult

	//TODO (jonfan): Merge all tests into one session for performance consideration.
	for _, test := range tests {
		tcResult, err := runMoblyCmd(ctx, td.logger, test)
		if err != nil {
			// Ideally we should not be here as all the errors should have been wrapped
			// as a failed test case inside of runMoblyCmd.
			return nil, errors.NewStatusError(errors.InvalidArgument,
				fmt.Errorf("fail to run mobly command for test %v", test.TestCase.Name))
		}
		testCaseResults = append(testCaseResults, tcResult)
	}
	return &api.CrosTestResponse{TestCaseResults: testCaseResults}, nil
}

// Collect tag names and tag values and convert to map of string arrays
func getTestCaseMetadataTags(test *api.TestCaseMetadata) map[string][]string {
	tagsMap := make(map[string][]string)
	for _, tag := range test.TestCase.GetTags() {
		// Splitting on the first semi colon
		arr := strings.SplitN(tag.Value, ":", 2)
		if len(arr) != 2 {
			continue
		}
		tagsMap[arr[0]] = append(tagsMap[arr[0]], arr[1])
	}
	return tagsMap
}
