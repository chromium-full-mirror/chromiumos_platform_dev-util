// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package driver

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
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
			createArtifacts func(testDir string) error
			artifactPattern string
			wantFileExist   []string
		}{
			{
				name: "Move single artifact by name",
				createArtifacts: func(testDir string) error {
					return createTestFile(testDir, "test.log", "test log")
				},
				artifactPattern: filepath.Join(tmpDir, "test.log"),
				wantFileExist:   []string{"test.log"},
			},
			{
				name: "Move single artifact by glob pattern",
				createArtifacts: func(testDir string) error {
					return createTestFile(testDir, "test.log", "test log")
				},
				artifactPattern: filepath.Join(tmpDir, "*"),
				wantFileExist:   []string{"test.log"},
			},
			{
				name: "Move multiple artifacts by glob pattern",
				createArtifacts: func(testDir string) error {
					if err := createTestFile(testDir, "test1.log", "test log"); err != nil {
						return err
					}
					if err := createTestFile(testDir, "test2.xml", "test log"); err != nil {
						return err
					}
					return createTestFile(testDir, "test3.log", "test log")
				},
				artifactPattern: filepath.Join(tmpDir, "*"),
				wantFileExist:   []string{"test1.log", "test2.xml", "test3.log"},
			},
		}

		for _, tc := range testCases {
			Convey(tc.name, func() {
				resTmpDir, err := os.MkdirTemp("", "results")
				So(err, ShouldBeNil)
				defer os.RemoveAll(resTmpDir)

				So(tc.createArtifacts(tmpDir), ShouldBeNil)

				td := &TradefedDriver{
					logger: log.New(io.Discard, "", 0),
				}
				td.moveArtifacts(resTmpDir, tc.artifactPattern)

				for _, file := range tc.wantFileExist {
					filePath := filepath.Join(resTmpDir, file)
					_, err := os.Stat(filePath)
					So(err, ShouldBeNil)
				}
			})
		}
	})
}
