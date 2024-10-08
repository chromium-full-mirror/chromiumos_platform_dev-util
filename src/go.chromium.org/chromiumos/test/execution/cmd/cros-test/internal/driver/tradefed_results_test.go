// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package driver

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	TradefedTestResultFile              = "tradefed_test_data/test_result.xml"
	TradefedTestResultFileModulesNoTest = "tradefed_test_data/test_result_modules_no_test.xml"
)
const (
	FileSelectFileByPattern1 = "/tmp/test-SelectFileByPattern-file1.txt"
	FileSelectFileByPattern2 = "/tmp/test-SelectFileByPattern-file2.txt"
	FileSelectFileByPattern3 = "/tmp/test-SelectFileByPattern-file3.txt"
	FileSelectGlobPattern    = "/tmp/test-Se*-file*.txt"
	FileSelectBadPattern     = "/tmp/mdfvblsakerbakl*"

	EventLogFile  = "tradefed_test_data/event-log-good.txt"
	MetricLogFile = "tradefed_test_data/metric-log.xml"
)

type filePathAndTime struct {
	path    string
	modTime time.Time
}

func createFileWithModTime(t *testing.T, filePath string, mod time.Time) {
	if _, err := os.Create(filePath); err != nil {
		t.Errorf("Can't create temp test file: %s, error: %v", filePath, err)
		return
	}
	if err := os.Chtimes(filePath, mod, mod); err != nil {
		t.Errorf("Can't modify temp test file: %s, error: %v", filePath, err)
	}
}

func TestSelectFileByPattern(t *testing.T) {
	baseTime := time.Now()

	Convey("Select file by pattern", t, func() {
		for _, tc := range []struct {
			name     string
			files    []filePathAndTime
			pattern  string
			selected string
		}{
			{
				name: "Single file by full name",
				files: []filePathAndTime{
					filePathAndTime{path: FileSelectFileByPattern1, modTime: baseTime},
				},
				pattern:  FileSelectFileByPattern1,
				selected: FileSelectFileByPattern1,
			},
			{
				name: "Multiple files by pattern",
				files: []filePathAndTime{
					filePathAndTime{path: FileSelectFileByPattern1, modTime: baseTime.Add(-time.Minute * 30)},
					filePathAndTime{path: FileSelectFileByPattern2, modTime: baseTime},
					filePathAndTime{path: FileSelectFileByPattern3, modTime: baseTime.Add(-time.Minute * 10)},
				},
				pattern:  FileSelectGlobPattern,
				selected: FileSelectFileByPattern2,
			},
			{
				name: "No matching files by pattern",
				files: []filePathAndTime{
					filePathAndTime{path: FileSelectFileByPattern1, modTime: baseTime},
				},
				pattern:  FileSelectBadPattern,
				selected: "",
			},
		} {
			Convey(fmt.Sprintf("%s, pattern: %s", tc.name, tc.pattern), func() {
				for _, file := range tc.files {
					createFileWithModTime(t, file.path, file.modTime)
					defer os.Remove(file.path)
				}
				got, err := selectFileByPattern(tc.pattern)

				if len(tc.selected) > 0 {
					So(err, ShouldBeNil)
				} else {
					So(err, ShouldNotBeNil)
				}
				So(got, ShouldEqual, tc.selected)
			})
		}
	})
}

func TestParseEventLogReport(t *testing.T) {
	Convey("Parse event log text file, extracting test info", t, func() {
		logger := log.New(io.Discard, "", 0)

		want := map[string]EventLogTestInfo{
			"com.android.desktop.bootperf.dts.BootPerfTest#bootPerf": EventLogTestInfo{
				TestName:   "com.android.desktop.bootperf.dts.BootPerfTest#bootPerf",
				ModuleName: "DtsBootPerfTest",
				Status:     "PASSED",
			},
			"com.android.desktop.rarperf.dts.RarPerfTestHostWrapper#rarPerfTest": EventLogTestInfo{
				TestName:   "com.android.desktop.rarperf.dts.RarPerfTestHostWrapper#rarPerfTest",
				ModuleName: "DtsRarPerfHostTestCases",
				Status:     "PASSED",
			},
			"com.android.desktop.sample.dts.SampleDeviceReportLogTest#testMultiplication": EventLogTestInfo{
				TestName:   "com.android.desktop.sample.dts.SampleDeviceReportLogTest#testMultiplication",
				ModuleName: "DtsSampleDeviceTestCases",
				Status:     "PASSED",
			},
			"com.android.desktop.sample.dts.SampleDeviceReportLogTest#testCountUp": EventLogTestInfo{
				TestName:   "com.android.desktop.sample.dts.SampleDeviceReportLogTest#testCountUp",
				ModuleName: "DtsSampleDeviceTestCases",
				Status:     "PASSED",
			},
			"com.android.desktop.sample.dts.SampleDeviceReportLogTest#testCountDown": EventLogTestInfo{
				TestName:   "com.android.desktop.sample.dts.SampleDeviceReportLogTest#testCountDown",
				ModuleName: "DtsSampleDeviceTestCases",
				Status:     "PASSED",
			},
			"com.google.android.power.dts.SuspendStressTest#suspendStress": EventLogTestInfo{
				TestName:   "com.google.android.power.dts.SuspendStressTest#suspendStress",
				ModuleName: "DtsSuspendStressTest",
				Status:     "ASSUMPTION_FAILURE",
			},
		}

		got := parseEventLogReport(logger, EventLogFile)

		So(got, ShouldNotBeNil)
		So(got, ShouldHaveLength, 6)
		So(got, ShouldResemble, want)
	})
}

