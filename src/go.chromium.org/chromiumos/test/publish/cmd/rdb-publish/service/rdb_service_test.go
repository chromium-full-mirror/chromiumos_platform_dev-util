// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package service

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"

	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/config/go/test/artifact"
)

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
