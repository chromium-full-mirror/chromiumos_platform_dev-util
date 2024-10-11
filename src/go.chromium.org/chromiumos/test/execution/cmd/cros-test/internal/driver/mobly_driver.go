// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package driver implements drivers to execute tests.
package driver

import (
	"context"
	"errors"
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

func buildMoblyCommand(logger *log.Logger, tests []*api.TestCaseMetadata, resultsPath string) []string {
	logger.Println("Building mobly command")
	baseArgs := []string{
		"local_mobly_runner.py",
		"-p", "betocq_test_suite",
		"-i",
		"-c", "test_config.yml",
		"-lp", resultsPath,
	}

	baseArgs = append(baseArgs, "--tests")
	for _, test := range tests {
		baseArgs = append(baseArgs, test.GetTestCase().GetName())
	}

	return baseArgs
}

// Execute Mobly command against Omnilab Gateway Client and process test results
func runMoblyTest(ctx context.Context, logger *log.Logger, tests []*api.TestCaseMetadata, serials []string, resultsPath string, metadata []*api.Arg) error {
	err := c.SetupAdbConnections(logger, serials)
	if err != nil {
		return fmt.Errorf("setupAdb failed, %w", err)
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

	if err := GenerateMoblyConfig(logger, "/usr/local/mobly", serials, metadata); err != nil {
		return fmt.Errorf("failed to generate mobly config, %w", err)
	}

	cmdArgs := buildMoblyCommand(logger, tests, resultsPath)
	cmd := exec.Command("python3", cmdArgs...)
	cmd.Dir = "/usr/local/mobly"
	logger.Println("Running Mobly: ", cmd.String())

	c.KeepAdbAlive(logger, serials, exit)
	return launchAndReadMobly(cmd, logger)
}

func buildMoblyResult(logger *log.Logger, resultsPath string) *api.CrosTestResponse {
	defer func() {
		if r := recover(); r != nil {
			logger.Println("found a panic")
			logger.Printf("Error: %+v", errors.New(fmt.Sprintf("%v", r)))
			logger.Println(string(debug.Stack()))
		}
	}()

	logger.Println("building results")
	f := &api.CrosTestResponse{}

	logger.Println("reading file")
	moblyResultBytes, err := os.ReadFile(resultsPath)
	logger.Println("reading file done")
	if err != nil {
		logger.Printf("failed to read %s, %s", resultsPath, err)
		return nil
	}

	logger.Println("translating")
	testCaseResults, err := common.TranslateMoblyResults(moblyResultBytes)
	logger.Println("translating done")
	if err != nil {
		logger.Printf("failed to translate mobly results, %s", err)
		return nil
	}

	f.TestCaseResults = testCaseResults

	logger.Println("Got results:")
	for _, result := range f.TestCaseResults {
		logger.Println(result.String())
	}

	return f
}

// RunTests drives a test framework to execute tests.
func (md *MoblyDriver) RunTests(ctx context.Context, resultsDir string, req *api.CrosTestRequest, tlwAddr string, tests []*api.TestCaseMetadata) (*api.CrosTestResponse, error) {
	allRspn := &api.CrosTestResponse{}

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

	err = runMoblyTest(ctx, md.logger, tests, serials, resultsDir, metadata)
	if err != nil {
		md.logger.Println(err)
		return nil, err
	}

	resultsPath := path.Join(resultsDir, "betocq_test_suite", "LocalTestBed", "latest", "test_summary.yaml")
	allRspn = buildMoblyResult(md.logger, resultsPath)

	return allRspn, nil
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
