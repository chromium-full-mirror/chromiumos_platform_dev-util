// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package service

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"

	_go "go.chromium.org/chromiumos/config/go"
	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/config/go/test/artifact"
	"go.chromium.org/chromiumos/test/publish/clients/rdb_client"
)

func TestIsChromiumTest(t *testing.T) {
	t.Parallel()

	Convey("Is Chromium Test", t, func() {
		// Create a test result proto with resultdb_settings flag in test args.
		testResult := &artifact.TestResult{
			TestRuns: []*artifact.TestRun{
				{
					ExecutionMetadata: &artifact.ExecutionMetadata{
						TestArgs: map[string]string{
							"resultdb_settings": "test",
						},
					},
				},
			},
		}

		// Call isChromiumTest.
		isChromium := isChromiumTest(testResult)

		// Verify that the function returns true.
		So(isChromium, ShouldBeTrue)
	})

	Convey("Is not Chromium Test", t, func() {
		// Create a test result proto without resultdb_settings flag in test
		// args.
		testResult := &artifact.TestResult{
			TestRuns: []*artifact.TestRun{
				{
					ExecutionMetadata: &artifact.ExecutionMetadata{
						TestArgs: map[string]string{},
					},
				},
			},
		}

		// Call isChromiumTest.
		isChromium := isChromiumTest(testResult)

		// Verify that the function returns false.
		So(isChromium, ShouldBeFalse)
	})
}

func TestExtractBaseChromiumRDBConfig(t *testing.T) {
	t.Parallel()

	Convey("Extract Base Chromium RDB Config", t, func() {
		// Create a test args map with resultdb_settings flag.
		testArgs := map[string]string{
			"resultdb_settings": "eyJiYXNlX3ZhcmlhbnQiOiB7ImJ1aWxkZXIiOiAiY2hyb21lb3MtYmV0dHktY2hyb21lIiwgImNyb3NfaW1nIjogImJldHR5LXJlbGVhc2UvUjEyNy0xNTkxMi4wLjAiLCAiZGV2aWNlX3R5cGUiOiAiYmV0dHkiLCAib3MiOiAiQ2hyb21lT1MiLCAidGVzdF9zdWl0ZSI6ICJtZWRpYV91bml0dGVzdHMgUkVMRUFTRV9MS0dNIn0sICJjb2VyY2VfbmVnYXRpdmVfZHVyYXRpb24iOiB0cnVlLCAiZXhvbmVyYXRlX3VuZXhwZWN0ZWRfcGFzcyI6IHRydWUsICJpbmNsdWRlIjogZmFsc2UsICJyZXN1bHRfZm9ybWF0IjogImd0ZXN0IiwgInRlc3RfaWRfcHJlZml4IjogIm5pbmphOi8vbWVkaWE6bWVkaWFfdW5pdHRlc3RzLyJ9",
		}
		wantRDBConfig := map[string]interface{}{
			"base_variant": map[string]any{
				"builder":     "chromeos-betty-chrome",
				"cros_img":    "betty-release/R127-15912.0.0",
				"device_type": "betty",
				"os":          "ChromeOS",
				"test_suite":  "media_unittests RELEASE_LKGM",
			},
			"result_format":             "gtest",
			"coerce_negative_duration":  true,
			"exonerate_unexpected_pass": true,
			"include":                   false,
			"test_id_prefix":            "ninja://media:media_unittests/",
		}

		// Call extractBaseChromiumRDBConfig.
		gotRDBSettings, err := extractBaseChromiumRDBConfig(testArgs)

		// Verify that the function returns the correct rdb settings.
		So(err, ShouldBeNil)
		So(gotRDBSettings, ShouldResemble, wantRDBConfig)
	})

	Convey("Extract Base Chromium RDB Config with invalid base64 string", t, func() {
		// Create a test args map with invalid resultdb_settings flag.
		testArgs := map[string]string{
			"resultdb_settings": "invalid_base64_string",
		}

		// Call extractBaseChromiumRDBConfig.
		gotRDBSettings, err := extractBaseChromiumRDBConfig(testArgs)

		// Verify that the function returns an error.
		So(err, ShouldNotBeNil)
		So(gotRDBSettings, ShouldBeNil)
	})

	Convey("Extract Base Chromium RDB Config with invalid JSON string", t, func() {
		// Create a test args map with invalid resultdb_settings flag.
		testArgs := map[string]string{
			"resultdb_settings": "eyJiYXNlX3ZhcmlhbnQiOiB7ImJ1aWxkZXIiOiAiY2hyb21lb3MtYmV0dHktY2hyb21lIiwgImNyb3NfaW1nIjogImJldHR5LXJlbGVhc2UvUjEyNy0xNTkxMi4wLjAiLCAiZGV2aWNlX3R5cGUiOiAiYmV0dHkiLCAib3MiOiAiQ2hyb21lT1MiLCAidGVzdF9zdWl0ZSI6ICJtZWRpYV91bml0dGVzdHMgUkVMRUFTRV9MS0dNIn0sICJjb2VyY2VfbmVnYXRpdmVfZHVyYXRpb24iOiB0cnVlLCAiZXhvbmVyYXRlX3VuZXhwZWN0ZWRfcGFzcyI6IHRydWUsICJpbmNsdWRlIjogZmFsc2UsICJyZXN1bHRfZm9ybWF0IjogImd0ZXN0IiwgInRlc3RfaWRfcHJlZml4IjogIm5pbmphOi8vbWVkaWE6bWV9",
		}

		// Call extractBaseChromiumRDBConfig.
		rdbSettings, err := extractBaseChromiumRDBConfig(testArgs)

		// Verify that the function returns an error.
		So(err, ShouldNotBeNil)
		So(rdbSettings, ShouldBeNil)
	})
}

