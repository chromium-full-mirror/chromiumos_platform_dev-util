// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package driver implements drivers to execute tests.
package driver

import (
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"go.chromium.org/chromiumos/config/go/test/api"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	tfStageLogsPath      = "/tmp/stage-android-build-api/stub"
	tfStageResultPattern = "subprocess-test_result.xml_*.xml.gz"
	tfAospResultPattern  = "android-%s/results/latest/test_result.xml"
	incompleteError      = "Module is missing from results."
)

//--------------------------------------------------------------
// Data structures for parsing compatibility reporter XML data.
//--------------------------------------------------------------

type TradeFedTestFailure struct {
	Message string `xml:"message,attr"`
}

type TradeFedTest struct {
	Result  string              `xml:"result,attr"`
	Name    string              `xml:"name,attr"`
	Failure TradeFedTestFailure `xml:"Failure"`
}

type TradefedTestCase struct {
	Name  string         `xml:"name,attr"`
	Tests []TradeFedTest `xml:"Test"`
}

type Reason struct {
	Message   string `xml:"message,attr"`
	ErrorName string `xml:"error_name,attr"`
	ErrorCode string `xml:"error_code,attr"`
}

type Module struct {
	Name        string             `xml:"name,attr"`
	Abi         string             `xml:"abi,attr"`
	Runtime     string             `xml:"runtime,attr"`
	Done        string             `xml:"done,attr"`
	Pass        string             `xml:"pass,attr"`
	Total_Tests string             `xml:"total_tests,attr"`
	TestCases   []TradefedTestCase `xml:"TestCase"`
	Reason      Reason             `xml:"Reason"`
}

type Result struct {
	// Start time and end time in milliseconds.
	Start    string   `xml:"start,attr"`
	End      string   `xml:"end,attr"`
	BuildNum string   `xml:"suite_build_number,attr"`
	Modules  []Module `xml:"Module"`
}

func selectMostRecentFile(files []string) string {
	var recentFile string
	var recentTime int64 = 0
	for _, file := range files {
		fi, err := os.Stat(file)
		if err == nil {
			fileTime := fi.ModTime().Unix()
			if fileTime > recentTime {
				recentTime = fileTime
				recentFile = file
			}
		}
	}
	return recentFile
}

func selectFileByPattern(pattern string) (string, error) {
	files, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no files found matching pattern: %s", pattern)
	}

	if len(files) == 1 {
		return files[0], nil
	}

	recentFile := selectMostRecentFile(files)
	if len(recentFile) > 0 {
		return recentFile, nil
	} else {
		return "", fmt.Errorf("no files found matching pattern: %s", pattern)
	}
}

// Unmarshal XML result file into a given struct.
func parseResultFile(logger *log.Logger, resultPath string, v any) error {
	resultData, err := os.ReadFile(resultPath)
	if err != nil {
		logger.Println("Error reading Tradefed result file: ", err)
		return err
	}
	err = xml.Unmarshal(resultData, v)
	// err = parseFunc(resultData)
	if err != nil {
		logger.Println("Error parsing Tradefed result file: ", err)
		return err
	}
	return nil
}

func uncompressGzip(gzFile string, outFileName string) error {
	gz, err := os.Open(gzFile)
	if err != nil {
		return fmt.Errorf("Error opening GZIP file: %s, error: %v", gzFile, err)
	}
	uncompressedStream, err := gzip.NewReader(gz)
	if err != nil {
		return fmt.Errorf("Error extracting GZIP file: %s, error: %v", gzFile, err)
	}
	// Make sure target path exists.
	targetDir := filepath.Dir(outFileName)
	cmd := exec.Command("mkdir", "-p", targetDir)
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("error creating target directory: %s, error: %s", targetDir, err)
	}
	outFile, err := os.Create(outFileName)
	if err != nil {
		return fmt.Errorf("Error creating output file: %s, error: %v", gzFile, err)
	}
	defer outFile.Close()
	if _, err = io.Copy(outFile, uncompressedStream); err != nil {
		return fmt.Errorf("Error extracting file: %s, error: %v", gzFile, err)
	}
	return nil
}

