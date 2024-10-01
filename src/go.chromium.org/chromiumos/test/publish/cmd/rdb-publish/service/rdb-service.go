// Copyright 2022 The ChromiumOS Authors.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

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

	// File path for the chromium test results
	ChromiumResultJSONFilePath = "chromium/results/output.json"

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
	testResult := rps.TestResultProto
	ingestPostProcessResponses(testResult, rps.PostProcessResponses)

	// Write test result to file
	testResultFilePath := path.Join(rps.TempDirPath, TestResultJsonFileName)
	err := common_utils.WriteProtoJsonFile(ctx, testResultFilePath, testResult)
	if err != nil {
		return fmt.Errorf("error during writing test result proto to file: %w", err)
	}
	log.Printf("test result file created in path: %q for the test result: %s before uploading to rdb", testResultFilePath, testResult)

	// Write test invocation proto to invocation-level properties file
	testInv := testResult.GetTestInvocation()
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

	// Build the resultdb upload config for test results.
	config := &rdb_client.RdbStreamConfig{}
	baseTags := map[string]string{}
	baseVariant := rps.BaseVariant
	testhausBaseURL := rps.TesthausURL

	if isChromiumTest(testResult) {
		// For Chromium test results
		config, err = chromiumTestRDBConfig(testResult, baseTags, baseVariant, testhausBaseURL, testResultFilePath)
		if err != nil {
			return fmt.Errorf("error during creating the resultdb config for chromium tests: %w", err)
		}
	} else {
		// For ChromeOS test results
		config = &rdb_client.RdbStreamConfig{
			BaseTags:        baseTags,
			BaseVariant:     baseVariant,
			ResultFile:      testResultFilePath,
			ResultFormat:    ResultAdapterResultFormat,
			TesthausBaseURL: testhausBaseURL,
		}
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

	// Upload test results.
	err = rdbLib.UploadTestResults(ctx, config, rps.TestResultProto)
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

// isChromiumTest returns true if the current test run is for chromium test.
func isChromiumTest(testResult *artifact.TestResult) bool {
	if len(testResult.TestRuns) == 0 {
		return false
	}

	// Chromium tests in Skylab set the resultdb_settings flag in the test args.
	// This validation logic is aligned with the both non-CFT and CFT flows in
	// TRv1.
	testArgs := testResult.TestRuns[0].GetExecutionMetadata().GetTestArgs()
	_, ok := testArgs["resultdb_settings"]
	return ok
}

// extractBaseChromiumRDBConfig extracts base resultdb config from test_args for
// chromium test results.
func extractBaseChromiumRDBConfig(testArgs map[string]string) (map[string]interface{}, error) {
	rdbSettingsBytes, err := base64.StdEncoding.DecodeString(testArgs["resultdb_settings"])
	if err != nil {
		return nil, fmt.Errorf("error decoding the resultdb config base64 string for chromium results: %v", err)
	}

	// Unmarshal the JSON bytes into a map
	rdbSettings := map[string]interface{}{}
	err = json.Unmarshal(rdbSettingsBytes, &rdbSettings)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON for chromium resultdb config: %v", err)
	}

	return rdbSettings, nil
}