func TestChromiumTestRDBConfig(t *testing.T) {
	t.Parallel()

	Convey("Chromium Test RDB Config for gtest result format", t, func() {
		// Create a test result proto with resultdb_settings flag in test args.
		testResult := &artifact.TestResult{
			TestRuns: []*artifact.TestRun{
				{
					TestCaseInfo: &artifact.TestCaseInfo{
						TestCaseResult: &api.TestCaseResult{
							ResultDirPath: &_go.StoragePath{
								Path: "test/results/dir",
							},
						},
					},
					ExecutionMetadata: &artifact.ExecutionMetadata{
						TestArgs: map[string]string{
							"resultdb_settings": "eyJiYXNlX3ZhcmlhbnQiOiB7ImJ1aWxkZXIiOiAiY2hyb21lb3MtYmV0dHktY2hyb21lIiwgImNyb3NfaW1nIjogImJldHR5LXJlbGVhc2UvUjEyNy0xNTkxMi4wLjAiLCAiZGV2aWNlX3R5cGUiOiAiYmV0dHkiLCAib3MiOiAiQ2hyb21lT1MiLCAidGVzdF9zdWl0ZSI6ICJtZWRpYV91bml0dGVzdHMgUkVMRUFTRV9MS0dNIn0sICJjb2VyY2VfbmVnYXRpdmVfZHVyYXRpb24iOiB0cnVlLCAiZXhvbmVyYXRlX3VuZXhwZWN0ZWRfcGFzcyI6IHRydWUsICJpbmNsdWRlIjogZmFsc2UsICJyZXN1bHRfZm9ybWF0IjogImd0ZXN0IiwgInRlc3RfaWRfcHJlZml4IjogIm5pbmphOi8vbWVkaWE6bWVkaWFfdW5pdHRlc3RzLyJ9",
						},
					},
				},
			},
		}
		wantRDBConfig := &rdb_client.RdbStreamConfig{
			BaseTags: map[string]string{},
			BaseVariant: map[string]string{
				"builder":     "chromeos-betty-chrome",
				"cros_img":    "betty-release/R127-15912.0.0",
				"device_type": "betty",
				"os":          "ChromeOS",
				"test_suite":  "media_unittests RELEASE_LKGM",
			},
			ResultFile:              "test/results/dir/chromium/results/output.json",
			ResultFormat:            "gtest",
			TesthausBaseURL:         "testhaus_url",
			TestIdPrefix:            "ninja://media:media_unittests/",
			CoerceNegativeDuration:  true,
			ExonerateUnexpectedPass: true,
			Include:                 false,
		}

		// Call chromiumTestRDBConfig.
		gotRDBconfig, err := chromiumTestRDBConfig(testResult, map[string]string{}, map[string]string{}, "testhaus_url", "")

		// Verify that the function returns the correct rdb config.
		So(err, ShouldBeNil)
		So(gotRDBconfig, ShouldResemble, wantRDBConfig)
	})

	Convey("Chromium Test RDB Config for tast result format", t, func() {
		// Create a test result proto with resultdb_settings flag in test args.
		testResult := &artifact.TestResult{
			TestRuns: []*artifact.TestRun{
				{
					TestCaseInfo: &artifact.TestCaseInfo{
						TestCaseResult: &api.TestCaseResult{
							ResultDirPath: &_go.StoragePath{
								Path: "test/results/dir",
							},
						},
					},
					ExecutionMetadata: &artifact.ExecutionMetadata{
						TestArgs: map[string]string{
							"resultdb_settings": "eyJiYXNlX3ZhcmlhbnQiOiB7ImJ1aWxkZXIiOiAiY2hyb21lb3MtYmV0dHktY2hyb21lIiwgImNyb3NfaW1nIjogImJldHR5LXJlbGVhc2UvUjEyNy0xNTkxMi4wLjAiLCAiZGV2aWNlX3R5cGUiOiAiYmV0dHkiLCAib3MiOiAiQ2hyb21lT1MiLCAidGVzdF9zdWl0ZSI6ICJtZWRpYV91bml0dGVzdHMgUkVMRUFTRV9MS0dNIn0sICJjb2VyY2VfbmVnYXRpdmVfZHVyYXRpb24iOiB0cnVlLCAiZXhvbmVyYXRlX3VuZXhwZWN0ZWRfcGFzcyI6IHRydWUsICJpbmNsdWRlIjogZmFsc2UsICJyZXN1bHRfZm9ybWF0IjogInRhc3QiLCAidGVzdF9pZF9wcmVmaXgiOiAibmluamE6Ly9tZWRpYTptZWRpYV91bml0dGVzdHMvIn0=",
						},
					},
				},
			},
		}
		wantResultFile := "cros-test-777c42c3/cros-test/results/tauto/results-1-tast.chrome-from-gcs/tast/results/streamed_results.jsonl"
		wantRDBConfig := &rdb_client.RdbStreamConfig{
			BaseTags: map[string]string{},
			BaseVariant: map[string]string{
				"builder":     "chromeos-betty-chrome",
				"cros_img":    "betty-release/R127-15912.0.0",
				"device_type": "betty",
				"os":          "ChromeOS",
				"test_suite":  "media_unittests RELEASE_LKGM",
			},
			ResultFile:              wantResultFile,
			ResultFormat:            ResultAdapterResultFormat,
			TesthausBaseURL:         "testhaus_url",
			TestIdPrefix:            "ninja://media:media_unittests/",
			CoerceNegativeDuration:  true,
			ExonerateUnexpectedPass: true,
			Include:                 false,
		}

		// Call chromiumTestRDBConfig.
		gotRDBconfig, err := chromiumTestRDBConfig(testResult, map[string]string{}, map[string]string{}, "testhaus_url", wantResultFile)

		// Verify that the function returns the correct rdb config.
		So(err, ShouldBeNil)
		So(gotRDBconfig, ShouldResemble, wantRDBConfig)
	})

	Convey("Chromium Test RDB Config with invalid resultdb_settings flag", t, func() {
		// Create a test result proto with invalid resultdb_settings flag in test args.
		testResult := &artifact.TestResult{
			TestRuns: []*artifact.TestRun{
				{
					TestCaseInfo: &artifact.TestCaseInfo{
						TestCaseResult: &api.TestCaseResult{
							ResultDirPath: &_go.StoragePath{
								Path: "test/results/dir",
							},
						},
					},
					ExecutionMetadata: &artifact.ExecutionMetadata{
						TestArgs: map[string]string{
							"resultdb_settings": "invalid_base64_string",
						},
					},
				},
			},
		}

		// Call chromiumTestRDBConfig.
		config, err := chromiumTestRDBConfig(testResult, map[string]string{}, map[string]string{}, "testhaus_url", "")

		// Verify that the function returns an error.
		So(err, ShouldNotBeNil)
		So(config, ShouldBeNil)
	})
}

