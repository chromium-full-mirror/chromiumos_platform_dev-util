// Copyright 2022 The ChromiumOS Authors.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"

	"google.golang.org/protobuf/proto"

	"go.chromium.org/chromiumos/test/publish/clients/rdb_client"
	common_utils "go.chromium.org/chromiumos/test/publish/cmd/common-utils"
	"go.chromium.org/chromiumos/test/publish/cmd/publishserver/storage"
	"go.chromium.org/chromiumos/test/publish/libs/rdb_lib"
	"go.chromium.org/chromiumos/test/util/common"

	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/config/go/test/api/metadata"
	"go.chromium.org/chromiumos/config/go/test/artifact"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/luci/common/errors"
	rdb_pb "go.chromium.org/luci/resultdb/proto/v1"
)

const (
	// Path to rdb executable
	RdbExecutablePath = "/usr/bin/rdb"

	// Path to result_adapter executable
	ResultAdapterExecutablePath = "/usr/bin/result_adapter"

	// Path to temp directory for rdb-publish
	RdbTempDirName = "rdb-publish-temp"

	// File name for test result json
	TestResultJsonFileName = "testResult.json"

	// File name for invocation-level properties json
	InvPropertiesFile = "invProperties.json"

	// File name for code sources under test
	CodeSourcesJsonFileName = "sources.jsonpb"

	// Result format for rdb
	ResultAdapterResultFormat = "cros-test-result"

	// MaxSizeInvocationProperties is the maximum size of the invocation level
	// properties stored in ResultDB.
	MaxSizeInvocationProperties = 16 * 1024 // 16 KB

	// Path to all results artifact directory
	ResultsArtifactDir = "/tmp/test/results"
)

type RdbPublishService struct {
	RetryCount           int
	CurrentInvocationId  string
	TestResultProto      *artifact.TestResult
	TesthausURL          string
	TempDirPath          string
	Sources              *metadata.PublishRdbMetadata_Sources
	BaseVariant          map[string]string
	PostProcessResponses *api.RunActivitiesResponse
}

// NewRdbPublishService creates a new ResultDB publish service to interact with
// ResultDB.
func NewRdbPublishService(req *api.PublishRequest) (*RdbPublishService, error) {
	m, err := unpackMetadata(req)
	if err != nil {
		return nil, err
	}

	if err = common_utils.ValidateRDBPublishRequest(req, m); err != nil {
		return nil, err
	}

	retryCount := 0
	if req.GetRetryCount() > 0 {
		retryCount = int(req.GetRetryCount())
	}

	return &RdbPublishService{
		RetryCount:           retryCount,
		CurrentInvocationId:  m.GetCurrentInvocationId(),
		TestResultProto:      m.GetTestResult(),
		TesthausURL:          m.GetTesthausUrl(),
		Sources:              m.GetSources(),
		BaseVariant:          m.GetBaseVariant(),
		PostProcessResponses: m.GetPostProcessResponses(),
	}, nil
}

