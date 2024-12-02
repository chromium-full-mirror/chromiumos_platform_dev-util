// Copyright 2024 The ChromiumOS Authors.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package service

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestArchiveInfoForTest(t *testing.T) {
	Convey("Test archive info for test", t, func() {
		tempDir := t.TempDir()
		testCases := []struct {
			name          string
			resultDir     string
			metadata      ArchiveMetadata
			wantInstr     []ArchiveInfo
			wantError     error
			createZipFile bool
			isALRun       bool
		}{
			{
				name:      "success",
				resultDir: filepath.Join(tempDir, "tauto", "results-1-cheets_CTS_12.android.api", "cheets_CTS.android.dpi", "results", "cts-results", "2024.11.17_15.15.51.364_5161"),
				metadata: ArchiveMetadata{
					APFEGCSBucket:   "gs://apfe-bucket",
					ResultGCSBucket: "gs://result-bucket",
					Build:           "brya-release/R133-16103.0.0",
					Product:         "brya.brya",
					ParentJobID:     "parent_job_id",
				},
				createZipFile: true,
				wantInstr: []ArchiveInfo{
					{
						Name:        "apfe:brya.brya-release/R133-16103.0.0/parent_job_id/cheets_CTS.android.dpi/tauto_2024.11.17_15.15.51.364_5161",
						Source:      filepath.Join(tempDir, "tauto", "results-1-cheets_CTS_12.android.api", "cheets_CTS.android.dpi", "results", "cts-results", "2024.11.17_15.15.51.364_5161.zip"),
						Destination: "gs://apfe-bucket/brya.brya-release/R133-16103.0.0/parent_job_id/cheets_CTS.android.dpi/tauto_2024.11.17_15.15.51.364_5161/",
					},
				},
			},
			{
				name:      "isALRun, skip apfe upload",
				resultDir: filepath.Join(tempDir, "tauto", "results-1-cheets_CTS_12.android.api", "cheets_CTS.android.dpi", "results", "cts-results", "2024.11.17_15.15.51.364_5161"),
				metadata: ArchiveMetadata{
					APFEGCSBucket:   "gs://apfe-bucket",
					ResultGCSBucket: "gs://result-bucket",
					Build:           "brya-release/R133-16103.0.0",
					Product:         "brya.brya",
					ParentJobID:     "parent_job_id",
					IsALRun:         true,
				},
				createZipFile: false,
				wantInstr:     nil,
			},
			{
				name:      "collect-test package, skip apfe upload",
				resultDir: filepath.Join(tempDir, "tauto", "results-1-cheets_CTS_12.android.api", "cheets_CTS.android.dpi", "results", "cts-results", "2024.11.17_15.15.51.364_5161"),
				metadata: ArchiveMetadata{
					APFEGCSBucket:   "gs://apfe-bucket",
					ResultGCSBucket: "gs://result-bucket",
					Build:           "brya-release/R133-16103.0.0",
					Product:         "brya.brya",
					ParentJobID:     "parent_job_id",
				},
				createZipFile: false,
				wantInstr:     nil,
			},
			{
				name:      "invalid result dir",
				resultDir: filepath.Join("invalid", "path"),
				metadata: ArchiveMetadata{
					APFEGCSBucket:   "gs://apfe-bucket",
					ResultGCSBucket: "gs://result-bucket",
					Build:           "brya-release/R133-16103.0.0",
					Product:         "brya.brya",
					ParentJobID:     "parent_job_id",
				},
				wantError: fmt.Errorf("invalid tauto path: %s", filepath.Join("invalid", "path")),
			},
		}

		for _, tc := range testCases {
			Convey(tc.name, func() {
				if tc.createZipFile {
					if err := os.MkdirAll(tc.resultDir, 0755); err != nil {
						t.Fatalf("failed to create directory structure for zip file: %v", err)
					}

					zipPath := fmt.Sprintf("%s.zip", tc.resultDir)
					if _, err := os.Create(zipPath); err != nil {
						t.Fatalf("Failed to create test zip file: %v", err)
					}

					defer os.RemoveAll(tc.resultDir)
				}

				instr, err := archiveInfoForTest(tc.resultDir, tc.metadata)

				if tc.wantError != nil {
					So(err, ShouldResemble, tc.wantError)
				} else {
					So(err, ShouldBeNil)
					So(instr, ShouldResemble, tc.wantInstr)
				}
			})
		}
	})
}