func TestIngestPostProcessResponses(t *testing.T) {
	t.Parallel()

	Convey("Ingest post process responses", t, func() {
		// Create a test result proto.
		testResult := &artifact.TestResult{
			TestInvocation: &artifact.TestInvocation{
				PrimaryExecutionInfo: &artifact.ExecutionInfo{
					BuildInfo: &artifact.BuildInfo{
						BuildMetadata: &artifact.BuildMetadata{},
					},
				},
			},
			TestRuns: []*artifact.TestRun{
				{
					TestCaseInfo: &artifact.TestCaseInfo{
						TestCaseResult: &api.TestCaseResult{
							TestCaseId: &api.TestCase_Id{
								Value: "tast.gscdevboard.GSCSysinfo",
							},
						},
					},
				},
			},
		}

		// Create a post process response proto.
		postProcessResps := &api.RunActivitiesResponse{
			Responses: []*api.RunActivityResponse{
				{
					Response: &api.RunActivityResponse_GetFwInfoResponse{
						GetFwInfoResponse: &api.GetFWInfoResponse{
							RoFwid:        "Google_Voema.13672.224.0",
							RwFwid:        "Google_Voema.13672.224.1",
							KernelVersion: "5.4.151-16902-g93699f4e73de",
						},
					},
				},
				{
					Response: &api.RunActivityResponse_GetGfxInfoResponse{
						GetGfxInfoResponse: &api.GetGfxInfoResponse{
							GfxLabels: map[string]string{
								"display_panel_name":    "AUO 10380",
								"display_present_hdr":   "hdr unsupported",
								"display_present_psr":   "psr unsupported",
								"display_present_vrr":   "vrr unsupported",
								"display_refresh_rate":  "60.06",
								"display_resolution":    "1366x768",
								"gpu_family":            "cezanne",
								"gpu_id":                "amd:15e7",
								"gpu_open_gles_version": "3.2",
								"gpu_vendor":            "amd",
								"gpu_vulkan_version":    "1.3.274",
								"platform_cpu_vendor":   "amd",
								"platform_disk_size":    "128",
								"platform_memory_size":  "32",
							},
						},
					},
				},
				{
					Response: &api.RunActivityResponse_GetAvlInfoResponse{
						GetAvlInfoResponse: &api.GetAvlInfoResponse{
							AvlInfos: map[string]*api.AvlInfo{
								"tast.gscdevboard.GSCSysinfo": {
									AvlComponentType: "storage",
									AvlPartFirmware:  "0xa200000000000000",
									AvlPartModel:     "0x0000f5 MMC32G",
								},
							},
						},
					},
				},
			},
		}

		// Call ingestPostProcessResponses.
		ingestPostProcessResponses(testResult, postProcessResps)

		// Verify that the all info are populated correctly.
		wantBuildMetadata := &artifact.BuildMetadata{
			Firmware: &artifact.BuildMetadata_Firmware{
				RoVersion: "Google_Voema.13672.224.0",
				RwVersion: "Google_Voema.13672.224.1",
			},
			Kernel: &artifact.BuildMetadata_Kernel{
				Version: "5.4.151-16902-g93699f4e73de",
			},
			GfxInfo: &artifact.BuildMetadata_GfxInfo{
				DisplayPanelName:   "AUO 10380",
				DisplayPresentHdr:  "hdr unsupported",
				DisplayPresentPsr:  "psr unsupported",
				DisplayPresentVrr:  "vrr unsupported",
				DisplayRefreshRate: "60.06",
				DisplayResolution:  "1366x768",
				GpuFamily:          "cezanne",
				GpuId:              "amd:15e7",
				GpuOpenGlesVersion: "3.2",
				GpuVendor:          "amd",
				GpuVulkanVersion:   "1.3.274",
				PlatformCpuVendor:  "amd",
				PlatformDiskSize:   128,
				PlatformMemorySize: 32,
			},
		}
		wantAVL := &artifact.AvlInfo{
			AvlComponentType: "storage",
			AvlPartFirmware:  "0xa200000000000000",
			AvlPartModel:     "0x0000f5 MMC32G",
		}
		So(testResult.TestInvocation.PrimaryExecutionInfo.BuildInfo.BuildMetadata, ShouldResembleProto, wantBuildMetadata)
		So(testResult.TestRuns[0].TestCaseInfo.AvlInfo, ShouldResembleProto, wantAVL)
	})
}

