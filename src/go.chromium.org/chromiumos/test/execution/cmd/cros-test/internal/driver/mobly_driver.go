// Copyright 2023 The ChromiumOS Authors
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
	"path"
	"runtime/debug"
	"sync"

	c "go.chromium.org/chromiumos/test/util/adb"

	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/test/execution/cmd/cros-test/internal/common"
	"go.chromium.org/chromiumos/test/execution/cmd/cros-test/internal/device"
)

// MoblyDriver runs mobly and report its results.
type MoblyDriver struct {
	// logger provides logging service.
	logger *log.Logger
}

// NewMoblyTestDriver creates a new driver to run tests.
func NewMoblyDriver(logger *log.Logger) *MoblyDriver {
	return &MoblyDriver{
		logger: logger,
	}
}

// Name returns the name of the driver.
func (md *MoblyDriver) Name() string {
	return "mobly"
}

func buildMoblyCommand(logger *log.Logger, test *api.TestCaseMetadata, resultsPath string) *exec.Cmd {
	parFile := test.GetTestCaseInfo().GetExtraInfo()
	par, ok := parFile["executable_name"]
	if !ok {
		return nil
	}
	par = fmt.Sprintf("/usr/local/mobly/%s", par)
	logger.Println("Building mobly command")
	baseArgs := []string{
		"--",
		"-c", "/usr/local/mobly/test_config.yml",
		// "-lp", resultsPath,
	}

	baseArgs = append(baseArgs, "--tests")
	baseArgs = append(baseArgs, test.GetTestCase().GetName())

	cmd := exec.Command(par, baseArgs...)

	return cmd
}

// Execute Mobly command against Omnilab Gateway Client and process test results
func runMoblyTest(ctx context.Context, logger *log.Logger, test *api.TestCaseMetadata, serials []string, resultsPath string, metadata []*api.Arg) error {
	if err := GenerateMoblyConfig(logger, "/usr/local/mobly", serials, metadata); err != nil {
		return fmt.Errorf("failed to generate mobly config, %w", err)
	}

	cmd := buildMoblyCommand(logger, test, resultsPath)
	if cmd == nil {
		return fmt.Errorf("failed to build mobly command")
	}
	logger.Println("Running Mobly: ", cmd.String())

	return launchAndReadMobly(cmd, logger)
}

func buildMoblyResult(logger *log.Logger, resultsPath, resultsDir string, f *api.CrosTestResponse) *api.CrosTestResponse {
	defer func() {
		if r := recover(); r != nil {
			logger.Println("found a panic")
			logger.Printf("Error: %+v", fmt.Errorf("%v", r))
			logger.Println(string(debug.Stack()))
		}
	}()

	logger.Println("reading file")
	moblyResultBytes, err := os.ReadFile(resultsPath)
	logger.Println("reading file done")
	if err != nil {
		logger.Printf("failed to read %s, %s", resultsPath, err)
		return nil
	}

	logger.Println("translating", moblyResultBytes)
	testCaseResults, err := common.TranslateMoblyResults(moblyResultBytes, resultsDir)
	logger.Println("translating done", testCaseResults)
	if err != nil {
		logger.Printf("failed to translate mobly results, %s", err)
		return nil
	}

	f.TestCaseResults = append(f.TestCaseResults, testCaseResults...)

	logger.Println("Got results:")
	for _, result := range f.TestCaseResults {
		logger.Println(result.String())
	}

	return f
}

// RunTests drives a test framework to execute tests.
func (md *MoblyDriver) RunTests(ctx context.Context, resultsDir string, req *api.CrosTestRequest, tlwAddr string, tests []*api.TestCaseMetadata) (*api.CrosTestResponse, error) {
	f := &api.CrosTestResponse{}

	serials, err := device.DerviceSerials(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call DeviceSerials: %s", err)
	}

	metadata, _, err := common.UnpackMetadata(req)
	if err != nil {
		md.logger.Println(err)
		return nil, err
	}

	metadata = append(metadata, common.ExtraArgs(req)...)

	err = c.SetupAdbConnections(md.logger, serials)
	if err != nil {
		return f, fmt.Errorf("setupAdb failed, %w", err)
	}

	exit := make(chan struct{})
	defer func() {
		exit <- struct{}{}
		for _, s := range serials {
			if err := c.TeardownAdb(md.logger, s); err != nil {
				md.logger.Printf("Failed to tear down adb connection to %s: %s", s, err)
			}
		}
	}()
	c.KeepAdbAlive(md.logger, serials, exit)

	// Currently, we will just loop each test execution and result processing; and continuously build/append results.
	for _, test := range tests {
		resDir := path.Join("tmp", "logs", "mobly", test.GetTestCase().GetName())
		os.Setenv("MOBLY_LOGPATH", resDir)

		err = runMoblyTest(ctx, md.logger, test, serials, resultsDir, metadata)
		if err != nil {
			md.logger.Println(err)
			return nil, err
		}

		// Override resultsDir to the latest directory.
		resultsDir = path.Join(resDir, "LocalTestBed", "latest")
		resultsPath := path.Join(resultsDir, "test_summary.yaml")

		buildMoblyResult(md.logger, resultsPath, resultsDir, f)
		md.logger.Println("Result after loop1: ", f)
	}

	return f, nil
}

func launchAndReadMobly(cmd *exec.Cmd, logger *log.Logger) error {
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
		common.TestScanner(stderr, logger, "Mobly")
	}()

	go func() {
		defer wg.Done()
		common.TestScanner(stdout, logger, "Mobly")
	}()

	wg.Wait()
	return nil
}