// UploadToRdb uploads test results to ResultDB.
func (rps *RdbPublishService) UploadToRdb(ctx context.Context) error {
	rdbClient := rdb_client.RdbClient{RdbExecutablePath: RdbExecutablePath, ResultAdapterExecutablePath: ResultAdapterExecutablePath}
	rdbLib := rdb_lib.RdbLib{CurrentInvocation: rps.CurrentInvocationId, RdbClient: &rdbClient}

	// Create temp dir if required
	if rps.TempDirPath == "" {
		dirPath, err := common_utils.MakeTempDir(ctx, "", RdbTempDirName)
		if err != nil {
			return fmt.Errorf("error during creating temp dir for rdb publish: %w", err)
		}
		rps.TempDirPath = dirPath
	}

	// Populate additional info in test results.
	ingestPostProcessResponses(rps.TestResultProto, rps.PostProcessResponses)

	// Write test result to file
	testResultFilePath := path.Join(rps.TempDirPath, TestResultJsonFileName)
	err := common_utils.WriteProtoJsonFile(ctx, testResultFilePath, rps.TestResultProto)
	if err != nil {
		return fmt.Errorf("error during writing test result proto to file: %w", err)
	}
	log.Printf("test result file created in path: %q", testResultFilePath)

	// Write test invocation proto to invocation-level properties file
	testInv := rps.TestResultProto.GetTestInvocation()
	invPropertiesFilePath := path.Join(rps.TempDirPath, InvPropertiesFile)
	err = common_utils.WriteProtoJsonFile(ctx, invPropertiesFilePath, testInv)
	if err != nil {
		return fmt.Errorf("error during writing test invocation proto to file: %w", err)
	}
	log.Printf("invocation properties file created in path: %q", invPropertiesFilePath)

	sourcesFilePath := path.Join(rps.TempDirPath, CodeSourcesJsonFileName)
	hasSources, err := downloadSources(ctx, sourcesFilePath, rps.Sources)
	if err != nil {
		return fmt.Errorf("downloading sources: %w", err)
	}
	if hasSources {
		log.Printf("code sources under test stored at path: %q", sourcesFilePath)
	}

	// Upload invocation artifacts
	if rps.TesthausURL != "" {
		artifact := rdb_pb.Artifact{ArtifactId: "testhaus_logs", ContentType: "text/x-uri", Contents: []byte(rps.TesthausURL)}
		err := rdbLib.UploadInvocationArtifacts(ctx, &artifact)
		if err != nil {
			return fmt.Errorf("error during rdb invocation artifact upload: %w", err)
		}
	}

	// Upload test results
	baseTags := map[string]string{}
	config := &rdb_client.RdbStreamConfig{
		BaseTags:        baseTags,
		BaseVariant:     rps.BaseVariant,
		ResultFile:      testResultFilePath,
		ResultFormat:    ResultAdapterResultFormat,
		TesthausBaseURL: rps.TesthausURL,
	}
	if hasSources {
		config.SourcesFile = sourcesFilePath
	}

	if validateInvProperties(testInv) {
		config.InvPropertiesFile = invPropertiesFilePath
	} else {
		log.Printf("invocation properties file is skipped from the upload because it exceeds the maximum size of: %d bytes", MaxSizeInvocationProperties)
	}

	_, err = os.Stat(ResultsArtifactDir)
	if err != nil {
		log.Printf("Warning: test result artifact directory: %q does not exist: %v ", ResultsArtifactDir, err)
	} else {
		config.ArtifactDir = ResultsArtifactDir
	}

	err = rdbLib.UploadTestResults(ctx, config)
	if err != nil {
		return fmt.Errorf("error during rdb test results upload: %w", err)
	}

	// Apply exonerations
	err = rdbLib.ApplyExonerations(ctx, []string{rps.CurrentInvocationId}, test_platform.Request_Params_NON_CRITICAL, nil, nil)
	if err != nil {
		return fmt.Errorf("error during exonerations: %w", err)
	}
	return nil
}

// unpackMetadata unpacks the Any metadata field into PublishGcsMetadata
func unpackMetadata(req *api.PublishRequest) (*metadata.PublishRdbMetadata, error) {
	var m metadata.PublishRdbMetadata
	if err := req.Metadata.UnmarshalTo(&m); err != nil {
		return &m, fmt.Errorf("improperly formatted input proto metadata, %s", err)
	}
	return &m, nil
}

