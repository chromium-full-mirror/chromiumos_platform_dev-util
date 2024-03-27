// Copyright 2022 The ChromiumOS Authors.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package service

import (
	"context"
	"fmt"
	"log"
	"path"

	"go.chromium.org/chromiumos/test/publish/clients/rdb_client"
	common_utils "go.chromium.org/chromiumos/test/publish/cmd/common-utils"
	"go.chromium.org/chromiumos/test/publish/cmd/publishserver/storage"
	"go.chromium.org/chromiumos/test/publish/libs/rdb_lib"

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

	// File name for code sources under test
	CodeSourcesJsonFileName = "sources.jsonpb"

	// Result format for rdb
	ResultAdapterResultFormat = "cros-test-result"
)

type RdbPublishService struct {
	RetryCount          int
	CurrentInvocationId string
	TestResultProto     *artifact.TestResult
	TesthausURL         string
	TempDirPath         string
	Sources             *metadata.PublishRdbMetadata_Sources
	BaseVariant         map[string]string
}

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
		RetryCount:          retryCount,
		CurrentInvocationId: m.GetCurrentInvocationId(),
		TestResultProto:     m.GetTestResult(),
		TesthausURL:         m.GetTesthausUrl(),
		Sources:             m.GetSources(),
		BaseVariant:         m.GetBaseVariant(),
	}, nil
}

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

	// Write test result to file
	testResultFilePath := path.Join(rps.TempDirPath, TestResultJsonFileName)
	err := common_utils.WriteProtoJsonFile(ctx, testResultFilePath, rps.TestResultProto)
	if err != nil {
		return fmt.Errorf("error during writing test result proto to file: %w", err)
	} else {
		log.Printf("test result file created in path: %q", testResultFilePath)
	}

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
		artifact := rdb_pb.Artifact{ArtifactId: "testhaus_logs", ContentType: "", Contents: []byte(rps.TesthausURL)}
		err := rdbLib.UploadInvocationArtifacts(ctx, &artifact)
		if err != nil {
			return fmt.Errorf("error during rdb invocation artifact upload: %w", err)
		}
	}

	// Upload test results
	baseTags := map[string]string{}
	config := &rdb_client.RdbStreamConfig{BaseTags: baseTags, BaseVariant: rps.BaseVariant, ResultFile: testResultFilePath, ResultFormat: ResultAdapterResultFormat}
	if hasSources {
		config.SourcesFile = sourcesFilePath
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
	err = common_utils.ReadProtoJsonFile(ctx, destPath, sourcesProto)
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
