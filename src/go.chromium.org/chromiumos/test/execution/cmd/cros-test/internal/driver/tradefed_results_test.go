// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package driver

import (
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
	TradefedTestResultModuleError       = "tradefed_test_data/test_result_module_error.xml"

	FileSelectFileByPattern1 = "/tmp/test-SelectFileByPattern-file1.txt"
	FileSelectFileByPattern2 = "/tmp/test-SelectFileByPattern-file2.txt"
	FileSelectFileByPattern3 = "/tmp/test-SelectFileByPattern-file3.txt"
	FileSelectGlobPattern    = "/tmp/test-Se*-file*.txt"
	FileSelectBadPattern     = "/tmp/mdfvblsakerbakl*"
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

func getTestCaseResults(prefix string) []*api.TestCaseResult {
	failureMessage := "java.lang.AssertionError\r"
	startTimestamp := int64(1724450705582)
	startTime := timestamppb.New(time.Unix(startTimestamp/1000, startTimestamp%1000*1000000))
	numOfTests := 2
	duration := &durationpb.Duration{Seconds: int64(2254 / 1000 / numOfTests)}
	testHarness := &api.TestHarness{TestHarnessType: &api.TestHarness_Tradefed_{Tradefed: &api.TestHarness_Tradefed{}}}

	return []*api.TestCaseResult{
		{
			TestCaseId:  &api.TestCase_Id{Value: prefix + "android.jvmti.cts.JvmtiHostTest1976#testJvmti"},
			TestHarness: testHarness,
			Verdict:     &api.TestCaseResult_Pass_{Pass: &api.TestCaseResult_Pass{}},
			StartTime:   startTime,
			Duration:    duration,
		},
		{
			TestCaseId:  &api.TestCase_Id{Value: prefix + "android.jvmti.cts.JvmtiHostTest1976#testAssumptionFail"},
			TestHarness: testHarness,
			Verdict:     &api.TestCaseResult_Pass_{Pass: &api.TestCaseResult_Pass{}},
			Errors:      []*api.TestCaseResult_Error{{Message: failureMessage}},
			StartTime:   startTime,
			Duration:    duration,
		},
		{
			TestCaseId:  &api.TestCase_Id{Value: prefix + "android.jvmti.cts.JvmtiHostTest1976#testIncomplate"},
			TestHarness: testHarness,
			Verdict:     &api.TestCaseResult_Crash_{Crash: &api.TestCaseResult_Crash{}},
			Errors:      []*api.TestCaseResult_Error{{Message: failureMessage}},
			StartTime:   startTime,
			Duration:    duration,
		},
		{
			TestCaseId:  &api.TestCase_Id{Value: prefix + "android.jvmti.cts.JvmtiHostTest1976#testIgnored"},
			TestHarness: testHarness,
			Verdict:     &api.TestCaseResult_Skip_{Skip: &api.TestCaseResult_Skip{}},
			Errors:      []*api.TestCaseResult_Error{{Message: ""}},
			StartTime:   startTime,
			Duration:    duration,
		},
		{
			TestCaseId:  &api.TestCase_Id{Value: prefix + "android.jvmti.cts.JvmtiHostTest1976#testUnknown"},
			TestHarness: testHarness,
			Verdict:     &api.TestCaseResult_Fail_{Fail: &api.TestCaseResult_Fail{}},
			Errors:      []*api.TestCaseResult_Error{{Message: ""}},
			StartTime:   startTime,
			Duration:    duration,
		},
		{
			TestCaseId:  &api.TestCase_Id{Value: prefix + "android.mediav2.cts.CodecDecoderSurfaceTest#testFlushNative"},
			TestHarness: testHarness,
			Verdict:     &api.TestCaseResult_Fail_{Fail: &api.TestCaseResult_Fail{}},
			Errors:      []*api.TestCaseResult_Error{{Message: failureMessage}},
			StartTime:   startTime,
			Duration:    duration,
		},
	}
}

func TestParseCompatibilityXmlResults(t *testing.T) {
	t.Parallel()

	Convey("Build TradeFed results from XML", t, func() {
		logger := log.New(io.Discard, "", 0)
		testType := "cts"
		want := &api.CrosTestResponse{
			TestCaseResults: getTestCaseResults("tradefed.cts.CtsJvmtiRunTest1976HostTestCases#"),
			GivenTestResults: []*api.CrosTestResponse_GivenTestResult{
				{
					ParentTest:           "tradefed.cts.CtsJvmtiRunTest1976HostTestCases",
					ChildTestCaseResults: getTestCaseResults(""),
				},
			},
		}

		R := Result{}
		err := parseResultFile(logger, TradefedTestResultFile, &R)
		So(err, ShouldBeNil)

		allTestCases, givenTestCases := generateTestCaseResult(logger, testType, R, nil)

		So(allTestCases, ShouldResemble, want.TestCaseResults)
		So(givenTestCases, ShouldResemble, want.GivenTestResults)
	})

	Convey("Build TradeFed results from empty XML", t, func() {
		logger := log.New(io.Discard, "", 0)
		R := Result{}
		testType := "cts"
		moduleName := fmt.Sprintf("tradefed.%s.CtsJvmtiRunTest1976HostTestCases", testType)
		passVerdict := &api.TestCaseResult_Pass_{Pass: &api.TestCaseResult_Pass{}}

		err := parseResultFile(logger, TradefedTestResultFileModulesNoTest, &R)
		So(err, ShouldBeNil)

		allTestCases, givenTestCases := generateTestCaseResult(logger, testType, R, nil)

		So(allTestCases[0].TestCaseId.Value, ShouldEqual, moduleName)
		So(allTestCases[0].Verdict, ShouldResemble, passVerdict)
		So(givenTestCases[0].GetParentTest(), ShouldEqual, moduleName)
		So(givenTestCases[0].ChildTestCaseResults[0].Verdict, ShouldResemble, passVerdict)
	})

	Convey("Build TradeFed results from XML with module error", t, func() {
		logger := log.New(io.Discard, "", 0)
		R := Result{}
		testType := "cts"
		moduleName := fmt.Sprintf("tradefed.%s.CtsWrapWrapNoDebugTestCases", testType)
		failVerdict := &api.TestCaseResult_Fail_{Fail: &api.TestCaseResult_Fail{}}

		err := parseResultFile(logger, TradefedTestResultModuleError, &R)
		So(err, ShouldBeNil)

		allTestCases, givenTestCases := generateTestCaseResult(logger, testType, R, nil)

		So(allTestCases[0].TestCaseId.Value, ShouldEqual, moduleName)
		So(allTestCases[0].Verdict, ShouldResemble, failVerdict)
		So(givenTestCases[0].GetParentTest(), ShouldEqual, moduleName)
		So(givenTestCases[0].ChildTestCaseResults[0].Verdict, ShouldResemble, failVerdict)
	})
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