// downloadSources prepares the sources.jsonpb file required by the `rdb`
// command at the given destPath, using information in the given sources proto.
func downloadSources(ctx context.Context, destPath string, sources *metadata.PublishRdbMetadata_Sources) (ok bool, err error) {
	if sources == nil {
		return false, nil
	}

	// Create Google Storage client with default credentials file.
	gsClient, err := storage.NewGSClient(ctx, "")
	if err != nil {
		log.Printf("error while creating new gs client: %s", err)
		return false, fmt.Errorf("creating new gs client: %w", err)
	}

	err = gsClient.DownloadFile(ctx, sources.GsPath, destPath)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			// Not all builds will have source metadata available.
			// This is normal.
			log.Printf("Google storage file %s not found, continuing without sources", sources.GsPath)
			return false, nil
		}
		return false, errors.Annotate(err, "downloading sources.jsonpb").Err()
	}

	sourcesProto := &rdb_pb.Sources{}
	err = common.ReadProtoJSONFile(ctx, destPath, sourcesProto)
	if err != nil {
		return false, errors.Annotate(err, "reading sources.jsonpb").Err()
	}

	// IsDirty indicates whether the source code tested is completely
	// described by the commit and changelists contained in the Sources
	// proto.
	//
	// If the build used other sources (e.g. uncommitted changes,
	// such as a new version of a dependecy) or if the deployment was
	// dirty (e.g. a custom version of Lacros was deployed), this
	// shall be set to true.
	if sources.IsDeploymentDirty {
		sourcesProto.IsDirty = true
	}

	err = common_utils.WriteProtoJsonFile(ctx, destPath, sourcesProto)
	if err != nil {
		return false, errors.Annotate(err, "writing sources.jsonpb").Err()
	}
	return true, nil
}

// validateInvProperties returns false if properties is invalid.
func validateInvProperties(properties *artifact.TestInvocation) bool {
	return proto.Size(properties) <= MaxSizeInvocationProperties
}

// ingestPostProcessResponses ingests the responses of the post process service
// into the test result.
func ingestPostProcessResponses(testResult *artifact.TestResult, postProcessResps *api.RunActivitiesResponse) {
	if postProcessResps == nil {
		log.Printf("post process response is empty")
		return
	}

	log.Printf("start to ingest post process responses: %s", postProcessResps)

	buildMetadata := testResult.GetTestInvocation().GetPrimaryExecutionInfo().GetBuildInfo().GetBuildMetadata()
	if buildMetadata == nil {
		buildMetadata = &artifact.BuildMetadata{}
		testResult.TestInvocation.PrimaryExecutionInfo.BuildInfo.BuildMetadata = buildMetadata
	}

	for _, resp := range postProcessResps.Responses {
		switch resp.GetResponse().(type) {
		case *api.RunActivityResponse_GetFwInfoResponse:
			populateFirmwareInfo(buildMetadata, resp.GetGetFwInfoResponse())
			populateKernelInfo(buildMetadata, resp.GetGetFwInfoResponse())
		case *api.RunActivityResponse_GetGfxInfoResponse:
			populateGfxInfo(buildMetadata, resp.GetGetGfxInfoResponse())
		case *api.RunActivityResponse_GetAvlInfoResponse:
			populateAVLInfo(testResult, resp.GetGetAvlInfoResponse())
		default:
			log.Printf("post process response type: %T is not supported", resp.GetResponse())
		}
	}

	log.Printf("successfully ingested post process responses: %s", postProcessResps)
}

// populateFirmwareInfo populates firmware info and kernel version.
func populateFirmwareInfo(buildMetadata *artifact.BuildMetadata, fwInfoResp *api.GetFWInfoResponse) {
	if fwInfoResp == nil {
		log.Printf("firmware info response is empty")
		return
	}

	log.Printf("start to populate firmware info: %s", fwInfoResp)

	firmware := &artifact.BuildMetadata_Firmware{}
	if fwInfoResp.GetRoFwid() != "" {
		firmware.RoVersion = fwInfoResp.GetRoFwid()
	}
	if fwInfoResp.GetRwFwid() != "" {
		firmware.RwVersion = fwInfoResp.GetRwFwid()
	}
	buildMetadata.Firmware = firmware

	// TODO(b/335363139): Populate GSC devboard tags
	log.Printf("successfully populated firmware info: %s", fwInfoResp)
}