func TestParseResultFile(t *testing.T) {
	Convey("Parse result XML file, extracting test cases", t, func() {
		logger := log.New(io.Discard, "", 0)

		// testHarness := &api.TestHarness{
		// 	TestHarnessType: &api.TestHarness_Tradefed_{Tradefed: &api.TestHarness_Tradefed{}},
		// }

		// failureMessage := "java.lang.AssertionError\r"
		// startTimestamp := int64(1726957857000)

		want := TestSuite{
			XMLName: xml.Name{Local: "testsuite"},
			Name:    "cts-test-module-x86", Tests: 6, Failures: 0, Time: 178403, Timestamp: "2024-09-21T22:30:57",
			TestCases: []TestCase{TestCase{
				XMLName:   xml.Name{Local: "testcase"},
				Name:      "rarPerfTest",
				ClassName: "com.android.desktop.rarperf.dts.RarPerfTestHostWrapper",
				Time:      94522,
			}, TestCase{
				XMLName:   xml.Name{Local: "testcase"},
				Name:      "testCollectDeviceInfo",
				ClassName: "com.android.desktop.sample.dts.SampleHostDeviceInfo",
				Time:      1788,
			}, TestCase{
				XMLName:   xml.Name{Local: "testcase"},
				Name:      "testRunDeviceTestsFailsInstant",
				ClassName: "com.android.desktop.sample.dts.SampleHostJUnit4DeviceTest",
				Time:      2627,
			}, TestCase{
				XMLName:            xml.Name{Local: "testcase"},
				Name:               "testRunDeviceTestsAssumeFailsInstant",
				ClassName:          "com.android.desktop.sample.dts.SampleHostJUnit4DeviceTest",
				Time:               2684,
				AsssumptionFailure: "org.junit.AssumptionViolatedException: org.junit.AssumptionViolatedException:\n",
			}, TestCase{
				XMLName:            xml.Name{Local: "testcase"},
				Name:               "testRunDeviceTestsAssumeFailsFull",
				ClassName:          "com.android.desktop.sample.dts.SampleHostJUnit4DeviceTest",
				Time:               2858,
				AsssumptionFailure: "org.junit.AssumptionViolatedException: org.junit.AssumptionViolatedException:\n",
			}, TestCase{
				XMLName:   xml.Name{Local: "testcase"},
				Name:      "testMetrics",
				ClassName: "com.android.desktop.sample.dts.SampleHostJUnit4Test",
				Time:      144,
			}}}

		got := TestSuite{}
		err := parseResultFile(logger, MetricLogFile, &got)
		So(err, ShouldBeNil)
		So(got, ShouldResemble, want)
	})
}

// TODO: Re-enable below test
// func TestBuildTradefedResult(t *testing.T) {
// 	t.Parallel()