// chromiumTestRDBConfig creates the resultdb config for chromium tests.
func chromiumTestRDBConfig(testResult *artifact.TestResult, baseTags map[string]string, baseVariant map[string]string, testhausBaseURL string, crosTestResultFile string) (*rdb_client.RdbStreamConfig, error) {
	if len(testResult.TestRuns) == 0 {
		return nil, fmt.Errorf("chromium test result is empty")
	}

	testArgs := testResult.TestRuns[0].GetExecutionMetadata().GetTestArgs()
	chromiumConfig, err := extractBaseChromiumRDBConfig(testArgs)
	if err != nil {
		return nil, err
	}

	for k, v := range chromiumConfig["base_variant"].(map[string]interface{}) {
		baseVariant[k] = v.(string)
	}

	resultFormat := chromiumConfig["result_format"].(string)
	resultFilePath := ""
	if resultFormat == "tast" {
		// For tast, use the default ChromeOS result format and result file.
		resultFormat = ResultAdapterResultFormat
		resultFilePath = crosTestResultFile
	} else {
		// For gtest, use the chromium result format and its result file.
		resultFileDir := testResult.TestRuns[0].GetTestCaseInfo().GetTestCaseResult().GetResultDirPath().GetPath()
		resultFilePath = path.Join(resultFileDir, ChromiumResultJSONFilePath)
	}

	config := &rdb_client.RdbStreamConfig{
		BaseTags:                baseTags,
		BaseVariant:             baseVariant,
		ResultFile:              resultFilePath,
		ResultFormat:            resultFormat,
		TesthausBaseURL:         testhausBaseURL,
		TestIdPrefix:            chromiumConfig["test_id_prefix"].(string),
		CoerceNegativeDuration:  chromiumConfig["coerce_negative_duration"].(bool),
		ExonerateUnexpectedPass: chromiumConfig["exonerate_unexpected_pass"].(bool),
		Include:                 chromiumConfig["include"].(bool),
	}

	log.Printf("successfully created the resultdb config for chromium tests: %v", config)
	return config, nil
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
			populateGSCFirmwareInfo(testResult, resp.GetGetFwInfoResponse())
		case *api.RunActivityResponse_GetGfxInfoResponse:
			populateGfxInfo(buildMetadata, resp.GetGetGfxInfoResponse())
		case *api.RunActivityResponse_GetAvlInfoResponse:
			populateAVLInfo(testResult, resp.GetGetAvlInfoResponse())
		case *api.RunActivityResponse_GetGscInfoResponse:
			populateGSCInfo(testResult, resp.GetGetGscInfoResponse())
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

	log.Printf("successfully populated kernel info: %s", fwInfoResp)
}

// populateGSCFirmwareInfo populates GSC firmware info.
func populateGSCFirmwareInfo(testResult *artifact.TestResult, fwInfoResp *api.GetFWInfoResponse) {
	if fwInfoResp == nil {
		log.Printf("firmware info response is empty")
		return
	}

	log.Printf("start to populate GSC firmware info from: %s", fwInfoResp)

	for _, testRun := range testResult.GetTestRuns() {
		testCaseInfo := testRun.GetTestCaseInfo()
		testCaseResult := testCaseInfo.GetTestCaseResult()
		if testCaseResult == nil {
			continue
		}
		testName := testCaseInfo.TestCaseResult.GetTestCaseId().GetValue()

		// Note that for devboard tests the GSC firmware info here comes from a
		// fake dut, and will be replaced by the values in gsc_info.json which
		// is processed by the populateGSCInfo function below.
		if testCaseInfo.GscInfo != nil {
			log.Printf("Skipped populating GSC firmware info because of existing GSC devboard info: %s for test: %q", testCaseInfo.GscInfo, testName)
			continue
		}

		gscInfo := &artifact.GscInfo{}
		if fwInfoResp.GscRo != "" {
			gscInfo.GscRoVersion = fwInfoResp.GscRo
		}

		if fwInfoResp.GscRw != "" {
			gscInfo.GscRwVersion = fwInfoResp.GscRw
		}
		testCaseInfo.GscInfo = gscInfo

		log.Printf("successfully populated GSC firmware info: %s for test: %q", testCaseInfo.GscInfo, testName)
	}

	log.Printf("successfully populated GSC firmware info from: %s", fwInfoResp)
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

// populateGSCInfo populates GSC devboard info.
func populateGSCInfo(testResult *artifact.TestResult, gscInfoResp *api.GetGscInfoResponse) {
	if gscInfoResp == nil {
		log.Printf("GSC devboard info response is empty")
		return
	}

	log.Printf("start to populate GSC devboard info: %s", gscInfoResp)

	gscInfoMap := gscInfoResp.GetGscInfos()
	for _, testRun := range testResult.GetTestRuns() {
		testCaseInfo := testRun.GetTestCaseInfo()
		testCaseResult := testCaseInfo.GetTestCaseResult()
		if testCaseResult == nil {
			continue
		}

		gscInfoAny := gscInfoMap[testCaseResult.GetTestCaseId().GetValue()]
		if gscInfoAny == nil || len(gscInfoAny.Value) == 0 {
			continue
		}

		gscInfo := &artifact.GscInfo{}
		opts := proto.UnmarshalOptions{DiscardUnknown: true}
		if err := anypb.UnmarshalTo(gscInfoAny, gscInfo, opts); err != nil {
			log.Printf("failed to unmarshal GSC devboard info: %s", gscInfoAny)
			continue
		}
		testCaseInfo.GscInfo = gscInfo
	}

	log.Printf("successfully populated GSC devboard info: %s", gscInfoResp)
}
