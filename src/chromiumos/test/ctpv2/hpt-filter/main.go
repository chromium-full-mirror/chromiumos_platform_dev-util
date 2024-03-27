// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	conf "go.chromium.org/chromiumos/config/go"
	"go.chromium.org/chromiumos/config/go/test/api"

	server "chromiumos/test/ctpv2/common/server_template"
	"go.chromium.org/chromiumos/test/publish/cmd/publishserver/storage"
)

const (
	artifactsBucket = "e2e-coverage-artifacts"
	versionPath     = "active_version/brya"
	versionFile     = "current.json"
	hptRoot         = "/tmp/cros-hpt"
	authTokenFile   = "authToken.txt"
	gsPrefix        = "gs://"
	archiveBucket   = "chromeos-image-archive"
)

type ActiveVersion struct {
	Board           string `json:"board"`
	SnapshotVersion string `json:"snapshot_version"`
	Version         string `json:"version"`
}

type GSClient interface {
	DownloadFile(context.Context, string, string) error
}

func getActiveVersion(ctx context.Context, gsClient GSClient, localPath string) (*ActiveVersion, error) {
	gsURL := gsPrefix + filepath.Join(artifactsBucket, versionPath, versionFile)
	if err := gsClient.DownloadFile(ctx, gsURL, localPath); err != nil {
		return nil, err
	}

	contents, err := os.ReadFile(localPath)
	if err != nil {
		return nil, err
	}

	v := &ActiveVersion{}
	if err := json.Unmarshal(contents, v); err != nil {
		log.Printf("Error getting active version: %v", err)
		return nil, err
	}

	return v, nil
}

func executor(req *api.InternalTestplan, log *log.Logger) (*api.InternalTestplan, error) {
	ctx := context.Background()
	gsClient, err := storage.NewGSClient(ctx, filepath.Join(hptRoot, authTokenFile))
	if err != nil {
		return nil, err
	}
	defer gsClient.Close()

	localPath := filepath.Join(hptRoot, versionFile)
	av, err := getActiveVersion(ctx, gsClient, localPath)
	if err != nil {
		return nil, err
	}

	for _, target := range req.GetSuiteInfo().GetSuiteMetadata().GetTargetRequirements() {
		variant := target.GetSwRequirement().GetVariant()

		for _, hwTarget := range target.HwRequirements.HwDefinition {
			log.Printf("Build: %s\nImagePath: %s\nVariant: %s\nBoard: %s\n", av.Version, av.SnapshotVersion, variant, av.Board)
			hwTarget.ProvisionInfo = generateProvisionInfo(hwTarget.GetProvisionInfo(), av.Version, av.SnapshotVersion, variant)
		}
	}

	return req, nil
}

// Currently this is simple... just append the new provision info onto the existing.
// We are appending it in case another provsion filter has run and populated info.
func generateProvisionInfo(current []*api.ProvisionInfo, build string, path string, variant string) []*api.ProvisionInfo {
	installInfo := &api.InstallRequest{
		ImagePath: &conf.StoragePath{
			Path:     path,
			HostType: conf.StoragePath_GS,
		},
	}

	info := &api.ProvisionInfo{
		Type:           api.ProvisionInfo_CROS,
		InstallRequest: installInfo,
		Identifier:     variant,
	}
	return append(current, info)
}

func main() {
	err := server.Server(executor, "hpt_filter")
	if err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}