func parseCompatibilityXmlResults(logger *log.Logger, testType string, req *api.CrosTestRequest) ([]*api.TestCaseResult, []*api.CrosTestResponse_GivenTestResult, []string, error) {
	var logsDir []string
	allTestCases := []*api.TestCaseResult{}

	aospResultXmlFile := filepath.Join(tradefedDir, fmt.Sprintf(tfAospResultPattern, testType))
	xmlResultFile, err := selectFileByPattern(aospResultXmlFile)
	if err != nil {
		// Fallback to stub location for XML result file.
		xmlResultFile, err = selectFileByPattern(filepath.Join(tfStageLogsPath, "*", "*", tfStageResultPattern))
		if err != nil {
			return allTestCases, nil, logsDir, fmt.Errorf("failed to locate result file: %w", err)
		} else {
			standardResultFileName := filepath.Join(filepath.Dir(xmlResultFile), filepath.Base(aospResultXmlFile))
			// Unzip stub location XML result file to the standard file name.
			// This is required by CTS Archiver.
			logger.Printf("Extracting stub XML result file: %s to: %s",
				xmlResultFile, standardResultFileName)

			if err = uncompressGzip(xmlResultFile, standardResultFileName); err != nil {
				return allTestCases, nil, logsDir, fmt.Errorf("failed to prepare result file: %w", err)
			} else {
				xmlResultFile = standardResultFileName
			}
		}
	}

	logsDir = append(logsDir, filepath.Join(filepath.Dir(xmlResultFile), "*"))

	logger.Println("Parsing results from: ", xmlResultFile)
	R := Result{}
	if err = parseResultFile(logger, xmlResultFile, &R); err != nil {
		return allTestCases, nil, logsDir, err
	}
	allTestCases, givenTestCases := generateTestCaseResult(logger, testType, R, req)

	return allTestCases, givenTestCases, logsDir, nil
}

func testResponseNoXMLFound(req *api.CrosTestRequest) *api.CrosTestResponse {
	r := &api.CrosTestResponse{}
	if req == nil || req.GetTestSuites() == nil {
		return r
	}
	for _, suites := range req.GetTestSuites() {
		for _, testCaseIds := range suites.GetTestCaseIds().GetTestCaseIds() {
			testCaseResult := buildTcResult(testCaseIds.GetValue(), "INCOMPLETE", time.Now().Add(-1*time.Second), 1, "")
			r.TestCaseResults = append(r.TestCaseResults, testCaseResult)
			r.GivenTestResults = append(r.GivenTestResults, &api.CrosTestResponse_GivenTestResult{
				ParentTest:           testCaseIds.GetValue(),
				ChildTestCaseResults: []*api.TestCaseResult{testCaseResult},
			})
		}
	}
	return r
}

func generateTestCaseResult(logger *log.Logger, testType string, R Result, req *api.CrosTestRequest) ([]*api.TestCaseResult, []*api.CrosTestResponse_GivenTestResult) {
	allTestCases := []*api.TestCaseResult{}
	givenTestCases := []*api.CrosTestResponse_GivenTestResult{}
	startTimeMs, _ := strconv.ParseInt(R.Start, 10, 64)
	startTime := time.UnixMilli(startTimeMs)
	logger.Println("Found n module results:", len(R.Modules))
	modulesFromReq := getAllModulesFromReq(req)
	for _, module := range R.Modules {
		moduleName := fmt.Sprintf("tradefed.%s.%s", testType, module.Name)
		logger.Println("Parsing Module: ", module.Name)
		delete(modulesFromReq, moduleName)
		nTests, _ := strconv.Atoi(module.Total_Tests)
		runTime, _ := strconv.Atoi(module.Runtime)
		// Make a duration of atleast 1 second so the proto isnt empty.
		testDur := int64(runTime / 1000)
		if testDur == 0 {
			testDur = 1
		}
		if nTests != 0 {
			testDur = testDur / int64(nTests)
		} else {
			// Check and report module-level error.
			if module.Reason.Message != "" {
				allTestCases = append(allTestCases,
					buildTcResult(moduleName, "FAILED", startTime, testDur, module.Reason.Message))
				givenTestCases = append(givenTestCases, &api.CrosTestResponse_GivenTestResult{
					ParentTest: moduleName,
					ChildTestCaseResults: []*api.TestCaseResult{
						buildTcResult(moduleName, "FAILED", startTime, testDur, module.Reason.Message),
					},
				})
			} else {
				// No tests in the module and no errors, so reporting this module as 'pass'.
				allTestCases = append(allTestCases,
					buildTcResult(moduleName, "PASSED", startTime, testDur, ""))
				givenTestCases = append(givenTestCases, &api.CrosTestResponse_GivenTestResult{
					ParentTest: moduleName,
					ChildTestCaseResults: []*api.TestCaseResult{
						buildTcResult(moduleName, "PASSED", startTime, testDur, ""),
					},
				})
			}
		}

		givenTestCase := &api.CrosTestResponse_GivenTestResult{}
		givenTestCase.ParentTest = moduleName
		childTestCases := []*api.TestCaseResult{}
		for _, testcase := range module.TestCases {
			for _, test := range testcase.Tests {
				fullTestName := fmt.Sprintf("tradefed.%s.%s#%s#%s", testType, module.Name, testcase.Name, test.Name)
				allTestCases = append(allTestCases,
					buildTcResult(fullTestName, test.Result, startTime, testDur, test.Failure.Message))
				testName := fmt.Sprintf("%s#%s", testcase.Name, test.Name)
				childTestCases = append(childTestCases,
					buildTcResult(testName, test.Result, startTime, testDur, test.Failure.Message))
			}
		}
		givenTestCase.ChildTestCaseResults = childTestCases
		givenTestCases = append(givenTestCases, givenTestCase)
	}
	for moduleName := range modulesFromReq {
		allTestCases = append(allTestCases,
			buildTcResult(moduleName, "INCOMPLETE", startTime, 1, incompleteError))
		givenTestCases = append(givenTestCases,
			&api.CrosTestResponse_GivenTestResult{
				ParentTest: moduleName,
				ChildTestCaseResults: []*api.TestCaseResult{
					buildTcResult(moduleName, "INCOMPLETE", startTime, 1, incompleteError),
				},
			})
	}
	return allTestCases, givenTestCases
}

