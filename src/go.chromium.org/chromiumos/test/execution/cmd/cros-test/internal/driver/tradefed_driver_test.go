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
	"go.chromium.org/chromiumos/config/go/test/api"
)

func createTestFile(dir, filename string, content string) error {
	filePath := filepath.Join(dir, filename)
	return os.WriteFile(filePath, []byte(content), 0644)
}

func createMetadataMessage(name string, tag string) *api.TestCaseMetadata {
	return &api.TestCaseMetadata{
		TestCase: &api.TestCase{
			Id:   &api.TestCase_Id{Value: "tradefed." + name},
			Name: name,
			Tags: []*api.TestCase_Tag{{Value: tag}},
		},
	}
}

func createTestCaseWithTags(tagValues []string) *api.TestCase {
	tags := []*api.TestCase_Tag{}
	for _, t := range tagValues {
		tags = append(tags, &api.TestCase_Tag{Value: t})
	}
	return &api.TestCase{
		Id:   &api.TestCase_Id{Value: "tradefed.TestCase"},
		Name: "TestCase",
		Tags: tags,
	}
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

func TestDetectTestType(t *testing.T) {
	t.Parallel()

	Convey("Test detectTestType", t, func() {
		testCases := []struct {
			name             string
			testCaseMetadata []*api.TestCaseMetadata
			expectedTestType string
		}{
			{
				name: "CTS test type by name",
				testCaseMetadata: []*api.TestCaseMetadata{
					createMetadataMessage("cts.CtsTestCase", ""),
				},
				expectedTestType: "cts",
			},
			{
				name: "CTS test type by tag",
				testCaseMetadata: []*api.TestCaseMetadata{
					createMetadataMessage("CtsTestCase", "suite:cts"),
				},
				expectedTestType: "cts",
			},
			{
				name: "CTS test type by name and tag",
				testCaseMetadata: []*api.TestCaseMetadata{
					createMetadataMessage("cts.CtsTestCase", "suite:cts"),
				},
				expectedTestType: "cts",
			},
			{
				name: "DTS test type by name",
				testCaseMetadata: []*api.TestCaseMetadata{
					createMetadataMessage("dts.CtsTestCase", ""),
				},
				expectedTestType: "dts",
			},
			{
				name: "DTS test type by tag",
				testCaseMetadata: []*api.TestCaseMetadata{
					createMetadataMessage("SomeDtsTestCase", "suite:dts"),
				},
				expectedTestType: "dts",
			},
			{
				name: "GTS test type by name",
				testCaseMetadata: []*api.TestCaseMetadata{
					createMetadataMessage("gts.CtsTestCase", ""),
				},
				expectedTestType: "gts",
			},
			{
				name: "GTS test type by tag",
				testCaseMetadata: []*api.TestCaseMetadata{
					createMetadataMessage("SomeGtsTestCase", "suite:gts"),
				},
				expectedTestType: "gts",
			},
			{
				name: "VTS test type by name",
				testCaseMetadata: []*api.TestCaseMetadata{
					createMetadataMessage("vts.CtsTestCase", ""),
				},
				expectedTestType: "vts",
			},
			{
				name: "VTS test type by tag",
				testCaseMetadata: []*api.TestCaseMetadata{
					createMetadataMessage("SomeVtsTestCase", "suite:vts"),
				},
				expectedTestType: "vts",
			},
		}

		for _, tc := range testCases {
			Convey(tc.name, func() {
				detectedTestType := detectTestType(tc.testCaseMetadata)
				So(detectedTestType, ShouldEqual, tc.expectedTestType)
			})
		}
	})
}

func TestGetArchFromBoard(t *testing.T) {
	t.Parallel()

	Convey("Test getArchFromBoard", t, func() {
		testCases := []struct {
			name         string
			board        string
			expectedArch string
		}{
			{
				name:         "Empty target",
				board:        "",
				expectedArch: "x86",
			},
			{
				name:         "Brya (x86) target",
				board:        "brya",
				expectedArch: "x86",
			},
			{
				name:         "Corsola (arm) target",
				board:        "corsola",
				expectedArch: "arm",
			},
		}

		for _, tc := range testCases {
			Convey(tc.name, func() {
				detectedArch := getArchFromBoard(tc.board)
				So(detectedArch, ShouldEqual, tc.expectedArch)
			})
		}
	})
}

func TestExtractBuildInfoFromTest(t *testing.T) {
	t.Parallel()

	Convey("Test extractBuildInfoFromTest", t, func() {
		testCases := []struct {
			name            string
			board           string
			tags            []string
			expectedBranch  string
			expectedTarget  string
			expectedBuildId string
		}{
			{
				name:            "Empty metadata",
				board:           "",
				tags:            []string{},
				expectedBranch:  "",
				expectedTarget:  "",
				expectedBuildId: "",
			},
			{
				name:            "1.0 metadata with x86 board",
				board:           "brya",
				tags:            []string{"branch:git_main", "target:target_x86_1", "build_id:12345"},
				expectedBranch:  "git_main",
				expectedTarget:  "target_x86_1",
				expectedBuildId: "12345",
			},
			{
				name:            "1.0 metadata with arm board",
				board:           "corsola",
				tags:            []string{"branch:git_main", "target:target_arm_1", "build_id:12345"},
				expectedBranch:  "git_main",
				expectedTarget:  "target_arm_1",
				expectedBuildId: "12681648",
			},
			{
				name:  "1.5 metadata with x86 board",
				board: "brya",
				tags: []string{
					"branch:git_main",
					"target_build=target:target_x86_1,build_id:12345,abi:x86_64",
					"target_build=target:target_arm_1,build_id:54321,abi:arm8_64",
				},
				expectedBranch:  "git_main",
				expectedTarget:  "target_x86_1",
				expectedBuildId: "12345",
			},
			{
				name:  "1.5 metadata with arm board",
				board: "corsola",
				tags: []string{
					"branch:git_main",
					"target_build=target:target_x86_1,build_id:12345,abi:x86_64",
					"target_build=target:target_arm_1,build_id:54321,abi:arm8_64",
				},
				expectedBranch:  "git_main",
				expectedTarget:  "target_arm_1",
				expectedBuildId: "54321",
			},
		}

		for _, tc := range testCases {
			Convey(tc.name, func() {
				testCase := createTestCaseWithTags(tc.tags)
				branch, target, build := extractBuildInfoFromTest(testCase, nil, tc.board)
				So(branch, ShouldEqual, tc.expectedBranch)
				So(target, ShouldEqual, tc.expectedTarget)
				So(build, ShouldEqual, tc.expectedBuildId)
			})
		}
	})
}
