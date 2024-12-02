// Copyright 2024 The ChromiumOS Authors.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package service

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// Autotest test to collect list of CTS tests
	TEST_LIST_COLLECTOR = "tradefed-run-collect-tests-only"
)

var (
	// The regexp to extract build version from the build, e.g. brya-release/R133-16103.0.0
	BUILD_VERSION_REGEX = regexp.MustCompile(`.*/(.*)`)
)

// ArchiveInfo defines the properties of a XTS archive process.
type ArchiveInfo struct {
	// User friendly name to be displayed in the LUCI MILO UI.
	Name string

	// The full path to the local file to be uploaded to GCS.
	// e.g. /path/to/result/dir/cros-test/results/1234567890.zip
	Source string

	// The GCS path to upload the source file to.
	Destination string
}

// ArchiveMetadata defines the metadata for archiving XTS results.
type ArchiveMetadata struct {
	// The path to GCS bucket for APFE upload.
	APFEGCSBucket string

	// The path to GCS bucket for result upload.
	ResultGCSBucket string

	// The full build name. It includes the release info.
	// e.g. brya-release/R133-16103.0.0 for ChromeOS.
	Build string

	// The product/device name used to construct the upload path.
	Product string

	// The build parent job/task ID used to construct the upload path.
	ParentJobID string

	// Whether the build is for AL project.
	IsALRun bool
}

// archiveInfoForTest gets the list of XTS archive info for a test result file found under
// the resultDir.
func archiveInfoForTest(resultDir string, metadata ArchiveMetadata) ([]ArchiveInfo, error) {
	var infos []ArchiveInfo
	harness, packageName, timestamp, err := parseJobResultsFilePath(resultDir)
	if err != nil {
		return nil, err
	}

	filePrefix := fmt.Sprintf("%s/%s_%s", packageName, harness, timestamp)

	// Add APFE archive info, skipping collect-test packages and AL runs.
	// TODO: b/379711782 - Support APFE upload for AL runs if needed.
	if !metadata.IsALRun && !strings.Contains(packageName, TEST_LIST_COLLECTOR) {
		log.Printf("Fetching XTS APFE archive info for package: %s", packageName)
		apfeInfo, err := apfeInfo(resultDir, filePrefix, metadata)
		if err != nil {
			log.Printf("Failed to get APFE infos: %v", err)
		} else {
			infos = append(infos, apfeInfo...)
		}
	} else {
		log.Printf("Skipping XTS results upload to APFE gs:// bucket for package: %s", packageName)
	}

	// TODO - b/379711782: Add result archive infos
	return infos, nil
}

// apfeInfo gets the list info for archiving results to APFE GCS bucket. It scans
// the local dir for zip file that has the same name as the resultDir and constructs
// the upload name based on filePrefix and metadata.
func apfeInfo(resultDir, filePrefix string, metadata ArchiveMetadata) ([]ArchiveInfo, error) {
	// Extract the build version from the build, a success match would have size of 2, one
	// for the entire string and one for the capturing clause.
	buildMatch := BUILD_VERSION_REGEX.FindStringSubmatch(metadata.Build)
	if len(buildMatch) != 2 {
		return nil, fmt.Errorf("failed to parse build string: %s", metadata.Build)
	}

	buildFormat := fmt.Sprintf("%s-release/%s", metadata.Product, buildMatch[1])
	prefix := fmt.Sprintf("%s/%s/%s", buildFormat, metadata.ParentJobID, filePrefix)
	apfeGCSPath := fmt.Sprintf("%s/%s/", metadata.APFEGCSBucket, prefix)

	// Find all the zip files in the XTS result dir
	zipFiles, err := filepath.Glob(fmt.Sprintf("%s.zip", resultDir))
	if err != nil {
		return nil, err
	}

	infos := make([]ArchiveInfo, 0, len(zipFiles))
	for _, zipFile := range zipFiles {
		infos = append(infos, ArchiveInfo{
			Name:        "apfe:" + prefix,
			Source:      zipFile,
			Destination: apfeGCSPath,
		})
	}

	return infos, nil
}

// parseJobResultsFilePath parses the path to extract test harness, package and timestamp information
// for the job. Currently only tauto paths are supported.
//
//   - Tauto paths contain least contain 6 dirs, e.g.:
//     /cros-test/results/tauto/results-1-cheets_GTS_12.internal.all.WvtsDeviceTestCases/
//     cheets_GTS_12.internal.all.WvtsDeviceTestCases/results/android-gts/2024.11.17_15.15.51.364_5161
func parseJobResultsFilePath(path string) (harness string, packageName string, timestamp string, err error) {
	folders := strings.Split(path, string(filepath.Separator))
	foldersLen := len(folders)
	if foldersLen < 6 {
		return "", "", "", fmt.Errorf("invalid tauto path: %s", path)
	}
	return folders[foldersLen-6], folders[foldersLen-4], folders[foldersLen-1], nil
}
