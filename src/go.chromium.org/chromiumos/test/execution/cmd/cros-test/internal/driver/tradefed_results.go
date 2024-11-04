// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package driver implements drivers to execute tests.
package driver

import (
	"bufio"
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.chromium.org/chromiumos/config/go/test/api"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	tfStageLogsPath      = "/tmp/stage-android-build-api/stub"
	tfStageResultPattern = "subprocess-test_result.xml_*.xml.gz"
	tfAospResultPattern  = "android-%s/results/latest/test_result.xml"
	tfMetricsPattern     = "metrics-*.xml"
)

var tfEventLogPatterns = []string{"event-logs_*.txt", "aggregated-events*.txt"}

//--------------------------------------------------------------
// Data structures for parsing metrics file XML data.
//--------------------------------------------------------------

type TestSuite struct {
	XMLName   xml.Name   `xml:"testsuite"`
	Name      string     `xml:"name,attr"`
	Tests     int64      `xml:"tests,attr"`
	Failures  int64      `xml:"failures,attr"`
	Time      int64      `xml:"time,attr"`
	Timestamp string     `xml:"timestamp,attr"`
	TestCases []TestCase `xml:"testcase"`
}

type TestCase struct {
	XMLName            xml.Name `xml:"testcase"`
	Name               string   `xml:"testname,attr"`
	ClassName          string   `xml:"classname,attr"`
	Time               int64    `xml:"time,attr"`
	Failure            string   `xml:"FAILURE"`
	Incomplete         string   `xml:"INCOMPLETE"`
	Ignored            string   `xml:"IGNORED"`
	Skipped            string   `xml:"SKIPPED"`
	AsssumptionFailure string   `xml:"ASSUMPTION_FAILURE"`
}

type EventLogTestInfo struct {
	TestName   string
	ModuleName string
	Status     string
}

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

