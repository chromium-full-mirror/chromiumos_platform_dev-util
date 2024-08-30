// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package driver implements drivers to execute tests.
package driver

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	"go.chromium.org/chromiumos/config/go/test/api"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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
	Build    Build    `xml:"Build"`
	Summary  Summary  `xml:"Summary"`
	Modules  []Module `xml:"Module"`
}

type Build struct{}

type Summary struct {
	Pass         string `xml:"pass,attr"`
	Failed       string `xml:"failed,attr"`
	ModulesDone  string `xml:"modules_done,attr"`
	ModulesTotal string `xml:"modules_total,attr"`
}

func readResults(logger *log.Logger, resultsPath string) []byte {
	xmlFile, err := os.Open(resultsPath)
	if err != nil {
		logger.Println(err)
	}

	logger.Println("Successfully Opened users.xml")
	defer xmlFile.Close()

	byteValue, _ := ioutil.ReadAll(xmlFile)
	return byteValue
}

// If err is populated, and ERROR status will be returned.
func buildTradefedResult(logger *log.Logger, resultsPath string) *api.CrosTestResponse {
	logger.Println("building results")
	f := &api.CrosTestResponse{}
	Res := Result{}
	allTestCases := []*api.TestCaseResult{}

	// Read the xmlapi.
	resultsByte := readResults(logger, resultsPath)
	xml.Unmarshal(resultsByte, &Res)

	startTime, _ := strconv.ParseInt(Res.Start, 10, 64)
	logger.Println("Found n module results:", len(Res.Modules))

	for _, module := range Res.Modules {
		nTests, _ := strconv.Atoi(module.Total_Tests)
		runTime, _ := strconv.Atoi(module.Runtime)
		duration := runTime
		if nTests != 0 {
			duration = runTime / nTests
		}

		logger.Println("Parsing Module: ", module.Name)
		// Make a duration of atleast 1 second so the proto isnt empty.
		testDur := int64(duration / 1000)
		if testDur == 0 {
			testDur = 1
		}

		for _, testcase := range module.TestCases {
			for _, test := range testcase.Tests {
				testName := fmt.Sprintf("%s.%s.%s", module.Name, testcase.Name, test.Name)
				allTestCases = append(allTestCases, buildTcResult(test, startTime, testDur, testName))
			}
		}
	}
	f.TestCaseResults = allTestCases
	logger.Println("Found n case results:", len(f.TestCaseResults))

	return f
}

func buildTcResult(test TradeFedTest, startTime int64, duration int64, testName string) *api.TestCaseResult {
	tcResult := new(api.TestCaseResult)

	tcResult.TestHarness = &api.TestHarness{TestHarnessType: &api.TestHarness_Tradefed_{Tradefed: &api.TestHarness_Tradefed{}}}
	tcResult.TestCaseId = &api.TestCase_Id{Value: testName}

	tcResult.StartTime = timestamppb.New(time.Unix(startTime/1000, startTime%1000*1000000))
	tcResult.Duration = &durationpb.Duration{Seconds: duration}

	error := &api.TestCaseResult_Error{}
	if test.Failure.Message != "" {
		error.Message = test.Failure.Message
	}

	switch test.Result {
	case "pass":
		tcResult.Verdict = &api.TestCaseResult_Pass_{Pass: &api.TestCaseResult_Pass{}}
	case "ASSUMPTION_FAILURE":
		tcResult.Verdict = &api.TestCaseResult_Pass_{Pass: &api.TestCaseResult_Pass{}}
		tcResult.Errors = []*api.TestCaseResult_Error{error}
	case "fail":
		tcResult.Verdict = &api.TestCaseResult_Fail_{Fail: &api.TestCaseResult_Fail{}}
		tcResult.Errors = []*api.TestCaseResult_Error{error}
	case "INCOMPLETE":
		tcResult.Verdict = &api.TestCaseResult_Crash_{Crash: &api.TestCaseResult_Crash{}}
		tcResult.Errors = []*api.TestCaseResult_Error{error}
	case "IGNORED":
		tcResult.Verdict = &api.TestCaseResult_Skip_{Skip: &api.TestCaseResult_Skip{}}
	default:
		tcResult.Verdict = &api.TestCaseResult_Fail_{Fail: &api.TestCaseResult_Fail{}}
	}

	return tcResult
}