// 	Convey("Build TradeFed results from XML", t, func() {
// 		logger := log.New(io.Discard, "", 0)
// 		failureMessage := "java.lang.AssertionError\r"
// 		startTimestamp := int64(1724450705582)
// 		startTime := timestamppb.New(time.Unix(startTimestamp/1000, startTimestamp%1000*1000000))
// 		numOfTests := 2
// 		duration := &durationpb.Duration{Seconds: int64(2254 / 1000 / numOfTests)}
// 		testHarness := &api.TestHarness{TestHarnessType: &api.TestHarness_Tradefed_{Tradefed: &api.TestHarness_Tradefed{}}}
// 		want := &api.CrosTestResponse{
// 			TestCaseResults: []*api.TestCaseResult{
// 				{
// 					TestCaseId:  &api.TestCase_Id{Value: "CtsJvmtiRunTest1976HostTestCases.android.jvmti.cts.JvmtiHostTest1976.testJvmti"},
// 					TestHarness: testHarness,
// 					Verdict:     &api.TestCaseResult_Pass_{Pass: &api.TestCaseResult_Pass{}},
// 					StartTime:   startTime,
// 					Duration:    duration,
// 				},
// 				{
// 					TestCaseId:  &api.TestCase_Id{Value: "CtsJvmtiRunTest1976HostTestCases.android.jvmti.cts.JvmtiHostTest1976.testAssumptionFail"},
// 					TestHarness: testHarness,
// 					Verdict:     &api.TestCaseResult_Pass_{Pass: &api.TestCaseResult_Pass{}},
// 					Errors: []*api.TestCaseResult_Error{
// 						{
// 							Message: failureMessage,
// 						},
// 					},
// 					StartTime: startTime,
// 					Duration:  duration,
// 				},
// 				{
// 					TestCaseId:  &api.TestCase_Id{Value: "CtsJvmtiRunTest1976HostTestCases.android.jvmti.cts.JvmtiHostTest1976.testIncomplate"},
// 					TestHarness: testHarness,
// 					Verdict:     &api.TestCaseResult_Crash_{Crash: &api.TestCaseResult_Crash{}},
// 					Errors: []*api.TestCaseResult_Error{
// 						{
// 							Message: failureMessage,
// 						},
// 					},
// 					StartTime: startTime,
// 					Duration:  duration,
// 				},
// 				{
// 					TestCaseId:  &api.TestCase_Id{Value: "CtsJvmtiRunTest1976HostTestCases.android.jvmti.cts.JvmtiHostTest1976.testIgnored"},
// 					TestHarness: testHarness,
// 					Verdict:     &api.TestCaseResult_Skip_{Skip: &api.TestCaseResult_Skip{}},
// 					StartTime:   startTime,
// 					Duration:    duration,
// 				},
// 				{
// 					TestCaseId:  &api.TestCase_Id{Value: "CtsJvmtiRunTest1976HostTestCases.android.jvmti.cts.JvmtiHostTest1976.testUnknown"},
// 					TestHarness: testHarness,
// 					Verdict:     &api.TestCaseResult_Fail_{Fail: &api.TestCaseResult_Fail{}},
// 					StartTime:   startTime,
// 					Duration:    duration,
// 				},
// 				{
// 					TestCaseId:  &api.TestCase_Id{Value: "CtsJvmtiRunTest1976HostTestCases.android.mediav2.cts.CodecDecoderSurfaceTest.testFlushNative"},
// 					TestHarness: testHarness,
// 					Verdict:     &api.TestCaseResult_Fail_{Fail: &api.TestCaseResult_Fail{}},
// 					Errors: []*api.TestCaseResult_Error{
// 						{
// 							Message: failureMessage,
// 						},
// 					},
// 					StartTime: startTime,
// 					Duration:  duration,
// 				},
// 			},
// 		}

// 		got := buildTradefedResult(logger, TradefedTestResultFile)

// 		So(got.TestCaseResults, ShouldResemble, want.TestCaseResults)
// 	})
// }

