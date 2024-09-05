// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package driver implements drivers to execute tests.
package driver

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"path"
	"strings"
	"sync"

	c "go.chromium.org/chromiumos/test/util/adb"

	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/test/execution/cmd/cros-test/internal/common"
	"go.chromium.org/chromiumos/test/execution/cmd/cros-test/internal/device"
)

const testPath = "/usr/local/tradefed/android-%s"
const tradefedTestResultFile = "results/latest/test_result.xml"
const ctsResultsDir = "results/"
const ctsLogsDir = "logs/"

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

func buildTradefedCommand(logger *log.Logger, tests []*api.TestCaseMetadata, primaryIp string) []string {
	baseArgs := []string{"run", "commandAndExit", testType}
	for _, t := range tests {
		logger.Println("Running provided test: ", t.TestCase.Name)
		baseArgs = append(baseArgs, "--include-filter", t.TestCase.Name)
	}
	baseArgs = append(baseArgs, "--logcat-on-failure")
	baseArgs = append(baseArgs, "-s", c.FmtAddr(primaryIp))
	return baseArgs
}

func runTradefedTest(ctx context.Context, logger *log.Logger, tests []*api.TestCaseMetadata, primary *device.DutInfo, resultsPath string) error {
	err := c.SetupAdb(logger, primary.Ip)
	if err != nil {
		return fmt.Errorf("setupAdb failed")
	}

	exit := make(chan struct{})
	defer func() {
		exit <- struct{}{}
		err = c.TeardownAdb(logger, primary.Ip)
	}()

	baseArgs := buildTradefedCommand(logger, tests, primary.Ip)
	terminalPath := path.Join(fmt.Sprintf(testPath, testType), fmt.Sprintf("tools/%s-tradefed", testType))
	cmd := exec.Command(terminalPath, baseArgs...)
	logger.Println("Running TF: ", cmd.String())

	c.KeepAdbAlive(logger, primary.Ip, exit)
	return launchAndRead(cmd, logger)
}

// RunTests drives a test framework to execute tests.
func (td *TradefedDriver) RunTests(ctx context.Context, resultsDir string, req *api.CrosTestRequest, tlwAddr string, tests []*api.TestCaseMetadata) (*api.CrosTestResponse, error) {
	allRspn := &api.CrosTestResponse{}

	primary, err := device.FillDUTInfo(req.Primary, "")
	if err != nil {
		return nil, err
	}

	for _, test := range tests {
		if strings.HasPrefix(test.GetTestCase().GetName(), "Dts") {
			testType = "dts"
			break
		}
	}

	_ = runTradefedTest(ctx, td.logger, tests, primary, resultsDir)
	// TODO determine what we want to do with an error _and_ possibly having results
	// In general I'd say just spit out the results.
	// One difference between Tast/Autotest and this is that the test case given will likely result in many
	// testcases; where T/AT were normally 1:1
	resultsPath := path.Join(fmt.Sprintf(testPath, testType), tradefedTestResultFile)
	results := buildTradefedResult(td.logger, resultsPath)
	if results.GetTestCaseResults() != nil {
		allRspn.TestCaseResults = append(allRspn.TestCaseResults, results.TestCaseResults...)
	} else {

		td.logger.Println("nill resulsts?")
	}

	td.moveArtifacts(resultsDir, ctsResultsDir)
	td.moveArtifacts(resultsDir, ctsLogsDir)
	td.logger.Println("final results: ", allRspn)
	td.logger.Println("final results: ", results)

	return allRspn, nil
}

// Generate unit tests for the function "moveArtifacts" below
func (td *TradefedDriver) moveArtifacts(resultsDir string, artifactsDir string) {
	mvCmd := exec.Command("mv", path.Join(fmt.Sprintf(testPath, testType), artifactsDir), resultsDir)
	td.logger.Printf("started to move artifacts from: %q to: %q", mvCmd.Path, resultsDir)

	out, err := mvCmd.CombinedOutput()
	if err != nil {
		td.logger.Printf("Error moving artifacts from: %q: %s: %s", mvCmd.Path, err, string(out))
		return
	}
	td.logger.Printf("successfully moved artifacts from: %q to: %q", mvCmd.Path, resultsDir)
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
