// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package driver

import (
	"io"
	"log"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const TradefedTestResultFile = "tradefed_test_data/test_result.xml"

func TestBuildTradefedResult(t *testing.T) {
	t.Parallel()

	Convey("Build TradeFed results from XML", t, func() {
		logger := log.New(io.Discard, "", 0)
		failureMessage := "java.lang.AssertionError\r"
		startTimestamp := int64(1724450705582)
		startTime := timestamppb.New(time.Unix(startTimestamp/1000, startTimestamp%1000*1000000))
		numOfTests := 2
		duration := &durationpb.Duration{Seconds: int64(2254 / 1000 / numOfTests)}
		testHarness := &api.TestHarness{TestHarnessType: &api.TestHarness_Tradefed_{Tradefed: &api.TestHarness_Tradefed{}}}
		want := &api.CrosTestResponse{
			TestCaseResults: []*api.TestCaseResult{
				{
					TestCaseId:  &api.TestCase_Id{Value: "CtsJvmtiRunTest1976HostTestCases.android.jvmti.cts.JvmtiHostTest1976.testJvmti"},
					TestHarness: testHarness,
					Verdict:     &api.TestCaseResult_Pass_{Pass: &api.TestCaseResult_Pass{}},
					StartTime:   startTime,
					Duration:    duration,
				},
				{
					TestCaseId:  &api.TestCase_Id{Value: "CtsJvmtiRunTest1976HostTestCases.android.jvmti.cts.JvmtiHostTest1976.testAssumptionFail"},
					TestHarness: testHarness,
					Verdict:     &api.TestCaseResult_Pass_{Pass: &api.TestCaseResult_Pass{}},
					Errors: []*api.TestCaseResult_Error{
						{
							Message: failureMessage,
						},
					},
					StartTime: startTime,
					Duration:  duration,
				},
				{
					TestCaseId:  &api.TestCase_Id{Value: "CtsJvmtiRunTest1976HostTestCases.android.jvmti.cts.JvmtiHostTest1976.testIncomplate"},
					TestHarness: testHarness,
					Verdict:     &api.TestCaseResult_Crash_{Crash: &api.TestCaseResult_Crash{}},
					Errors: []*api.TestCaseResult_Error{
						{
							Message: failureMessage,
						},
					},
					StartTime: startTime,
					Duration:  duration,
				},
				{
					TestCaseId:  &api.TestCase_Id{Value: "CtsJvmtiRunTest1976HostTestCases.android.jvmti.cts.JvmtiHostTest1976.testIgnored"},
					TestHarness: testHarness,
					Verdict:     &api.TestCaseResult_Skip_{Skip: &api.TestCaseResult_Skip{}},
					StartTime:   startTime,
					Duration:    duration,
				},
				{
					TestCaseId:  &api.TestCase_Id{Value: "CtsJvmtiRunTest1976HostTestCases.android.jvmti.cts.JvmtiHostTest1976.testUnknown"},
					TestHarness: testHarness,
					Verdict:     &api.TestCaseResult_Fail_{Fail: &api.TestCaseResult_Fail{}},
					StartTime:   startTime,
					Duration:    duration,
				},
				{
					TestCaseId:  &api.TestCase_Id{Value: "CtsJvmtiRunTest1976HostTestCases.android.mediav2.cts.CodecDecoderSurfaceTest.testFlushNative"},
					TestHarness: testHarness,
					Verdict:     &api.TestCaseResult_Fail_{Fail: &api.TestCaseResult_Fail{}},
					Errors: []*api.TestCaseResult_Error{
						{
							Message: failureMessage,
						},
					},
					StartTime: startTime,
					Duration:  duration,
				},
			},
		}

		got := buildTradefedResult(logger, TradefedTestResultFile)

		So(got.TestCaseResults, ShouldResemble, want.TestCaseResults)
	})
}

func TestBuildTcResult(t *testing.T) {
	t.Parallel()

	Convey("Build test results for different statuses", t, func() {
		startTime, _ := time.Parse(time.RFC3339, "2024-01-18T00:12:34Z")
		endTime, _ := time.Parse(time.RFC3339, "2024-01-18T00:13:34Z")
		duration := endTime.Sub(startTime)
		testName := "cts-test"
		failureMessage := "Failed for some reason"

		testCaseID := &api.TestCase_Id{Value: testName}
		testHarness := &api.TestHarness{TestHarnessType: &api.TestHarness_Tradefed_{Tradefed: &api.TestHarness_Tradefed{}}}
		wantStartTime := timestamppb.New(startTime)
		wantDuration := &durationpb.Duration{Seconds: int64(duration.Seconds())}

		for _, tc := range []struct {
			name string
			test TradeFedTest
			want *api.TestCaseResult
		}{
			{
				name: "Pass test",
				test: TradeFedTest{
					Result: "pass",
					Name:   testName,
				},
				want: &api.TestCaseResult{
					TestCaseId:  testCaseID,
					TestHarness: testHarness,
					Verdict:     &api.TestCaseResult_Pass_{Pass: &api.TestCaseResult_Pass{}},
					StartTime:   wantStartTime,
					Duration:    wantDuration,
				},
			},
			{
				name: "Assumption failure test",
				test: TradeFedTest{
					Result: "ASSUMPTION_FAILURE",
					Name:   testName,
					Failure: TradeFedTestFailure{
						Message: failureMessage,
					},
				},
				want: &api.TestCaseResult{
					TestCaseId:  testCaseID,
					TestHarness: testHarness,
					Verdict:     &api.TestCaseResult_Pass_{Pass: &api.TestCaseResult_Pass{}},
					Errors: []*api.TestCaseResult_Error{
						{
							Message: failureMessage,
						},
					},
					StartTime: wantStartTime,
					Duration:  wantDuration,
				},
			},
			{
				name: "Fail test",
				test: TradeFedTest{
					Result: "fail",
					Name:   testName,
					Failure: TradeFedTestFailure{
						Message: failureMessage,
					},
				},
				want: &api.TestCaseResult{
					TestCaseId:  testCaseID,
					TestHarness: testHarness,
					Verdict:     &api.TestCaseResult_Fail_{Fail: &api.TestCaseResult_Fail{}},
					Errors: []*api.TestCaseResult_Error{
						{
							Message: failureMessage,
						},
					},
					StartTime: wantStartTime,
					Duration:  wantDuration,
				},
			},
			{
				name: "Incomplete test",
				test: TradeFedTest{
					Result: "INCOMPLETE",
					Name:   testName,
					Failure: TradeFedTestFailure{
						Message: failureMessage,
					},
				},
				want: &api.TestCaseResult{
					TestCaseId:  testCaseID,
					TestHarness: testHarness,
					Verdict:     &api.TestCaseResult_Crash_{Crash: &api.TestCaseResult_Crash{}},
					Errors: []*api.TestCaseResult_Error{
						{
							Message: failureMessage,
						},
					},
					StartTime: wantStartTime,
					Duration:  wantDuration,
				},
			},
			{
				name: "Ignore test",
				test: TradeFedTest{
					Result: "IGNORED",
					Name:   testName,
				},
				want: &api.TestCaseResult{
					TestCaseId:  testCaseID,
					TestHarness: testHarness,
					Verdict:     &api.TestCaseResult_Skip_{Skip: &api.TestCaseResult_Skip{}},
					StartTime:   wantStartTime,
					Duration:    wantDuration,
				},
			},
		} {
			Convey(tc.name, func() {
				got := buildTcResult(tc.test, int64(startTime.UnixMilli()), int64(duration.Seconds()), testName)
				So(got, ShouldResemble, tc.want)
			})
		}
	})
}