func TestPopulateFirmwareInfo(t *testing.T) {
	t.Parallel()

	Convey("Populates Firmware info", t, func() {
		// Create a build metadata proto.
		buildMetadata := &artifact.BuildMetadata{}

		// Create a firmware info response proto.
		fwInfoResp := &api.GetFWInfoResponse{
			RoFwid: "Google_Voema.13672.224.0",
			RwFwid: "Google_Voema.13672.224.1",
		}

		// Call populateFirmwareInfo.
		populateFirmwareInfo(buildMetadata, fwInfoResp)

		// Verify that the firmware info is populated correctly.
		wantFwInfo := &artifact.BuildMetadata_Firmware{
			RoVersion: "Google_Voema.13672.224.0",
			RwVersion: "Google_Voema.13672.224.1",
		}
		So(buildMetadata.Firmware, ShouldResembleProto, wantFwInfo)
	})

	Convey("Populates Firmware info with empty values", t, func() {
		// Create a build metadata proto.
		buildMetadata := &artifact.BuildMetadata{}

		// Create a firmware info response proto with empty values.
		fwInfoResp := &api.GetFWInfoResponse{}

		// Call populateFirmwareInfo.
		populateFirmwareInfo(buildMetadata, fwInfoResp)

		// Verify that the firmware info is populated correctly.
		So(buildMetadata.Firmware, ShouldResembleProto, &artifact.BuildMetadata_Firmware{})
	})

	Convey("Skip if the response is nil", t, func() {
		// Create a build metadata proto.
		buildMetadata := &artifact.BuildMetadata{}

		// Call populateFirmwareInfo.
		populateFirmwareInfo(buildMetadata, nil)

		// Verify no value is populated.
		So(buildMetadata.Firmware, ShouldBeNil)
	})
}

