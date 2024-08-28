// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package common

import (
	"io"
	"log"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	_go "go.chromium.org/chromiumos/config/go"
	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/config/go/test/artifact"
)

func TestTestLevelFiles(t *testing.T) {
	t.Parallel()

	emptyLogger := log.New(io.Discard, "", 0)
	gscInfoFileName := "gsc_info.json"

	Convey("GSCInfo works", t, func() {
		tastFirstClassTest := "tast.storage.FirstClass"
		tastSecondClassTest := "tast.storage.SecondClass"
		tautoTest := "tauto.foo.bar"
		tastResultDirPath := "/tmp/test/results/tast/tests/" + tastFirstClassTest
		tautoResultDirPath := "/tmp/test/tauto/results-1-storage_testing_v3_part_perf"
		testResult := &artifact.TestResult{
			TestRuns: []*artifact.TestRun{
				// For Tast first class test
				{
					TestCaseInfo: &artifact.TestCaseInfo{
						TestCaseResult: &api.TestCaseResult{
							TestCaseId: &api.TestCase_Id{
								Value: tastFirstClassTest,
							},
							ResultDirPath: &_go.StoragePath{
								HostType: _go.StoragePath_LOCAL,
								Path:     tastResultDirPath,
							},
						},
					},
				},
				// For Tast second class test wrapped by Tauto test suite
				{
					TestCaseInfo: &artifact.TestCaseInfo{
						TestCaseResult: &api.TestCaseResult{
							TestCaseId: &api.TestCase_Id{
								Value: tastSecondClassTest,
							},
							ResultDirPath: &_go.StoragePath{
								HostType: _go.StoragePath_LOCAL,
								Path:     tautoResultDirPath,
							},
						},
					},
				},
				// For Tauto test
				{
					TestCaseInfo: &artifact.TestCaseInfo{
						TestCaseResult: &api.TestCaseResult{
							TestCaseId: &api.TestCase_Id{
								Value: tautoTest,
							},
							ResultDirPath: &_go.StoragePath{
								HostType: _go.StoragePath_LOCAL,
								Path:     tautoResultDirPath,
							},
						},
					},
				},
			},
		}
		want := map[string]string{
			tastFirstClassTest:  tastResultDirPath + "/" + gscInfoFileName,
			tastSecondClassTest: tautoResultDirPath + "/" + TastTestsDir + "/storage.SecondClass/" + gscInfoFileName,
			tautoTest:           tautoResultDirPath + "/foo.bar/" + gscInfoFileName,
		}

		got := TestLevelFiles(emptyLogger, testResult, gscInfoFileName)

		So(got, ShouldResemble, want)
	})

	Convey("Return empty if no test result", t, func() {
		got := TestLevelFiles(emptyLogger, nil, gscInfoFileName)

		So(got, ShouldResemble, map[string]string{})
	})

	Convey("Return empty if no test case result", t, func() {
		testResult := &artifact.TestResult{
			TestRuns: []*artifact.TestRun{
				{
					TestCaseInfo: &artifact.TestCaseInfo{},
				},
			},
		}

		got := TestLevelFiles(emptyLogger, testResult, gscInfoFileName)

		So(got, ShouldResemble, map[string]string{})
	})
}
