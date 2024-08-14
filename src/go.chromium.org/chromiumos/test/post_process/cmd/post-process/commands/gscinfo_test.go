// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package commands

import (
	"context"
	"io"
	"log"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	_go "go.chromium.org/chromiumos/config/go"
	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/config/go/test/artifact"
	"google.golang.org/protobuf/types/known/anypb"
)

const GSCInfoJSONFile = "test_data/gsc_info.json"

func TestGSCFiles(t *testing.T) {
	t.Parallel()

	emptyLogger := log.New(io.Discard, "", 0)

	Convey("GSCInfo works", t, func() {
		testResult := &artifact.TestResult{
			TestRuns: []*artifact.TestRun{
				{
					TestCaseInfo: &artifact.TestCaseInfo{
						TestCaseResult: &api.TestCaseResult{
							TestCaseId: &api.TestCase_Id{
								Value: "tast.storage.LowPowerStateResidence",
							},
							ResultDirPath: &_go.StoragePath{
								HostType: _go.StoragePath_LOCAL,
								Path:     "gs://chromeos-image-archive/path/to/test/results",
							},
						},
					},
				},
				{
					TestCaseInfo: &artifact.TestCaseInfo{
						TestCaseResult: &api.TestCaseResult{
							TestCaseId: &api.TestCase_Id{
								Value: "tauto.foo.bar",
							},
							ResultDirPath: &_go.StoragePath{
								HostType: _go.StoragePath_LOCAL,
								Path:     "gs://chromeos-image-archive/path/to/test/results",
							},
						},
					},
				},
			},
		}
		want := map[string]string{
			"tast.storage.LowPowerStateResidence": "gs://chromeos-image-archive/path/to/test/results/tast/results/tests/storage.LowPowerStateResidence/gsc_info.json",
			"tauto.foo.bar":                       "gs://chromeos-image-archive/path/to/test/results/foo.bar/gsc_info.json",
		}

		got := gscFiles(emptyLogger, testResult)

		So(got, ShouldResemble, want)
	})

	Convey("Return empty if no test result", t, func() {
		got := gscFiles(emptyLogger, nil)

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

		got := gscFiles(emptyLogger, testResult)

		So(got, ShouldResemble, map[string]string{})
	})
}

func TestGSCInfos(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	test := "storage.LowPowerStateResidence"
	emptyLogger := log.New(io.Discard, "", 0)

	Convey(`GSCInfos works`, t, func() {
		wantGSCInfo, _ := anypb.New(&artifact.GscInfo{
			GscRoVersion:   "0.0.59",
			GscRwVersion:   "0.26.112",
			GscRwBranch:    "tot:v0.0",
			GscRwRev:       "1408",
			GscRwSha:       "-ac00db8d",
			GscBuildurl:    "gs://chromeos-image-archive/firmware-ti50-postsubmit/R129-15969.0.0-102267-8741384054848642993",
			GscTestbedType: "gsc_dt_shield",
			GscCcdSerial:   "1482101a-4c2ac261",
		})
		want := map[string]*anypb.Any{
			test: wantGSCInfo,
		}
		gscFiles := map[string]string{
			test: GSCInfoJSONFile,
		}

		got := gscInfos(ctx, emptyLogger, gscFiles)

		So(got, ShouldResemble, want)
	})

	Convey(`Return empty if no GSC devboard info file is found`, t, func() {
		wantGSCInfo, _ := anypb.New(&artifact.GscInfo{})
		want := map[string]*anypb.Any{
			test: wantGSCInfo,
		}
		gscFiles := map[string]string{
			test: "",
		}

		got := gscInfos(ctx, emptyLogger, gscFiles)

		So(got, ShouldResemble, want)
	})
}
