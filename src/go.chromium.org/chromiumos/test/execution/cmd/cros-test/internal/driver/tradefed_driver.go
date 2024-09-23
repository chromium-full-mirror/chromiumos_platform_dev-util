// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package driver implements drivers to execute tests.
package driver

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	c "go.chromium.org/chromiumos/test/util/adb"

	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/test/execution/cmd/cros-test/internal/common"
	"go.chromium.org/chromiumos/test/execution/cmd/cros-test/internal/device"
)

const (
	tradefedTestResultFile = "results/latest/test_result.xml"
	ctsResultsDir          = "results/"
	ctsLogsDir             = "logs/"

	tradefedConsoleBinary = "/tradefed/tradefed_runner.sh"
	tradefedGlobalLogs    = "tradefed_global_log_*.txt"
)

var testType = "cts"

type TradefedDriver struct {
	logger *log.Logger
}

func NewTradefedDriver(logger *log.Logger) *TradefedDriver {
	return &TradefedDriver{
		logger: logger,
	}
}
func (d *TradefedDriver) Name() string {
	return "tradefed"
}

func runTradefedTest(ctx context.Context, logger *log.Logger, tests []*api.TestCaseMetadata,
	serials []string, resultsPath string, metadata *api.ExecutionMetadata) error {

	for _, s := range serials {
		if err := c.SetupAdb(logger, s); err != nil {
			return fmt.Errorf("setupAdb failed for %s", s)
		}
	}

	exit := make(chan struct{})
	defer func() {
		exit <- struct{}{}
		for _, s := range serials {
			if err := c.TeardownAdb(logger, s); err != nil {
				logger.Printf("Failed to tear down adb connection to %s: %s", s, err)
			}
		}
	}()

	var cmd *exec.Cmd

	if testType == "cts" || testType == "dts" {
		// Using Google TradeFed console for CTS and DTS tests.
		baseArgs := []string{"run", "commandAndExit"}
		baseArgs = append(baseArgs, BuildXtsTestCommand(logger, testType, tests, serials, metadata)...)
		cmd = exec.Command(tradefedConsoleBinary, baseArgs...)
	}

	logger.Println("Running TF: ", cmd.String())

	c.KeepAdbAlive(logger, serials, exit)
	return launchAndRead(cmd, logger)
}

// RunTests drives a test framework to execute tests.
func (td *TradefedDriver) RunTests(ctx context.Context, resultsDir string, req *api.CrosTestRequest, tlwAddr string, tests []*api.TestCaseMetadata) (*api.CrosTestResponse, error) {
	allRspn := &api.CrosTestResponse{}

	serials, err := device.DerviceSerials(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call DerviceSerials: %s", err)
	}

	dtsFound := false
	for _, test := range tests {
		if strings.HasPrefix(test.GetTestCase().GetName(), "dts.") {
			dtsFound = true
		}
		for _, tag := range test.GetTestCase().GetTags() {
			dtsFound = dtsFound || tag.GetValue() == "suite:dts"
		}
		if dtsFound {
			testType = "dts"
			break
		}
	}

	executionMD := &api.ExecutionMetadata{}
	if len(req.GetTestSuites()) > 0 {
		executionMD = req.GetTestSuites()[0].GetExecutionMetadata()
	}

	_ = runTradefedTest(ctx, td.logger, tests, serials, resultsDir, executionMD)
	// TODO determine what we want to do with an error _and_ possibly having results
	// In general I'd say just spit out the results.
	// One difference between Tast/Autotest and this is that the test case given will likely result in many
	// testcases; where T/AT were normally 1:1

	results, artifacts := buildTradefedResult(td.logger, testType)
	if results.GetTestCaseResults() != nil {
		allRspn.TestCaseResults = append(allRspn.TestCaseResults, results.TestCaseResults...)
	} else {
		td.logger.Println("No results to report: ", results)
	}

	// Adding all global TradeFed logs to the list of artifacts.
	artifacts = append(artifacts, filepath.Join(os.Getenv("GLOBAL_LOG_PATH"), tradefedGlobalLogs))

	td.logger.Println("Collecting result artifacts to:", resultsDir)
	for _, artifact := range artifacts {
		if len(artifact) > 0 {
			td.moveArtifacts(resultsDir, artifact)
		}
	}

	return allRspn, nil
}

// Move artifact files to resultsDir, deleting the source files. Supports glob patterns.
// Doesn't remove source directory, only files/dirs inside.
func (td *TradefedDriver) moveArtifacts(resultsDir string, artifacts string) {
	matches, err := filepath.Glob(artifacts)
	if err != nil {
		td.logger.Printf("Failed to match %q: %v", artifacts, err)
		return
	}

	for _, match := range matches {
		td.logger.Printf("Moving result artifact: %q to: %q", match, resultsDir)
		if err := moveSingleArtifact(resultsDir, match); err != nil {
			td.logger.Printf("Failed to move %q to: %q, error: %v", match, resultsDir, err)
		}
	}
}

func moveSingleArtifact(resultsDir string, artifact string) error {
	mvCmd := exec.Command("mv", artifact, resultsDir)
	out, err := mvCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error moving file: %s, output: %s", err, string(out))
	}
	return nil
}

func launchAndRead(cmd *exec.Cmd, logger *log.Logger) error {
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("StderrPipe failed")
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("StdoutPipe failed")
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to run Tauto: %v", err)
	}
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		common.TestScanner(stderr, logger, "Tradefed")
	}()

	go func() {
		defer wg.Done()
		common.TestScanner(stdout, logger, "Tradefed")
	}()

	wg.Wait()
	return nil
}
