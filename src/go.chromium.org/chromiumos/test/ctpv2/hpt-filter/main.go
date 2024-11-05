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
	labapi "go.chromium.org/chromiumos/config/go/test/lab/api"
	libapi "go.chromium.org/chromiumos/test/ctpv2/common/dynamic_updates"
	"go.chromium.org/chromiumos/test/ctpv2/common/dynamic_updates/builders"
	"go.chromium.org/chromiumos/test/ctpv2/common/dynamic_updates/common"
	"go.chromium.org/chromiumos/test/ctpv2/common/dynamic_updates/generators"
	"go.chromium.org/chromiumos/test/ctpv2/common/dynamic_updates/interfaces"
	server "go.chromium.org/chromiumos/test/ctpv2/common/server_template"
	"go.chromium.org/chromiumos/test/publish/cmd/publishserver/storage"
	"go.chromium.org/tast/core/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	artifactsBucket = "e2e-coverage-artifacts"
	versionPath     = "active_version/brya"
	versionFile     = "current.json"
	hptRoot         = "/tmp/cros-hpt"
	authTokenFile   = "/creds/service_accounts/service-account-chromeos.json"
	gsPrefix        = "gs://"
	archiveBucket   = "chromeos-image-archive"
	perfProject     = "google.com:cr-code-cov-merger"
	perfTopic       = "perf-data"
	perfBucket      = "e2e_perf_data"
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

func hptGenerator(av *ActiveVersion) *generators.GenericTaskGenerator {
	hptContainer := builders.NewContainerBuilder(
		"cros-hpt",
		"cros-hpt",
		"",
		"/tmp/cros-hpt",
		"cros-hpt server -port 0",
	)

	// Insert cros-hpt after last provisioning and last test executed step.
	// We need to start and run cros-hpt container after provisioning.
	// Also, we need to stop it after tests have been executed.
	insertInstructions := map[string]interfaces.InsertInstructionGetter{
		"afterProvision": common.InsertActionWrapper(
			api.UpdateAction_Insert_APPEND,
			common.FindLast(api.FocalTaskFinder_PROVISION),
		),
		"afterTest": common.InsertActionWrapper(
			api.UpdateAction_Insert_APPEND,
			common.FindLast(api.FocalTaskFinder_TEST),
		),
	}

	inputs := map[string]proto.Message{
		"authTokenFilePath": structpb.NewStringValue(authTokenFile),
		"board":             structpb.NewStringValue(av.Board),
		"version":           structpb.NewStringValue(av.Version),
		"snapshotVersion":   structpb.NewStringValue(av.SnapshotVersion),
		"projectId":         structpb.NewStringValue(perfProject),
		"topicId":           structpb.NewStringValue(perfTopic),
		"bucket":            structpb.NewStringValue(perfBucket),
		"dutIP":             &labapi.IpEndpoint{},
	}

	// Start cros-hpt container after provisioning and stop after running the tests.
	generator := generators.NewGenericTaskGenerator(
		common.NewTaskIdentifier("hpt").Id,
		"cros-hpt",
		[]*builders.ContainerBuilder{hptContainer},
		insertInstructions,
		&interfaces.GenericTaskMessageRequest{
			InsertionId:  "afterProvision",
			StaticInputs: inputs,
			DynamicInputs: map[string]string{
				"dutIP.value": common.NewPrimaryDeviceIdentifier().GetCrosDutServer(),
			},
		},
		&interfaces.GenericTaskMessageRequest{
			InsertionId: "afterProvision",
		},
		&interfaces.GenericTaskMessageRequest{
			InsertionId: "afterTest",
		},
	)

	return generator
}

func executor(req *api.InternalTestplan, log *log.Logger, commonParams *server.CommonFilterParams) (*api.InternalTestplan, error) {
	ctx := context.Background()
	gsClient, err := storage.NewGSClient(ctx, authTokenFile)
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

	du := req.GetSuiteInfo().GetSuiteMetadata().GetDynamicUpdates()
	if err := libapi.AppendUserDefinedDynamicUpdates(&du, hptGenerator(av).Generate); err != nil {
		return nil, errors.Wrapf(err, "Error adding hpt container.")
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
