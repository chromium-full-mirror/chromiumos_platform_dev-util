// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package driver

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/config/go/test/api"
)

func createTestFile(dir, filename string, content string) error {
	filePath := filepath.Join(dir, filename)
	return os.WriteFile(filePath, []byte(content), 0644)
}

func TestMoveArtifacts(t *testing.T) {
	t.Parallel()

	Convey("Test moveArtifacts", t, func() {
		tmpDir, err := os.MkdirTemp("", "artifacts")
		So(err, ShouldBeNil)
		defer os.RemoveAll(tmpDir)

		testCases := []struct {
			name            string
			test            *api.TestCaseResult
			createArtifacts func(testDir string) error
			wantFileExist   []string
		}{
			{
				name: "Move artifacts for passed test",
				test: &api.TestCaseResult{
					TestCaseId: &api.TestCase_Id{
						Value: "test.case.one",
					},
					Verdict: &api.TestCaseResult_Pass_{},
				},
				createArtifacts: func(testDir string) error {
					return createTestFile(testDir, "test.log", "test log")
				},
				wantFileExist: []string{
					"test.case.one/test.log",
				},
			},
			{
				name: "Move artifacts for failed test",
				test: &api.TestCaseResult{
					TestCaseId: &api.TestCase_Id{
						Value: "test.case.two",
					},
					Verdict: &api.TestCaseResult_Fail_{},
				},
				createArtifacts: func(testDir string) error {
					return createTestFile(testDir, "test.png", "test image")
				},
				wantFileExist: []string{
					"test.case.two/test.png",
				},
			},
		}

		for _, tc := range testCases {
			Convey(tc.name, func() {
				testDir := filepath.Join(tmpDir, tc.test.GetTestCaseId().GetValue())
				So(os.MkdirAll(testDir, 0755), ShouldBeNil)
				So(tc.createArtifacts(testDir), ShouldBeNil)

				td := &TradefedDriver{
					logger: log.New(io.Discard, "", 0),
				}
				resultsPath := path.Join(fmt.Sprintf(testPath, testType), tradefedTestResultFile)
				td.moveArtifacts(resultsPath, tmpDir)

				for _, file := range tc.wantFileExist {
					filePath := filepath.Join(tmpDir, file)
					_, err := os.Stat(filePath)
					So(err, ShouldBeNil)
				}
			})
		}
	})
}