func TestPopulateKernelInfo(t *testing.T) {
	t.Parallel()

	Convey("Populates Kernel info", t, func() {
		// Create a build metadata proto.
		buildMetadata := &artifact.BuildMetadata{}

		// Create a firmware info response proto.
		fwInfoResp := &api.GetFWInfoResponse{
			KernelVersion: "5.4.151-16902-g93699f4e73de",
		}

		// Call populateKernelInfo.
		populateKernelInfo(buildMetadata, fwInfoResp)

		// Verify that the kernel info is populated correctly.
		wantKernel := &artifact.BuildMetadata_Kernel{
			Version: "5.4.151-16902-g93699f4e73de",
		}
		So(buildMetadata.Kernel, ShouldResembleProto, wantKernel)
	})

	Convey("Populates Kernel info with empty values", t, func() {
		// Create a build metadata proto.
		buildMetadata := &artifact.BuildMetadata{}

		// Create a firmware info response proto with empty values.
		fwInfoResp := &api.GetFWInfoResponse{}

		// Call populateKernelInfo.
		populateKernelInfo(buildMetadata, fwInfoResp)

		// Verify that the kernel info is populated correctly.
		So(buildMetadata.Kernel, ShouldResembleProto, &artifact.BuildMetadata_Kernel{})
	})

	Convey("Skip if the response is nil", t, func() {
		// Create a build metadata proto.
		buildMetadata := &artifact.BuildMetadata{}

		// Call populateKernelInfo.
		populateKernelInfo(buildMetadata, nil)

		// Verify no value is populated.
		So(buildMetadata.Kernel, ShouldBeNil)
	})
}

