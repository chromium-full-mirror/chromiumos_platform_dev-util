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
	"path"
	"strconv"
	"time"

	"go.chromium.org/chromiumos/config/go/test/api"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TradeFedTest struct {
	Result string `xml:"result,attr"`
	Name   string `xml:"name,attr"`
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

func readResults() []byte {
	resultsPath := path.Join(ctsPath, "results/latest/test_result.xml")
	xmlFile, err := os.Open(resultsPath)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Successfully Opened users.xml")
	defer xmlFile.Close()

	byteValue, _ := ioutil.ReadAll(xmlFile)
	return byteValue
}

// If err is populated, and ERROR status will be returned.
func buildTradefedResult(logger *log.Logger) *api.CrosTestResponse {
	f := &api.CrosTestResponse{}
	Res := Result{}
	allTestCases := []*api.TestCaseResult{}

	// Read the xmlapi.
	resultsByte := readResults()
	xml.Unmarshal(resultsByte, &Res)

	startTime, _ := strconv.ParseInt(Res.Start, 10, 64)

	for _, module := range Res.Modules {
		nTests, _ := strconv.Atoi(module.Total_Tests)
		runTime, _ := strconv.Atoi(module.Runtime)
		duration := runTime / nTests

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
	return f
}

func buildTcResult(test TradeFedTest, startTime int64, duration int64, testName string) *api.TestCaseResult {
	tcResult := new(api.TestCaseResult)

	tcResult.TestHarness = &api.TestHarness{TestHarnessType: &api.TestHarness_Tradefed_{Tradefed: &api.TestHarness_Tradefed{}}}
	tcResult.TestCaseId = &api.TestCase_Id{Value: testName}

	// TODO swap it over to the provided int64
	tcResult.StartTime = timestamppb.New(time.Now())
	tcResult.Duration = &durationpb.Duration{Seconds: duration}

	if test.Result == "pass" {
		tcResult.Verdict = &api.TestCaseResult_Pass_{Pass: &api.TestCaseResult_Pass{}}
	} else {
		// TODO, fill these out. I haven't looked up their strings yet.
		tcResult.Verdict = &api.TestCaseResult_Fail_{Fail: &api.TestCaseResult_Fail{}}
	}

	return tcResult
}