type Module struct {
	Name        string             `xml:"name,attr"`
	Abi         string             `xml:"abi,attr"`
	Runtime     string             `xml:"runtime,attr"`
	Done        string             `xml:"done,attr"`
	Pass        string             `xml:"pass,attr"`
	Total_Tests string             `xml:"total_tests,attr"`
	TestCases   []TradefedTestCase `xml:"TestCase"`
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

func parseEventLogReport(logger *log.Logger, eventFilePath string) map[string]EventLogTestInfo {
	testInfo := make(map[string]EventLogTestInfo)

	eventFile, err := os.Open(eventFilePath)
	if err != nil {
		logger.Println("Error reading staging event log file: ", err)
		return testInfo
	}
	fileScanner := bufio.NewScanner(eventFile)
	fileScanner.Split(bufio.ScanLines)

	moduleStartRE := regexp.MustCompile(`\[module \S+ (\S+) started\]`)
	testRE := regexp.MustCompile(`- test: (\S+) \(status=(\S+)\)`)

	var currentModule string
	for fileScanner.Scan() {
		line := fileScanner.Text()
		if moduleStartRE.MatchString(line) {
			currentModule = moduleStartRE.FindStringSubmatch(line)[1]
		}
		if testRE.MatchString(line) {
			match := testRE.FindStringSubmatch(line)
			testName := match[1]
			status := match[2]
			testInfo[testName] = EventLogTestInfo{TestName: testName,
				ModuleName: currentModule, Status: status}
		}
	}
	eventFile.Close()
	return testInfo
}

// Returns map of full test name to test info (incl. module name & status).
func loadEventLogReport(logger *log.Logger) (map[string]EventLogTestInfo, string) {
	var candidates []string
	for _, pattern := range tfEventLogPatterns {
		eventLogPathPattern := path.Join(tfStageLogsPath, "*", "*", pattern)
		eventPath, err := selectFileByPattern(eventLogPathPattern)
		if err == nil && len(eventPath) > 0 {
			candidates = append(candidates, eventPath)
		}
	}
	if len(candidates) > 0 {
		eventPath := selectMostRecentFile(candidates)
		if len(eventPath) > 0 {
			return parseEventLogReport(logger, eventPath), filepath.Join(filepath.Dir(eventPath), "*")
		}
	}
	logger.Println("Error locating Tradefed staging event log file.")
	return make(map[string]EventLogTestInfo), ""
}

// Returns a full file path of the XML metrics result file.
func getResultFilePath() (string, error) {
	resultsPattern := path.Join(os.Getenv("GLOBAL_LOG_PATH"), tfMetricsPattern)
	resultPath, err := selectFileByPattern(resultsPattern)
	if err != nil {
		return "", err
	}
	return resultPath, nil
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

func normalizeTime(timeStr string) string {
	if strings.Contains(timeStr, "Z") {
		return timeStr
	} else {
		return timeStr + "Z"
	}
}

func processResults(logger *log.Logger, TS TestSuite, testInfoMap map[string]EventLogTestInfo, testType string) []*api.TestCaseResult {
	allTestCases := []*api.TestCaseResult{}

	endTime, err := time.Parse(time.RFC3339, normalizeTime(TS.Timestamp))
	if err != nil {
		logger.Println("Error parsing Tradefed test start time: ", TS.Timestamp)
		endTime = time.Now()
	}
	startTime := endTime.Add(time.Duration(-TS.Time) * time.Millisecond)

	for _, test := range TS.TestCases {
		fullTestName := fmt.Sprintf("%s#%s", test.ClassName, test.Name)
		status := "UNKNOWN" // Default status
		testInfo, ok := testInfoMap[fullTestName]
		if ok {
			status = testInfo.Status
		}
		// TODO; if no testInfo matched, calculate status.
		errorText := getErrorText(status, test)
		allTestCases = append(allTestCases,
			buildTcResult(fmt.Sprintf("tradefed.%s.%s#%s", testType, testInfo.ModuleName, fullTestName),
				status, startTime, test.Time/1000, errorText))
	}

	return allTestCases
}

func getErrorText(status string, test TestCase) string {
	errorText := ""
	switch status {
	case "ASSUMPTION_FAILURE":
		errorText = test.AsssumptionFailure
	case "FAILED":
		errorText = test.Failure
	case "INCOMPLETE":
		errorText = test.Incomplete
	case "SKIPPED":
		errorText = test.Skipped
	case "IGNORED":
		errorText = test.Ignored
	default:
		errorText = "Unknown test status"
	}
	return errorText
}

func processGivenTestResults(logger *log.Logger, TS TestSuite, testInfoMap map[string]EventLogTestInfo) []*api.CrosTestResponse_GivenTestResult {
	givenTestCases := []*api.CrosTestResponse_GivenTestResult{}
	// map format for holding parent child like relationship. This will be used later to populate givenTestCases
	testMap := make(map[string][]*api.TestCaseResult)

	startTime, err := time.Parse(time.RFC3339, normalizeTime(TS.Timestamp))
	if err != nil {
		logger.Println("Error parsing Tradefed test start time: ", TS.Timestamp)
		startTime = time.Now()
	}

	logger.Println("Creating test map with parent/module as key and test case result as child list")
	// for all test cases parsed from metrics-*.xml
	for _, test := range TS.TestCases {
		fullTestName := fmt.Sprintf("%s#%s", test.ClassName, test.Name)
		testInfo, ok := testInfoMap[fullTestName]
		// check if exists in testInfoMap parsed from aggregated-events*.txt, testInfoMap has module name, status info as value for each test in map
		if ok {
			// form child test case result
			testCaseResult := buildTcResult(fullTestName, testInfo.Status, startTime, test.Time/1000, getErrorText(testInfo.Status, test))
			// add to child case list by module
			if val, ok := testMap[testInfo.ModuleName]; ok {
				testMap[testInfo.ModuleName] = append(val, testCaseResult)
			} else {
				testMap[testInfo.ModuleName] = []*api.TestCaseResult{testCaseResult}
			}
		}
	}

	// create given test cases object from the testMap
	logger.Println("Creating given test case results object from test map")
	for parent, childCaseList := range testMap {
		paretTestName := fmt.Sprintf("tradefed.%s.%s", testType, parent)
		givenTestCases = append(givenTestCases, &api.CrosTestResponse_GivenTestResult{
			ParentTest:           paretTestName,
			ChildTestCaseResults: childCaseList,
		})
		logger.Printf("Given test case with parent: %s added with %d child test cases\n", paretTestName, len(childCaseList))
	}

	return givenTestCases
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
	allTestCases = generateTestCaseResult(logger, testType, R, req)
	givenTestCases := generateGivenTestCases(logger, testType, R, req)

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

func generateTestCaseResult(logger *log.Logger, testType string, R Result, req *api.CrosTestRequest) []*api.TestCaseResult {
	allTestCases := []*api.TestCaseResult{}
	startTimeMs, _ := strconv.ParseInt(R.Start, 10, 64)
	startTime := time.UnixMilli(startTimeMs)
	logger.Println("Found n module results:", len(R.Modules))
	modulesFromReq := getAllModulesFromReq(req)
	for _, module := range R.Modules {
		delete(modulesFromReq, fmt.Sprintf("tradefed.%s.%s", testType, module.Name))
		nTests, _ := strconv.Atoi(module.Total_Tests)
		runTime, _ := strconv.Atoi(module.Runtime)
		duration := runTime
		if nTests != 0 {
			duration = runTime / nTests
		} else {
			// mark module as Crash if no tests were executed for the module
			allTestCases = append(allTestCases,
				buildTcResult(fmt.Sprintf("tradefed.%s.%s", testType, module.Name), "INCOMPLETE", startTime, 1, ""))
		}

		logger.Println("Parsing Module: ", module.Name)
		// Make a duration of atleast 1 second so the proto isnt empty.
		testDur := int64(duration / 1000)
		if testDur == 0 {
			testDur = 1
		}

		for _, testcase := range module.TestCases {
			for _, test := range testcase.Tests {
				testName := fmt.Sprintf("tradefed.%s.%s#%s#%s", testType, module.Name, testcase.Name, test.Name)
				allTestCases = append(allTestCases,
					buildTcResult(testName, test.Result, startTime, testDur, test.Failure.Message))
			}
		}
	}
	for moduleName := range modulesFromReq {
		allTestCases = append(allTestCases, buildTcResult(moduleName, "INCOMPLETE", startTime, 1, ""))
	}
	return allTestCases
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

func generateGivenTestCases(logger *log.Logger, testType string, R Result, req *api.CrosTestRequest) []*api.CrosTestResponse_GivenTestResult {
	givenTestCases := []*api.CrosTestResponse_GivenTestResult{}
	startTimeMs, _ := strconv.ParseInt(R.Start, 10, 64)
	startTime := time.UnixMilli(startTimeMs)
	logger.Println("Found n module results:", len(R.Modules))
	modulesFromReq := getAllModulesFromReq(req)
	for _, module := range R.Modules {
		delete(modulesFromReq, fmt.Sprintf("tradefed.%s.%s", testType, module.Name))
		nTests, _ := strconv.Atoi(module.Total_Tests)
		runTime, _ := strconv.Atoi(module.Runtime)
		duration := runTime
		if nTests != 0 {
			duration = runTime / nTests
		} else {
			// mark module as Crash if no tests were executed for the module
			givenTestCases = append(givenTestCases, &api.CrosTestResponse_GivenTestResult{
				ParentTest:           fmt.Sprintf("tradefed.%s.%s", testType, module.Name),
				ChildTestCaseResults: []*api.TestCaseResult{buildTcResult(fmt.Sprintf("tradefed.%s.%s", testType, module.Name), "INCOMPLETE", startTime, 1, "")},
			})

		}

		logger.Println("Parsing Module: ", module.Name)
		// Make a duration of atleast 1 second so the proto isnt empty.
		testDur := int64(duration / 1000)
		if testDur == 0 {
			testDur = 1
		}
		givenTestCase := &api.CrosTestResponse_GivenTestResult{}
		for _, testcase := range module.TestCases {

			childTestCases := []*api.TestCaseResult{}
			for _, test := range testcase.Tests {
				testName := fmt.Sprintf("%s#%s", testcase.Name, test.Name)
				childTestCases = append(childTestCases,
					buildTcResult(testName, test.Result, startTime, testDur, test.Failure.Message))
			}
			givenTestCase.ParentTest = fmt.Sprintf("tradefed.%s.%s", testType, module.Name)
			givenTestCase.ChildTestCaseResults = childTestCases
		}
		givenTestCases = append(givenTestCases, givenTestCase)
	}
	for moduleName := range modulesFromReq {
		givenTestCases = append(givenTestCases,
			&api.CrosTestResponse_GivenTestResult{ParentTest: moduleName, ChildTestCaseResults: []*api.TestCaseResult{buildTcResult(moduleName, "INCOMPLETE", startTime, 1, "")}})
	}
	return givenTestCases
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
		logger.Println("Trying to locate and parse metric result file...")
		resultPath, err := getResultFilePath()
		if err != nil {
			logger.Println("Error locating Tradefed metric result file: ", err)
			return testResponseNoXMLFound(req), logsToSave
		} else {
			logsToSave = append(logsToSave, resultPath)

			TS := TestSuite{}
			if err = parseResultFile(logger, resultPath, &TS); err != nil {
				return testResponseNoXMLFound(req), logsToSave
			}

			testInfoMap, eventLogs := loadEventLogReport(logger)
			if len(eventLogs) > 0 {
				logsToSave = append(logsToSave, eventLogs)
			}

			logger.Printf("Parsing test suite %s; tests: %d, failures: %d",
				TS.Name, TS.Tests, TS.Failures)

			f.TestCaseResults = processResults(logger, TS, testInfoMap, testType)
			f.GivenTestResults = processGivenTestResults(logger, TS, testInfoMap)

			// report modules as Crash for which no tests were reported.
			modulesWithNoResult := noResultModules(logger, TS, testInfoMap, req, testType)
			for module := range modulesWithNoResult {
				crashStatusTestResult := buildTcResult(module, "INCOMPLETE", time.Now().Add(-1*time.Second), 1, "")
				f.TestCaseResults = append(f.TestCaseResults, crashStatusTestResult)
				f.GivenTestResults = append(f.GivenTestResults, &api.CrosTestResponse_GivenTestResult{
					ParentTest:           module,
					ChildTestCaseResults: []*api.TestCaseResult{crashStatusTestResult},
				})
			}
		}
	}
	logger.Println("Found n case results:", len(f.TestCaseResults))

	return f, logsToSave
}

func noResultModules(logger *log.Logger, TS TestSuite, testInfoMap map[string]EventLogTestInfo, req *api.CrosTestRequest, testType string) map[string]struct{} {
	modulesFromRequest := getAllModulesFromReq(req)
	for _, test := range TS.TestCases {
		fullTestName := fmt.Sprintf("%s#%s", test.ClassName, test.Name)
		testInfo, ok := testInfoMap[fullTestName]
		if ok {
			// remove module if a test has been reported for it.
			delete(modulesFromRequest, fmt.Sprintf("tradefed.%s.%s", testType, testInfo.ModuleName))
		}
	}
	// logging modules which were not found in results from xml
	for module := range modulesFromRequest {
		logger.Printf(module)
	}
	return modulesFromRequest
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