func TestPopulateGfxInfo(t *testing.T) {
	t.Parallel()

	Convey("Populates Graphics info", t, func() {
		// Create a build metadata proto.
		buildMetadata := &artifact.BuildMetadata{}

		// Create a Graphics info response proto.
		gfxInfoResp := &api.GetGfxInfoResponse{
			GfxLabels: map[string]string{
				"display_panel_name":    "AUO 10380",
				"display_present_hdr":   "hdr unsupported",
				"display_present_psr":   "psr unsupported",
				"display_present_vrr":   "vrr unsupported",
				"display_refresh_rate":  "60.06",
				"display_resolution":    "1366x768",
				"gpu_family":            "cezanne",
				"gpu_id":                "amd:15e7",
				"gpu_open_gles_version": "3.2",
				"gpu_vendor":            "amd",
				"gpu_vulkan_version":    "1.3.274",
				"platform_cpu_vendor":   "amd",
				"platform_disk_size":    "128",
				"platform_memory_size":  "32",
			},
		}

		// Call populateGfxInfo.
		populateGfxInfo(buildMetadata, gfxInfoResp)

		// Verify that the Graphics info is populated correctly.
		wantGFXInfo := &artifact.BuildMetadata_GfxInfo{
			DisplayPanelName:   "AUO 10380",
			DisplayPresentHdr:  "hdr unsupported",
			DisplayPresentPsr:  "psr unsupported",
			DisplayPresentVrr:  "vrr unsupported",
			DisplayRefreshRate: "60.06",
			DisplayResolution:  "1366x768",
			GpuFamily:          "cezanne",
			GpuId:              "amd:15e7",
			GpuOpenGlesVersion: "3.2",
			GpuVendor:          "amd",
			GpuVulkanVersion:   "1.3.274",
			PlatformCpuVendor:  "amd",
			PlatformDiskSize:   128,
			PlatformMemorySize: 32,
		}
		So(buildMetadata.GfxInfo, ShouldResembleProto, wantGFXInfo)
	})

	Convey("Populates Graphics info with empty values", t, func() {
		// Create a build metadata proto.
		buildMetadata := &artifact.BuildMetadata{}

		// Create a Graphics info response proto with empty values.
		gfxInfoResp := &api.GetGfxInfoResponse{}

		// Call populateGfxInfo.
		populateGfxInfo(buildMetadata, gfxInfoResp)

		// Verify that the Graphics info is populated correctly.
		So(buildMetadata.GfxInfo, ShouldResembleProto, &artifact.BuildMetadata_GfxInfo{})
	})

	Convey("Skip if the response is nil", t, func() {
		// Create a build metadata proto.
		buildMetadata := &artifact.BuildMetadata{}

		// Call populateGfxInfo.
		populateGfxInfo(buildMetadata, nil)

		// Verify no value is populated.
		So(buildMetadata.GfxInfo, ShouldBeNil)
	})
}

