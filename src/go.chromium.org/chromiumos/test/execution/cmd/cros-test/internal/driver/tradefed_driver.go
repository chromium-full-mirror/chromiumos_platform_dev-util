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
	"sync"

	c "go.chromium.org/chromiumos/test/util/adb"

	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/test/execution/cmd/cros-test/internal/common"
	"go.chromium.org/chromiumos/test/execution/cmd/cros-test/internal/device"
)

const ctsPath = "/usr/local/tradefed/android-cts"
const tradefedTestResultFile = "results/latest/test_result.xml"

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
	baseArgs := []string{"run", "commandAndExit", "cts"}
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

	defer func() {
		err = c.TeardownAdb(logger, primary.Ip)
	}()

	baseArgs := buildTradefedCommand(logger, tests, primary.Ip)
	terminalPath := path.Join(ctsPath, "tools/cts-tradefed")
	cmd := exec.Command(terminalPath, baseArgs...)
	logger.Println("Running TF: ", cmd.String())
	return launchAndRead(cmd, logger)
}

// RunTests drives a test framework to execute tests.
func (td *TradefedDriver) RunTests(ctx context.Context, resultsDir string, req *api.CrosTestRequest, tlwAddr string, tests []*api.TestCaseMetadata) (*api.CrosTestResponse, error) {
	allRspn := &api.CrosTestResponse{}

	primary, err := device.FillDUTInfo(req.Primary, "")
	if err != nil {
		return nil, err
	}

	_ = runTradefedTest(ctx, td.logger, tests, primary, resultsDir)
	// TODO determine what we want to do with an error _and_ possibly having results
	// In general I'd say just spit out the results.
	// One difference between Tast/Autotest and this is that the test case given will likely result in many
	// testcases; where T/AT were normally 1:1
	resultsPath := path.Join(ctsPath, tradefedTestResultFile)
	results := buildTradefedResult(td.logger, resultsPath)
	if results.GetTestCaseResults() != nil {
		allRspn.TestCaseResults = append(allRspn.TestCaseResults, results.TestCaseResults...)
	} else {

		td.logger.Println("nill resulsts?")
	}

	td.logger.Println("Got results, moving resultsidr")
	td.moveResults(resultsDir)
	td.logger.Println("final results: ", allRspn)
	td.logger.Println("final results: ", results)

	return allRspn, nil
}

func (td *TradefedDriver) moveResults(resultsDir string) error {

	mvCmd := exec.Command("mv", path.Join(ctsPath, "results/"), resultsDir)
	out, err := mvCmd.CombinedOutput()
	if err != nil {
		td.logger.Printf("Error moving results: %s: %s", err, string(out))
	}
	td.logger.Println("moved results")
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