func getAllModulesFromReq(req *api.CrosTestRequest) map[string]struct{} {
	modules := make(map[string]struct{})
	if req == nil || req.GetTestSuites() == nil {
		return modules
	}
	for _, suites := range req.GetTestSuites() {
		for _, testCaseIds := range suites.GetTestCaseIds().GetTestCaseIds() {
			modules[testCaseIds.GetValue()] = struct{}{}
		}
	}
	return modules
}

func buildTradefedResult(logger *log.Logger, testType string, req *api.CrosTestRequest) (*api.CrosTestResponse, []string) {
	// List of glob patterns of artifacts that should be saved in test results.
	var logsToSave = []string{}

	logger.Println("Building results...")
	f := &api.CrosTestResponse{}

	var err error
	f.TestCaseResults, f.GivenTestResults, logsToSave, err = parseCompatibilityXmlResults(logger, testType, req)
	if err != nil {
		logger.Println("Error locating or parsing Tradefed compatibility result file: ", err)
		return testResponseNoXMLFound(req), logsToSave
	}
	logger.Println("Found n case results:", len(f.TestCaseResults))

	return f, logsToSave
}

func buildTcResult(testName string, testStatus string, startTime time.Time, duration int64,
	errorMessage string) *api.TestCaseResult {

	tcResult := new(api.TestCaseResult)

	tcResult.TestHarness = &api.TestHarness{TestHarnessType: &api.TestHarness_Tradefed_{Tradefed: &api.TestHarness_Tradefed{}}}
	tcResult.TestCaseId = &api.TestCase_Id{Value: testName}

	tcResult.StartTime = timestamppb.New(startTime)
	// Make a duration of atleast 1 second so the proto isnt empty.
	testDuration := duration
	if testDuration == 0 {
		testDuration = 1
	}
	tcResult.Duration = &durationpb.Duration{Seconds: duration}

	testError := &api.TestCaseResult_Error{Message: errorMessage}

	switch testStatus {
	case "PASSED", "pass":
		tcResult.Verdict = &api.TestCaseResult_Pass_{Pass: &api.TestCaseResult_Pass{}}
	case "ASSUMPTION_FAILURE":
		tcResult.Verdict = &api.TestCaseResult_Pass_{Pass: &api.TestCaseResult_Pass{}}
		tcResult.Errors = []*api.TestCaseResult_Error{testError}
	case "FAILED", "fail":
		tcResult.Verdict = &api.TestCaseResult_Fail_{Fail: &api.TestCaseResult_Fail{}}
		tcResult.Errors = []*api.TestCaseResult_Error{testError}
	case "INCOMPLETE":
		tcResult.Verdict = &api.TestCaseResult_Crash_{Crash: &api.TestCaseResult_Crash{}}
		tcResult.Errors = []*api.TestCaseResult_Error{testError}
	case "SKIPPED":
		tcResult.Verdict = &api.TestCaseResult_Skip_{Skip: &api.TestCaseResult_Skip{}}
		tcResult.Errors = []*api.TestCaseResult_Error{testError}
	case "IGNORED":
		tcResult.Verdict = &api.TestCaseResult_Skip_{Skip: &api.TestCaseResult_Skip{}}
		tcResult.Errors = []*api.TestCaseResult_Error{testError}
	default:
		tcResult.Verdict = &api.TestCaseResult_Fail_{Fail: &api.TestCaseResult_Fail{}}
		tcResult.Errors = []*api.TestCaseResult_Error{testError}
	}

	return tcResult
}
