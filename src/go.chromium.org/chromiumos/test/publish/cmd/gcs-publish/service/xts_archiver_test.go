// Copyright 2024 The ChromiumOS Authors.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package service

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestArchiveInfoForTest(t *testing.T) {
	Convey("Test archive info for test", t, func() {
		tempDir := t.TempDir()
		testCases := []struct {
			name             string
			resultDir        string
			resultFilename   string
			metadata         ArchiveMetadata
			wantInstr        []ArchiveInfo
			wantError        error
			createZipFile    bool
			createResultFile bool
		}{
			{
				name:           "success",
				resultDir:      filepath.Join(tempDir, "tauto", "results-1-cheets_CTS_12.android.api", "cheets_CTS.android.dpi", "results", "cts-results", "2024.11.17_15.15.51.364_5161"),
				resultFilename: "test_result.xml.tgz",
				metadata: ArchiveMetadata{
					APFEGCSBucket:   "gs://apfe-bucket",
					ResultGCSBucket: "gs://result-bucket",
					Build:           "brya-release/R133-16103.0.0",
					Product:         "brya.brya",
					ParentJobID:     "parent_job_id",
				},
				createZipFile:    true,
				createResultFile: true,
				wantInstr: []ArchiveInfo{
					{
						Name:        "apfe:brya.brya-release/R133-16103.0.0/parent_job_id/cheets_CTS.android.dpi/tauto_2024.11.17_15.15.51.364_5161",
						Source:      filepath.Join(tempDir, "tauto", "results-1-cheets_CTS_12.android.api", "cheets_CTS.android.dpi", "results", "cts-results", "2024.11.17_15.15.51.364_5161.zip"),
						Destination: "gs://apfe-bucket/brya.brya-release/R133-16103.0.0/parent_job_id/cheets_CTS.android.dpi/tauto_2024.11.17_15.15.51.364_5161/",
					},
					{
						Name:        "results:cheets_CTS.android.dpi/tauto_2024.11.17_15.15.51.364_5161",
						Source:      filepath.Join(tempDir, "tauto", "results-1-cheets_CTS_12.android.api", "cheets_CTS.android.dpi", "results", "cts-results", "2024.11.17_15.15.51.364_5161", "test_result.xml.gz"),
						Destination: "gs://result-bucket/cheets_CTS.android.dpi/tauto_2024.11.17_15.15.51.364_5161/",
					},
				},
			},
			{
				name:           "success with tgz result file",
				resultDir:      filepath.Join(tempDir, "tauto", "results-1-cheets_CTS_12.android.api", "cheets_CTS.android.dpi", "results", "cts-results", "2024.11.17_15.15.51.364_5161"),
				resultFilename: "test_result.xml.tgz",
				metadata: ArchiveMetadata{
					APFEGCSBucket:   "gs://apfe-bucket",
					ResultGCSBucket: "gs://result-bucket",
					Build:           "brya-release/R133-16103.0.0",
					Product:         "brya.brya",
					ParentJobID:     "parent_job_id",
				},
				createZipFile:    true,
				createResultFile: true,
				wantInstr: []ArchiveInfo{
					{
						Name:        "apfe:brya.brya-release/R133-16103.0.0/parent_job_id/cheets_CTS.android.dpi/tauto_2024.11.17_15.15.51.364_5161",
						Source:      filepath.Join(tempDir, "tauto", "results-1-cheets_CTS_12.android.api", "cheets_CTS.android.dpi", "results", "cts-results", "2024.11.17_15.15.51.364_5161.zip"),
						Destination: "gs://apfe-bucket/brya.brya-release/R133-16103.0.0/parent_job_id/cheets_CTS.android.dpi/tauto_2024.11.17_15.15.51.364_5161/",
					},
					{
						Name:        "results:cheets_CTS.android.dpi/tauto_2024.11.17_15.15.51.364_5161",
						Source:      filepath.Join(tempDir, "tauto", "results-1-cheets_CTS_12.android.api", "cheets_CTS.android.dpi", "results", "cts-results", "2024.11.17_15.15.51.364_5161", "test_result.xml.gz"),
						Destination: "gs://result-bucket/cheets_CTS.android.dpi/tauto_2024.11.17_15.15.51.364_5161/",
					},
				},
			},
			{
				name:           "isALRun, skip apfe upload",
				resultDir:      filepath.Join(tempDir, "tauto", "results-1-cheets_CTS_12.android.api", "cheets_CTS.android.dpi", "results", "cts-results", "2024.11.17_15.15.51.364_5161"),
				resultFilename: "test_result.xml",
				metadata: ArchiveMetadata{
					APFEGCSBucket:   "gs://apfe-bucket",
					ResultGCSBucket: "gs://result-bucket",
					Build:           "brya-release/R133-16103.0.0",
					Product:         "brya.brya",
					ParentJobID:     "parent_job_id",
					IsALRun:         true,
				},
				createZipFile:    false,
				createResultFile: true,
				wantInstr: []ArchiveInfo{
					{
						Name:        "results:cheets_CTS.android.dpi/tauto_2024.11.17_15.15.51.364_5161",
						Source:      filepath.Join(tempDir, "tauto", "results-1-cheets_CTS_12.android.api", "cheets_CTS.android.dpi", "results", "cts-results", "2024.11.17_15.15.51.364_5161", "test_result.xml.gz"),
						Destination: "gs://result-bucket/cheets_CTS.android.dpi/tauto_2024.11.17_15.15.51.364_5161/",
					},
				},
			},
			{
				name:             "collect-test package with empty result, skip both apfe and result upload",
				resultDir:        filepath.Join(tempDir, "tauto", "results-1-cheets_CTS_12.android.api", "cheets_CTS.android.dpi", "results", "cts-results", "2024.11.17_15.15.51.364_5161"),
				resultFilename:   "test_result.xml",
				createResultFile: false,
				metadata: ArchiveMetadata{
					APFEGCSBucket:   "gs://apfe-bucket",
					ResultGCSBucket: "gs://result-bucket",
					Build:           "brya-release/R133-16103.0.0",
					Product:         "brya.brya",
					ParentJobID:     "parent_job_id",
				},
				createZipFile: false,
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
				// Create the zip file for APFE instructions
				if tc.createZipFile {
					if err := os.MkdirAll(tc.resultDir, 0755); err != nil {
						t.Fatalf("failed to create directory structure for zip file: %v", err)
					}

					zipPath := fmt.Sprintf("%s.zip", tc.resultDir)
					if _, err := os.Create(zipPath); err != nil {
						t.Fatalf("Failed to create test zip file: %v", err)
					}
				}

				// Create the result file for result info
				if tc.createResultFile {
					resultPath := filepath.Join(tc.resultDir, tc.resultFilename)
					if filepath.Ext(tc.resultFilename) == ".tgz" {
						createTGZFile(t, resultPath, strings.TrimSuffix(tc.resultFilename, ".tgz"))
					} else {
						createFile(t, resultPath)
					}
				}
				defer os.RemoveAll(tc.resultDir)

				instr, err := archiveInfoForTest(tc.resultDir, tc.resultFilename, tc.metadata)

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

func TestResultsInfo(t *testing.T) {
	Convey("Test result info", t, func() {
		resultDir := t.TempDir()
		filePrefix := "test_prefix"
		metadata := ArchiveMetadata{
			ResultGCSBucket: "gs://result-bucket",
		}
		destination := fmt.Sprintf("%s/%s/", metadata.ResultGCSBucket, filePrefix)

		testCases := []struct {
			name           string
			resultDir      string
			resultFilename string
			createFile     bool
			wantInstr      []ArchiveInfo
			wantError      error
		}{
			{
				name:           "success",
				resultFilename: "testResult.xml",
				createFile:     true,
				wantInstr: []ArchiveInfo{
					{
						Name:        "results:test_prefix",
						Source:      filepath.Join(resultDir, "testResult.xml.gz"),
						Destination: destination,
					},
				},
			},
			{
				name:           "success with tgz extraction",
				resultFilename: "test_result.xml.tgz",
				createFile:     true,
				wantInstr: []ArchiveInfo{
					{
						Name:        "results:test_prefix",
						Source:      filepath.Join(resultDir, "test_result.xml.gz"),
						Destination: destination,
					},
				},
			},
			{
				name:           "no result files",
				resultFilename: "nonexistent.xml",
				createFile:     false,
				wantInstr:      []ArchiveInfo{},
			},
		}

		for _, tc := range testCases {
			Convey(tc.name, func() {
				if tc.createFile {
					filePath := filepath.Join(resultDir, tc.resultFilename)
					if filepath.Ext(tc.resultFilename) == ".tgz" {
						createTGZFile(t, filePath, strings.TrimSuffix(tc.resultFilename, ".tgz"))
					} else {
						createFile(t, filePath)
					}
					defer os.RemoveAll(resultDir)
				}

				instr, err := resultsInfo(resultDir, tc.resultFilename, filePrefix, metadata)

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

func createFile(t *testing.T, path string) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create directory structure for file: %v", err)
	}
	if _, err := os.Create(path); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
}

func createTGZFile(t *testing.T, tgzFile, xmlFilename string) {
	file, err := os.Create(tgzFile)
	if err != nil {
		t.Fatalf("failed to create test tgz file: %v", err)
	}
	defer file.Close()

	gzw := gzip.NewWriter(file)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	xmlBytes := []byte("<test></test>")
	hdr := &tar.Header{
		Name: xmlFilename,
		Mode: 0600,
		Size: int64(len(xmlBytes)),
	}

	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("failed to write tar header for test xml file: %v", err)
	}

	if _, err := tw.Write(xmlBytes); err != nil {
		t.Fatalf("failed to write tar file contents for test xml file: %v", err)
	}
}
