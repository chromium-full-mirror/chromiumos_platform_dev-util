// Copyright 2022 The ChromiumOS Authors.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package service

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"

	common_utils "go.chromium.org/chromiumos/test/publish/cmd/common-utils"
	"go.chromium.org/chromiumos/test/publish/cmd/publishserver/storage"

	"go.chromium.org/chromiumos/config/go/test/api"
)

type GcsPublishService struct {
	LocalArtifactPath           string
	GcsPath                     string
	ServiceAccountCredsFilePath string
	RetryCount                  int
}

func NewGcsPublishService(req *api.PublishRequest) (*GcsPublishService, error) {
	m, err := unpackMetadata(req)
	if err != nil {
		return nil, err
	}

	if err = common_utils.ValidateGCSPublishRequest(req, m); err != nil {
		return nil, err
	}

	retryCount := 0
	if req.GetRetryCount() > 0 {
		retryCount = int(req.GetRetryCount())
	}

	return &GcsPublishService{
		LocalArtifactPath:           req.GetArtifactDirPath().GetPath(),
		GcsPath:                     m.GetGcsPath().GetPath(),
		RetryCount:                  retryCount,
		ServiceAccountCredsFilePath: m.GetServiceAccountCredsFilePath().GetPath(),
	}, nil
}

func (gs *GcsPublishService) UploadToGS(ctx context.Context) error {
	gsClient, err := storage.NewGSClient(ctx, gs.ServiceAccountCredsFilePath)
	if err != nil {
		log.Printf("error while creating new gs client: %s", err)
		return fmt.Errorf("error while creating new gs client: %s", err)
	}

	if err := gsClient.Upload(ctx, gs.LocalArtifactPath, gs.GcsPath); err != nil {
		log.Printf("error in gcs upload: %s", err)
		return fmt.Errorf("error in gcs upload: %s", err)
	}
	return nil
}

// ArchiveXTSResults uploads all XTS results in the local artifact path to the specific GCS buckets in the metadata.
// TODO: b/379711782 - Propagate errors after process has been stabilized and include the gsClient in constructor.
func (gs *GcsPublishService) ArchiveXTSResults(ctx context.Context, gsClient *storage.GSClient, metadata ArchiveMetadata) error {
	log.Printf("Archiving XTS results for metadata: %v", metadata)

	// Find the root result dir where XTS results are located.
	commonPattern := filepath.Join(gs.LocalArtifactPath, "**", "cros-test", "results")
	var resultDir string
	if metadata.IsALRun {
		// TODO: b/379711782 - Verify and propagate the AL flow logic.
		resultDir = filepath.Join(commonPattern, "tradefed")
	} else {
		resultDir = filepath.Join(commonPattern, "tauto")
	}

	dirs, err := filepath.Glob(resultDir)
	if err != nil {
		fmt.Printf("Error finding test result dir: %s: %s", resultDir, err)
		return nil
	}

	var infos []ArchiveInfo
	for _, dir := range dirs {
		archiveInfos, err := archiveInfo(dir, metadata)
		if err != nil {
			fmt.Printf("Error in archive info for dir %s: %s", dir, err)
			continue
		}
		infos = append(infos, archiveInfos...)
	}

	var wg sync.WaitGroup
	wg.Add(len(infos))
	for _, info := range infos {
		go func(info ArchiveInfo) {
			defer wg.Done()
			log.Printf("Archiving XTS result: %s from local file: %s to GCS bucket: %s", info.Name, info.Source, info.Destination)
			if err := gsClient.Upload(ctx, info.Source, info.Destination); err != nil {
				log.Printf("Error in gcs upload for XTS result: %s from local file: %s to GCS bucket: %s: %s", info.Name, info.Source, info.Destination, err)
			}
		}(info)
	}

	wg.Wait()
	return nil
}

// unpackMetadata unpacks the Any metadata field into PublishGcsMetadata
func unpackMetadata(req *api.PublishRequest) (*api.PublishGcsMetadata, error) {
	var m api.PublishGcsMetadata
	if err := req.Metadata.UnmarshalTo(&m); err != nil {
		return &m, fmt.Errorf("improperly formatted input proto metadata, %s", err)
	}
	return &m, nil
}
