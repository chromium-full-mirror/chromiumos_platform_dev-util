// Copyright 2021 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package driver implements drivers to execute tests.
package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"

	"chromiumos/lro"

	"go.chromium.org/chromiumos/config/go/test/api"

	"chromiumos/test/execution/cmd/cros-test/internal/common"
	"chromiumos/test/execution/cmd/cros-test/internal/device"
	"chromiumos/test/execution/cmd/cros-test/internal/tautoresults"
)

// TautoDriver runs Tauto and report its results.
type TautoDriver struct {
	// logger provides logging service.
	logger *log.Logger

	// Long running operation manager
	manager *lro.Manager

	// operation name
	op string
}

// NewTautoDriver creates a new driver to run tests.
func NewTautoDriver(logger *log.Logger) *TautoDriver {
	return &TautoDriver{
		logger: logger,
	}
}

// Name returns the name of the driver.
func (td *TautoDriver) Name() string {
	return "tauto"
}

// RunTests drives a test framework to execute tests.
func (td *TautoDriver) RunTests(ctx context.Context, resultsDir string, req *api.CrosTestRequest, tlwAddr string, tests []*api.TestCaseMetadata) (*api.CrosTestResponse, error) {
	testNamesToIds := getTestNamesToIds(tests)
	testNamesToMetadata := getTestNamesToMetadata(tests)
	testNames := getTestNames(tests)

	primary, err := device.FillDUTInfo(req.Primary, "")
	chromeOSCompanions, AndriodCompanions, err := common.Companions(req.Companions)

	// Fill in DUT server var flags.
	var dutServers []string
	if primary.DutServer != "" {
		dutServers = append(dutServers, fmt.Sprintf("%s", primary.DutServer))
	}
	for _, c := range chromeOSCompanions {
		if c.DutServer != "" {
			dutServers = append(dutServers, fmt.Sprintf("%s", c.DutServer))
		}
	}

	// Fill in DUT server var flags.
	var libsServer string
	if primary.LibsServer != "" {
		libsServer = fmt.Sprintf("%s", primary.LibsServer)
	}

	// Get autotest execution args.
	customArgs, err := processArgs(req)
	// Args are not going to be formally supported; so if unpacking fails, log the error
	// and continue.
	if err != nil {
		td.logger.Println("Error during args Parsing, will continue: ", err)

	}
	args, err := newTautoArgs(primary, chromeOSCompanions, AndriodCompanions, testNames, dutServers, resultsDir, libsServer, customArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to create tauto args: %v", err)
	}

	// Run RTD.
	cmd := exec.Command("/usr/bin/test_that", genTautoArgList(args)...)

	td.logger.Println("Running Autotest: ", cmd.String())

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("StderrPipe failed")
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("StdoutPipe failed")
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to run Tauto: %v", err)
	}
	var wg sync.WaitGroup
	wg.Add(2)

	const maxCapacity = 4096 * 1024

	go func() {
		defer wg.Done()
		common.TestScanner(stderr, td.logger, "tauto")
	}()

	go func() {
		defer wg.Done()
		common.TestScanner(stdout, td.logger, "tauto")
	}()

	wg.Wait()

	MissingTestErrMsg := ""
	if err := cmd.Wait(); err != nil {
		td.logger.Println("Failed to run Tauto: ", err)
		MissingTestErrMsg = fmt.Sprintf("Test did not run due to %s", err)
	}

	results, err := tautoresults.TestsReports(
		resultsDir,
		testNames,
		testNamesToIds,
		testNamesToMetadata,
		MissingTestErrMsg,
	)

	if err != nil {
		return &api.CrosTestResponse{}, err
	}

	return &api.CrosTestResponse{TestCaseResults: results}, nil
}

// Flag names. More to be populated once impl details are firmed.
const (
	autotestDirFlag     = "--autotest_dir"
	tautoResultsDirFlag = "--results_dir"
	companionFlag       = "--companion_hosts"
	dutServerFlag       = "--dut_servers"
	libsServerFlag      = "--libs_server"
	// Must be formatted to test_that as follows: ... --host_labels label1 label2 label3
	// Thus, no quotes, etc just a space deliminated list of strings
	labels = "--host_labels"
	// Must be formatted to test_that as follows: ... --host_attributes='{"key": "value"}'
	// Thus, single quoted, with k/v in double quotes.
	attributes = "--host_attributes"
	// Setting the CFT has minor changes in Autotest, such as no exit(1) on failure.
	cft       = "--CFT"
	tautoArgs = "--args"
)