func TestApfeInfo(t *testing.T) {
	Convey("Test APFE info", t, func() {
		// Create dir1 under test temp dir
		tempDir := t.TempDir()
		testDir, err := os.MkdirTemp(tempDir, "dir1")
		if err != nil {
			t.Fatal(err)
		}
		zipFilepath := fmt.Sprintf("%s.zip", testDir)
		defer os.RemoveAll(testDir)

		testCases := []struct {
			name          string
			filePrefix    string
			metadata      ArchiveMetadata
			createZipFile bool
			wantInstr     []ArchiveInfo
			wantError     error
		}{
			{
				name:       "success",
				filePrefix: "test_prefix",
				metadata: ArchiveMetadata{
					APFEGCSBucket: "gs://apfe-bucket",
					Build:         "brya-release/R133-16103.0.0",
					Product:       "brya.brya",
					ParentJobID:   "parent_job_id",
				},
				createZipFile: true,
				wantInstr: []ArchiveInfo{
					{
						Name:        "apfe:brya.brya-release/R133-16103.0.0/parent_job_id/test_prefix",
						Source:      zipFilepath,
						Destination: "gs://apfe-bucket/brya.brya-release/R133-16103.0.0/parent_job_id/test_prefix/",
					},
				},
			},
			{
				name:       "no zip files",
				filePrefix: "test_prefix",
				metadata: ArchiveMetadata{
					APFEGCSBucket: "gs://apfe-bucket",
					Build:         "brya.brya-release/R133-16103.0.0",
					Product:       "brya.brya",
					ParentJobID:   "parent_job_id",
				},
				wantInstr: []ArchiveInfo{},
			},
			{
				name:       "invalid build string",
				filePrefix: "test_prefix",
				metadata: ArchiveMetadata{
					APFEGCSBucket: "gs://apfe-bucket",
					Build:         "invalid-build",
					Product:       "brya.brya",
					ParentJobID:   "parent_job_id",
				},
				wantError: fmt.Errorf("failed to parse build string: invalid-build"),
			},
		}

		for _, tc := range testCases {
			Convey(tc.name, func() {
				if tc.createZipFile {
					zipFile, err := os.Create(zipFilepath)
					if err != nil {
						t.Fatalf("Failed to create test zip file: %v", err)
					}
					zipFile.Close()
				}

				instr, err := apfeInfo(testDir, tc.filePrefix, tc.metadata)

				if tc.wantError != nil {
					So(err, ShouldResemble, tc.wantError)
				} else {
					So(err, ShouldBeNil)
					So(instr, ShouldResemble, tc.wantInstr)
				}
			})
		}
	})
}

func TestParseJobResultsFilePath(t *testing.T) {
	Convey("Test parse job results filepath", t, func() {
		testCases := []struct {
			name        string
			path        string
			wantHarness string
			wantPkg     string
			wantTs      string
			wantError   error
		}{
			{
				name:        "tauto path",
				path:        "/cros-test/results/tauto/results-1-cheets_GTS_12.internal.all.WvtsDeviceTestCases/cheets_GTS_12.internal.all.WvtsDeviceTestCases/results/android-gts/2024.11.17_15.15.51.364_5161",
				wantHarness: "tauto",
				wantPkg:     "cheets_GTS_12.internal.all.WvtsDeviceTestCases",
				wantTs:      "2024.11.17_15.15.51.364_5161",
			},
			{
				name:      "invalid tauto path: less than 6",
				path:      "cros-test/results/tauto/results-1-cheets_GTS_12.internal.all.WvtsDeviceTestCases",
				wantError: fmt.Errorf("invalid tauto path: cros-test/results/tauto/results-1-cheets_GTS_12.internal.all.WvtsDeviceTestCases"),
			},
		}

		for _, tc := range testCases {
			Convey(tc.name, func() {
				harness, packageName, timestamp, err := parseJobResultsFilePath(tc.path)

				if tc.wantError != nil {
					So(err, ShouldResemble, tc.wantError)
				} else {
					So(err, ShouldBeNil)
					So(harness, ShouldEqual, tc.wantHarness)
					So(packageName, ShouldEqual, tc.wantPkg)
					So(timestamp, ShouldEqual, tc.wantTs)
				}
			})
		}
	})
}
