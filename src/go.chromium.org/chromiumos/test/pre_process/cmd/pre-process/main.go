// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package main implements the pre-process for finding tests based on tags.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"go.chromium.org/chromiumos/config/go/test/api"

	"go.chromium.org/chromiumos/test/execution/errors"
	"go.chromium.org/chromiumos/test/pre_process/cmd/pre-process/interfaces"
	"go.chromium.org/chromiumos/test/pre_process/cmd/pre-process/policies"
	"go.chromium.org/chromiumos/test/pre_process/cmd/pre-process/structs"

	"go.chromium.org/chromiumos/test/util/helpers"

	"google.golang.org/protobuf/encoding/protojson"
)

const (
	defaultRootPath       = "/tmp/test/pre-process"
	defaultInputFileName  = "request.json"
	defaultOutputFileName = "result.json"
)

// Filter struct shows what was removed, notfound etc
type Filter struct {
	data     map[string]structs.SignalFormat
	removed  []string
	notFound []string
	req      *api.FilterFlakyRequest
}

func readInput(fileName string) (*api.FilterFlakyRequest, error) {
	f, err := os.ReadFile(fileName)
	if err != nil {
		return nil, errors.NewStatusError(errors.IOAccessError,
			fmt.Errorf("fail to read file %v: %v", fileName, err))
	}
	req := api.FilterFlakyRequest{}
	if err := protojson.Unmarshal(f, &req); err != nil {
		return nil, errors.NewStatusError(errors.UnmarshalError,
			fmt.Errorf("fail to unmarshal file %v: %v", fileName, err))
	}

	return &req, nil
}

// writeOutput writes a FilterFlakyResponse json.
func writeOutput(output string, resp *api.FilterFlakyResponse) error {
	f, err := os.Create(output)
	if err != nil {
		return errors.NewStatusError(errors.IOCreateError,
			fmt.Errorf("fail to create file %v: %v", output, err))
	}
	if json, err := protojson.Marshal(resp); err != nil {
		return errors.NewStatusError(errors.MarshalError,
			fmt.Errorf("failed to marshall response to file %v: %v", output, err))
	} else {
		f.Write(json)
	}
	return nil
}

// Version is the version info of this command. It is filled in during emerge.
var Version = "<unknown>"

type args struct {
	// Common input params.
	logPath   string
	inputPath string
	output    string
	version   bool
}

// filterCases based on the preset f.data. NOTE: f.data must be populated before calling.
func (f *Filter) filterCases(testcases []*api.TestCase, name string) *api.TestSuite {
	keepCases := []*api.TestCase{}
	fmt.Println("Filtering Cases")

	// Iterate through the TC, keep the ones which are given the green.
	for _, tc := range testcases {
		value := tc.Id.Value
		if f.testEligible(value) {
			keepCases = append(keepCases, tc)
		}
	}

	return &api.TestSuite{
		Name: name,
		Spec: &api.TestSuite_TestCases{
			TestCases: &api.TestCaseList{TestCases: keepCases},
		},
	}

}

// filterMetadata based on the preset f.data. NOTE: f.data must be populated before calling.
func (f *Filter) filterMetadata(op *api.TestSuite_TestCasesMetadata, name string) *api.TestSuite {
	keepMetas := []*api.TestCaseMetadata{}
	mdl := op.TestCasesMetadata.Values

	for _, tc := range mdl {
		value := tc.TestCase.Id.Value
		if f.testEligible(value) {
			keepMetas = append(keepMetas, tc)
		}
	}

	return &api.TestSuite{
		Name: name,
		Spec: &api.TestSuite_TestCasesMetadata{
			TestCasesMetadata: &api.TestCaseMetadataList{Values: keepMetas},
		},
	}
}

// testEligible will return false if the stabiliyData shows the test is unstable, else true.
// This indicates that if there is no stabilityData, we will return true.
func (f *Filter) testEligible(value string) bool {
	// This loop doesn't "determine" eligibity, as that is left to the policy.
	// It is simply looping through the given tests, and checking for their "signal" (ie, eligibity)
	// and applying returning that; in conjunction with applying the defaultEnabled rule.
	_, ok := f.data[value]
	isAllowed := false

	// If the test is found in the stabilityData, check for its signal and use that
	if ok {
		isAllowed = f.data[value].Signal
		if isAllowed {
			return true
		}
		f.removed = append(f.removed, value)
	} else {
		// If the test is not found, and `DefaultEnabled` is set, keep the test
		f.notFound = append(f.notFound, value)
		if f.req.DefaultEnabled {
			return true
		}
	}
	log.Printf("Test %s marked as not eligible.\n", value)
	return false
}