// tautoRunArgs stores arguments to invoke tauto
// Change target from string to the dut api
type tautoRunArgs struct {
	target   *device.DutInfo   // The information of the target machine.
	patterns []string          // The names of test to be run.
	runFlags map[string]string // The flags for tauto run command.
	cftFlag  string
}

// newTautoArgs created an argument structure for invoking tauto
func newTautoArgs(dut *device.DutInfo, companionDuts []*device.DutInfo, andriods []*device.AndroidInfo, tests, dutServers []string, resultsDir string, libsServer string, customArgs string) (*tautoRunArgs, error) {
	args := tautoRunArgs{
		target: dut,
		runFlags: map[string]string{
			autotestDirFlag: common.AutotestDir,
		},
	}

	var companionsAddresses []string
	for _, c := range companionDuts {
		companionsAddresses = append(companionsAddresses, c.Addr)
	}
	for _, a := range andriods {
		companionsAddresses = append(companionsAddresses, a.AssoicateAddr)
	}
	if len(companionsAddresses) > 0 {
		args.runFlags[companionFlag] = strings.Join(companionsAddresses, ",")
	}

	tautoArgsStr := ""
	if len(dutServers) > 0 {
		dutServerAddresses := strings.Join(dutServers, ",")
		args.runFlags[dutServerFlag] = dutServerAddresses
		tautoArgsStr = tautoArgsStr + fmt.Sprintf("%v=%v", "dut_servers", dutServerAddresses)
	}

	if libsServer != "" {
		args.runFlags[libsServerFlag] = libsServer
		tautoArgsStr = tautoArgsStr + fmt.Sprintf(" %v=%v", "libs_server", libsServer)
	}

	if dut.CacheServer != "" {
		tautoArgsStr = tautoArgsStr + fmt.Sprintf(" %v=%v", "cache_endpoint", dut.CacheServer)
	}

	if customArgs != "" {
		if string(customArgs[0]) != " " {
			customArgs = fmt.Sprintf(" %v", customArgs)
		}
		tautoArgsStr = tautoArgsStr + customArgs
	}

	args.runFlags[tautoArgs] = tautoArgsStr

	// Now we need to get a list of all labels, then load the labels const.
	attrMap, infoLabels, err := common.ConvertDutTopologyToHostInfo(dut)
	if err != nil {
		return nil, fmt.Errorf("failed to convert dutotopology: %v", err)
	}

	if len(infoLabels) > 0 {
		args.runFlags[labels] = strings.Join(infoLabels, " ")
	}

	if len(attrMap) > 0 {
		jsonStr, err := json.Marshal(attrMap)
		if err != nil {
			return nil, fmt.Errorf("failed to convert attrMap to string %v", err)
		}
		args.runFlags[attributes] = fmt.Sprintf("%v", string(jsonStr))
	}

	args.cftFlag = cft
	args.patterns = tests // TO-DO Support Tags
	args.runFlags[tautoResultsDirFlag] = resultsDir
	return &args, nil
}

// genTautoArgList generates argument list for invoking Tauto
func genTautoArgList(args *tautoRunArgs) (argList []string) {
	for flag, value := range args.runFlags {
		argList = append(argList, fmt.Sprintf("%v=%v", flag, value))
	}
	argList = append(argList, args.cftFlag)
	argList = append(argList, args.target.Addr)
	argList = append(argList, args.patterns...)
	return argList
}

func processArgs(req *api.CrosTestRequest) (string, error) {
	customArgs := ""
	suites := req.GetTestSuites()
	// In reality; we should never have an empty suite.
	if len(suites) > 0 {
		rawArgs := suites[0].GetExecutionMetadata()
		if rawArgs != nil {
			subCustomArgs := ""
			for _, customAutotestArg := range rawArgs.GetArgs() {
				if customAutotestArg.GetFlag() != "" && customAutotestArg.GetValue() != "" {
					if subCustomArgs != "" {
						subCustomArgs += " "
					}
					subCustomArgs += fmt.Sprintf("%v=%v", customAutotestArg.GetFlag(), customAutotestArg.GetValue())
				}
			}
			return subCustomArgs, nil
		}
	}

	// Backwards compatibility.
	if customArgs == "" {
		md, err := common.UnpackMetadata(req)
		if err != nil {
			return "", err
		}
		customAutotestArgs, _ := md.(*api.AutotestExecutionMetadata)
		subCustomArgs := ""
		for _, customAutotestArg := range customAutotestArgs.GetArgs() {
			subCustomArgs += fmt.Sprintf(" %v=%v", customAutotestArg.GetFlag(), customAutotestArg.GetValue())
		}
		customArgs = subCustomArgs

	}
	return customArgs, nil
}