// populateKernelInfo populates kernel info.
func populateKernelInfo(buildMetadata *artifact.BuildMetadata, fwInfoResp *api.GetFWInfoResponse) {
	if fwInfoResp == nil {
		log.Printf("firmware info response is empty")
		return
	}

	log.Printf("start to populate kernel info: %s", fwInfoResp)

	kernel := &artifact.BuildMetadata_Kernel{}
	if fwInfoResp.GetKernelVersion() != "" {
		kernel.Version = fwInfoResp.GetKernelVersion()
	}
	buildMetadata.Kernel = kernel

	// TODO(b/335363139): Populate GSC devboard tags
	log.Printf("successfully populated kernel info: %s", fwInfoResp)
}

// populateGfxInfo populates graphics info.
func populateGfxInfo(buildMetadata *artifact.BuildMetadata, gfxInfoResp *api.GetGfxInfoResponse) {
	if gfxInfoResp == nil {
		log.Printf("graphics info response is empty")
		return
	}

	log.Printf("start to populate graphics info: %s", gfxInfoResp)

	gfxInfo := &artifact.BuildMetadata_GfxInfo{}
	gfxLabels := gfxInfoResp.GetGfxLabels()
	if gfxLabels != nil {
		gfxInfo.DisplayPanelName = gfxLabels["display_panel_name"]
		gfxInfo.DisplayPresentHdr = gfxLabels["display_present_hdr"]
		gfxInfo.DisplayPresentPsr = gfxLabels["display_present_psr"]
		gfxInfo.DisplayPresentVrr = gfxLabels["display_present_vrr"]
		gfxInfo.DisplayRefreshRate = gfxLabels["display_refresh_rate"]
		gfxInfo.DisplayResolution = gfxLabels["display_resolution"]
		gfxInfo.GpuFamily = gfxLabels["gpu_family"]
		gfxInfo.GpuId = gfxLabels["gpu_id"]
		gfxInfo.GpuOpenGlesVersion = gfxLabels["gpu_open_gles_version"]
		gfxInfo.GpuVendor = gfxLabels["gpu_vendor"]
		gfxInfo.GpuVulkanVersion = gfxLabels["gpu_vulkan_version"]
		gfxInfo.PlatformCpuVendor = gfxLabels["platform_cpu_vendor"]
		gfxInfo.PlatformDiskSize, _ = strconv.ParseUint(gfxLabels["platform_disk_size"], 10, 64)
		gfxInfo.PlatformMemorySize, _ = strconv.ParseUint(gfxLabels["platform_memory_size"], 10, 64)
	}
	buildMetadata.GfxInfo = gfxInfo

	log.Printf("successfully populated graphics info: %s", gfxInfoResp)
}

// populateAVLInfo populates AVL qualification info.
func populateAVLInfo(testResult *artifact.TestResult, avlInfoResp *api.GetAvlInfoResponse) {
	if avlInfoResp == nil {
		log.Printf("AVL info response is empty")
		return
	}

	log.Printf("start to populate AVL qual info: %s", avlInfoResp)

	avlInfoMap := avlInfoResp.GetAvlInfos()
	for _, testRun := range testResult.GetTestRuns() {
		testCaseInfo := testRun.GetTestCaseInfo()
		testCaseResult := testCaseInfo.GetTestCaseResult()
		if testCaseResult == nil {
			continue
		}

		avlInfo := avlInfoMap[testCaseResult.GetTestCaseId().GetValue()]
		if avlInfo == nil {
			continue
		}

		testCaseAVLInfo := &artifact.AvlInfo{}
		if avlInfo.GetAvlComponentType() != "" {
			testCaseAVLInfo.AvlComponentType = avlInfo.GetAvlComponentType()
		}
		if avlInfo.GetAvlPartFirmware() != "" {
			testCaseAVLInfo.AvlPartFirmware = avlInfo.GetAvlPartFirmware()
		}
		if avlInfo.GetAvlPartModel() != "" {
			testCaseAVLInfo.AvlPartModel = avlInfo.GetAvlPartModel()
		}
		testCaseInfo.AvlInfo = testCaseAVLInfo
	}

	log.Printf("successfully populated AVL qual info: %s", avlInfoResp)
}