func getStabilityData(req *api.FilterFlakyRequest, tcList map[string]struct{}) (map[string]structs.SignalFormat, error) {
	var data map[string]structs.SignalFormat

	// TODO: this datatype will need to evolve from a board string to something more complex.
	var variant string
	switch variantOp := req.Variant.(type) {
	case *api.FilterFlakyRequest_Board:
		variant = variantOp.Board
	}
	if variant == "" {
		fmt.Println("No variant")
		return nil, fmt.Errorf("no variant provided, cannot filter")
	}

	// Currently 2 types of policies will be supported. This can be expanded if newer types are added.
	switch op := req.Policy.(type) {
	case *api.FilterFlakyRequest_PassRatePolicy:
		return policies.StabilityFromPolicy(op, variant, req.Milestone, tcList)
	case *api.FilterFlakyRequest_StabilitySensorPolicy:
		return policies.StabilityFromStabilitySensor()
	}

	return data, nil
}

func testMDToSet(op *api.TestSuite_TestCasesMetadata) map[string]struct{} {
	tcList := make(map[string]struct{})
	mdl := op.TestCasesMetadata.Values
	for _, tc := range mdl {
		value := tc.TestCase.Id.Value
		tcList[value] = struct{}{}
	}

	return tcList
}

func testCasesToSet(testcases []*api.TestCase) map[string]struct{} {
	tcList := make(map[string]struct{})
	for _, tc := range testcases {
		value := tc.Id.Value
		tcList[value] = struct{}{}
	}
	return tcList
}

func innerMain(req *api.FilterFlakyRequest) (*api.FilterFlakyResponse, error) {
	filter := Filter{req: req}
	var err error
	var filteredSuites []*api.TestSuite
	for _, md := range req.TestSuites {
		// The input request, TestSuites, can be either TestCases OR TestCaseMetadata. Support both options.
		var filteredSuite *api.TestSuite
		switch op := md.Spec.(type) {
		case *api.TestSuite_TestCases:

			// Generate a set of tests, these will be used when searching for signal on the tests.
			testCases := op.TestCases.TestCases
			filter.data, err = getStabilityData(filter.req, testCasesToSet(testCases))
			if err != nil {
				fmt.Printf("err during stability fetching, will apply rules as possible %s,", err)
			}
			filteredSuite = filter.filterCases(testCases, md.Name)
		case *api.TestSuite_TestCasesMetadata:
			// Generate a set of tests, these will be used when searching for signal on the tests.
			filter.data, err = getStabilityData(filter.req, testMDToSet(op))
			if err != nil {
				fmt.Printf("err during stability fetching, will apply rules as possible %s,", err)
			}
			filteredSuite = filter.filterMetadata(op, md.Name)
		}
		filteredSuites = append(filteredSuites, filteredSuite)

	}

	rspn := &api.FilterFlakyResponse{
		TestSuites:   filteredSuites,
		RemovedTests: filter.removed,
	}

	log.Printf("The following tests are set to be removed from this scheduled attempt:\n %s\n", rspn.RemovedTests)

	err = interfaces.WriteResults(rspn.RemovedTests, req, filter.data)
	if err != nil {

		log.Println("!!!")
		log.Println(err)
	}
	return rspn, nil

}

// runCLI is the entry point for running cros-test (PreProcess) in CLI mode.
func runCLI(ctx context.Context, d []string) int {
	t := time.Now()
	defaultLogPath := filepath.Join(defaultRootPath, t.Format("20060102-150405"))
	defaultRequestFile := filepath.Join(defaultRootPath, defaultInputFileName)
	defaultResultFile := filepath.Join(defaultRootPath, defaultOutputFileName)

	a := args{}

	fs := flag.NewFlagSet("Run pre-process", flag.ExitOnError)
	fs.StringVar(&a.logPath, "log", defaultLogPath, fmt.Sprintf("Path to record pre-process logs. Default value is %s", defaultLogPath))
	fs.StringVar(&a.inputPath, "input", defaultRequestFile, "specify the test filter request json input file")
	fs.StringVar(&a.output, "output", defaultResultFile, "specify the test filter request json input file")
	fs.BoolVar(&a.version, "version", false, "print version and exit")
	fs.Parse(d)

	if a.version {
		fmt.Println("pre-process version ", Version)
		return 0
	}

	logFile, err := helpers.CreateLogFile(a.logPath)
	if err != nil {
		log.Fatalln("Failed to create log file", err)
		return 2
	}
	defer logFile.Close()
	mw := io.MultiWriter(logFile, os.Stderr)
	log.SetOutput(mw)

	log.Println("pre-process version ", Version)

	log.Println("Reading input file: ", a.inputPath)
	req, err := readInput(a.inputPath)
	if err != nil {
		log.Println("Error: ", err)
		return errors.WriteError(os.Stderr, err)
	}

	rspn, err := innerMain(req)
	if err != nil {
		return 2
	}

	log.Println("Writing output file: ", a.output)
	if err := writeOutput(a.output, rspn); err != nil {
		log.Println("Error: ", err)
		return errors.WriteError(os.Stderr, err)
	}

	return 0
}

func mainInternal(ctx context.Context) int {
	return runCLI(ctx, os.Args[1:])
}

func main() {
	os.Exit(mainInternal(context.Background()))
}
