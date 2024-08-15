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
	. "go.chromium.org/luci/common/testing/assertions"
)

const AvlInfoFile = "test_data/avl_info.json"

func TestAVLFiles(t *testing.T) {
	t.Parallel()

	emptyLogger := log.New(io.Discard, "", 0)

	Convey("AVLFiles works", t, func() {
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
			"tast.storage.LowPowerStateResidence": "gs://chromeos-image-archive/path/to/test/results/tast/results/tests/storage.LowPowerStateResidence/avl_info.json",
			"tauto.foo.bar":                       "gs://chromeos-image-archive/path/to/test/results/foo.bar/avl_info.json",
		}

		got := avlFiles(emptyLogger, testResult)

		So(got, ShouldResemble, want)
	})

	Convey("Return empty if no test result", t, func() {
		got := avlFiles(emptyLogger, nil)

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

		got := avlFiles(emptyLogger, testResult)

		So(got, ShouldResemble, map[string]string{})
	})
}

func TestAvlInfos(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	test := "storage.LowPowerStateResidence"
	emptyLogger := log.New(io.Discard, "", 0)

	Convey(`AVLInfos works`, t, func() {
		want := map[string]*api.AvlInfo{
			test: {
				AvlPartModel:     "0x0000f5 MMC32G",
				AvlPartFirmware:  "0xa200000000000000",
				AvlComponentType: "storage",
			},
		}
		avlFiles := map[string]string{
			test: AvlInfoFile,
		}

		got := avlInfos(ctx, emptyLogger, avlFiles)

		So(len(got), ShouldEqual, 1)
		So(got[test], ShouldResembleProto, want[test])
	})

	Convey(`Return empty if no avl info file is found`, t, func() {
		want := map[string]*api.AvlInfo{
			test: {},
		}
		avlFiles := map[string]string{
			test: "",
		}

		got := avlInfos(ctx, emptyLogger, avlFiles)

		So(len(got), ShouldEqual, 1)
		So(got[test], ShouldResembleProto, want[test])
	})
}