func Test_BuildTFResult(t *testing.T) {
	R := Result{}
	testType := "CTS"
	moduleName := "tradefed" + "." + testType + "." + "CtsJvmtiRunTest1976HostTestCases"
	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	err := parseResultFile(logger, TradefedTestResultFileModulesNoTest, &R)
	if err != nil {
		t.Logf("No error expected")
	}
	allTestCases := generateTestCaseResult(logger, testType, R, nil)
	givenTestCases := generateGivenTestCases(logger, testType, R, nil)
	if allTestCases[0].TestCaseId.Value != moduleName {
		t.Errorf("Expected %s, found %s", moduleName, allTestCases[0].TestCaseId.Value)
	}
	if givenTestCases[0].GetParentTest() != moduleName {
		t.Errorf("Expected %s, found %s", moduleName, givenTestCases[0].GetParentTest())
	}
	if _, ok := allTestCases[0].Verdict.(*api.TestCaseResult_Crash_); !ok {
		t.Errorf("Expected verdict CRASH")
	}
	if givenTestCases[0].GetParentTest() != moduleName {
		t.Errorf("Expected %s, found %s", moduleName, givenTestCases[0].GetParentTest())
	}
	if _, ok := givenTestCases[0].ChildTestCaseResults[0].Verdict.(*api.TestCaseResult_Crash_); !ok {
		t.Errorf("Expected verdict CRASH")
	}

}
func TestBuildTcResult(t *testing.T) {
	t.Parallel()

	Convey("Build test results for different statuses", t, func() {
		startTime, _ := time.Parse(time.RFC3339, "2024-01-18T00:12:34Z")
		endTime, _ := time.Parse(time.RFC3339, "2024-01-18T00:13:34Z")
		duration := endTime.Sub(startTime)
		failureMessage := "Failed for some reason"
		testName := "tradefed.cts.CtsSampleTestCase"

		testCaseID := &api.TestCase_Id{Value: testName}
		testHarness := &api.TestHarness{TestHarnessType: &api.TestHarness_Tradefed_{Tradefed: &api.TestHarness_Tradefed{}}}
		wantStartTime := timestamppb.New(startTime)
		wantDuration := &durationpb.Duration{Seconds: int64(duration.Seconds())}

		for _, tc := range []struct {
			name         string
			errorMessage string
			status       string
			want         *api.TestCaseResult
		}{
			{
				name:         "Pass test",
				errorMessage: "",
				status:       "PASSED",
				want: &api.TestCaseResult{
					TestCaseId:  testCaseID,
					TestHarness: testHarness,
					Verdict:     &api.TestCaseResult_Pass_{Pass: &api.TestCaseResult_Pass{}},
					StartTime:   wantStartTime,
					Duration:    wantDuration,
				},
			},
			{
				name:         "Assumption failure test",
				errorMessage: failureMessage,
				status:       "ASSUMPTION_FAILURE",
				want: &api.TestCaseResult{
					TestCaseId:  testCaseID,
					TestHarness: testHarness,
					Verdict:     &api.TestCaseResult_Pass_{Pass: &api.TestCaseResult_Pass{}},
					Errors:      []*api.TestCaseResult_Error{{Message: failureMessage}},
					StartTime:   wantStartTime,
					Duration:    wantDuration,
				},
			},
			{
				name:         "Fail test",
				errorMessage: failureMessage,
				status:       "FAILED",
				want: &api.TestCaseResult{
					TestCaseId:  testCaseID,
					TestHarness: testHarness,
					Verdict:     &api.TestCaseResult_Fail_{Fail: &api.TestCaseResult_Fail{}},
					Errors:      []*api.TestCaseResult_Error{{Message: failureMessage}},
					StartTime:   wantStartTime,
					Duration:    wantDuration,
				},
			},
			{
				name:         "Incomplete test",
				errorMessage: failureMessage,
				status:       "INCOMPLETE",
				want: &api.TestCaseResult{
					TestCaseId:  testCaseID,
					TestHarness: testHarness,
					Verdict:     &api.TestCaseResult_Crash_{Crash: &api.TestCaseResult_Crash{}},
					Errors:      []*api.TestCaseResult_Error{{Message: failureMessage}},
					StartTime:   wantStartTime,
					Duration:    wantDuration,
				},
			},
			{
				name:         "Skipped test",
				errorMessage: failureMessage,
				status:       "SKIPPED",
				want: &api.TestCaseResult{
					TestCaseId:  testCaseID,
					TestHarness: testHarness,
					Verdict:     &api.TestCaseResult_Skip_{Skip: &api.TestCaseResult_Skip{}},
					Errors:      []*api.TestCaseResult_Error{{Message: failureMessage}},
					StartTime:   wantStartTime,
					Duration:    wantDuration,
				},
			},
			{
				name:         "Ignore test",
				errorMessage: failureMessage,
				status:       "IGNORED",
				want: &api.TestCaseResult{
					TestCaseId:  testCaseID,
					TestHarness: testHarness,
					Verdict:     &api.TestCaseResult_Skip_{Skip: &api.TestCaseResult_Skip{}},
					Errors:      []*api.TestCaseResult_Error{{Message: failureMessage}},
					StartTime:   wantStartTime,
					Duration:    wantDuration,
				},
			},
		} {
			Convey(tc.name, func() {
				got := buildTcResult(testName, tc.status, startTime, int64(duration.Seconds()), tc.errorMessage)
				So(got, ShouldResemble, tc.want)
			})
		}
	})
}