func TestPopulateAVLInfo(t *testing.T) {
	t.Parallel()

	Convey("Populates AVL info", t, func() {
		// Create a test result proto with a test run and a test case.
		testResult := &artifact.TestResult{
			TestRuns: []*artifact.TestRun{
				{
					TestCaseInfo: &artifact.TestCaseInfo{
						TestCaseResult: &api.TestCaseResult{
							TestCaseId: &api.TestCase_Id{
								Value: "tast.gscdevboard.GSCSysinfo",
							},
						},
					},
				},
			},
		}

		// Create an AVL info response proto with AVL info for the test case.
		avlInfoResp := &api.GetAvlInfoResponse{
			AvlInfos: map[string]*api.AvlInfo{
				"tast.gscdevboard.GSCSysinfo": {
					AvlComponentType: "storage",
					AvlPartFirmware:  "0xa200000000000000",
					AvlPartModel:     "0x0000f5 MMC32G",
				},
			},
		}

		// Call populateAVLInfo.
		populateAVLInfo(testResult, avlInfoResp)

		wantAVL := &artifact.AvlInfo{
			AvlComponentType: "storage",
			AvlPartFirmware:  "0xa200000000000000",
			AvlPartModel:     "0x0000f5 MMC32G",
		}
		So(testResult.TestRuns[0].TestCaseInfo.AvlInfo, ShouldResembleProto, wantAVL)
	})

	Convey("Populates AVL info with missing test case", t, func() {
		// Create a test result proto with a test run without any test case.
		testResult := &artifact.TestResult{
			TestRuns: []*artifact.TestRun{
				{
					TestCaseInfo: &artifact.TestCaseInfo{},
				},
			},
		}

		// Create an AVL info response proto with AVL info for the test case.
		avlInfoResp := &api.GetAvlInfoResponse{
			AvlInfos: map[string]*api.AvlInfo{
				"tast.gscdevboard.GSCSysinfo": {
					AvlComponentType: "storage",
					AvlPartFirmware:  "0xa200000000000000",
					AvlPartModel:     "0x0000f5 MMC32G",
				},
			},
		}

		// Call populateAVLInfo.
		populateAVLInfo(testResult, avlInfoResp)

		So(testResult.TestRuns[0].TestCaseInfo.AvlInfo, ShouldBeNil)
	})

	Convey("Populates AVL info with missing AVL info", t, func() {
		// Create a test result proto with a test run and a test case.
		testResult := &artifact.TestResult{
			TestRuns: []*artifact.TestRun{
				{
					TestCaseInfo: &artifact.TestCaseInfo{
						TestCaseResult: &api.TestCaseResult{
							TestCaseId: &api.TestCase_Id{
								Value: "tast.gscdevboard.GSCSysinfo",
							},
						},
					},
				},
			},
		}

		// Create an AVL info response proto with AVL info for the test case.
		avlInfoResp := &api.GetAvlInfoResponse{
			AvlInfos: map[string]*api.AvlInfo{},
		}

		// Call populateAVLInfo.
		populateAVLInfo(testResult, avlInfoResp)

		So(testResult.TestRuns[0].TestCaseInfo.AvlInfo, ShouldBeNil)
	})

	Convey("Skip if the response is nil", t, func() {
		// Create a test result proto with a test run and a test case.
		testResult := &artifact.TestResult{
			TestRuns: []*artifact.TestRun{
				{
					TestCaseInfo: &artifact.TestCaseInfo{
						TestCaseResult: &api.TestCaseResult{
							TestCaseId: &api.TestCase_Id{
								Value: "tast.gscdevboard.GSCSysinfo",
							},
						},
					},
				},
			},
		}

		// Call populateAVLInfo.
		populateAVLInfo(testResult, nil)

		So(testResult.TestRuns[0].TestCaseInfo.AvlInfo, ShouldBeNil)
	})
}
