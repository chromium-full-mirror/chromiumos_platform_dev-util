// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Container for the FoilProvision state machine
package service

import (
	"fmt"
	"net/url"
	"regexp"

	api1 "go.chromium.org/chromiumos/config/go/test/lab/api"
	common_utils "go.chromium.org/chromiumos/test/provision/v2/common-utils"
	"go.chromium.org/chromiumos/test/provision/v2/common-utils/metadata"

	conf "go.chromium.org/chromiumos/config/go"
	"go.chromium.org/chromiumos/config/go/test/api"
	lab_api "go.chromium.org/chromiumos/config/go/test/lab/api"
)

// FoilService inherits ServiceInterface
type FoilService struct {
	Connection      common_utils.ServiceAdapterInterface
	MachineMetadata metadata.MachineMetadata
	// ImagePath is the android build explorer path.
	// example: android-build/build_explorer/build_details/P78687640/brya-trunk_staging-userdebug/android-desktop-ota-packages.zip
	ImagePath        *conf.StoragePath
	OverwritePayload *conf.StoragePath
	SkipUpdate       bool
	QuickResetDevice bool
	DutIp            string
	UpdateEnginePid  string
	CurrentBuild     string
	TargetBuild      string
	CacheServerUrl   url.URL
}

func NewFoilService(dut *lab_api.Dut, dutClient api.DutServiceClient, req *api.InstallRequest) (*FoilService, error) {

	cacheServerAddr, err := ipEndpointToHostPort(dut.GetCacheServer().GetAddress())
	if err != nil {
		return nil, fmt.Errorf("invalid cache server address %v", err)
	}
	var cacheUrl url.URL
	cacheUrl.Scheme = "http"
	cacheUrl.Host = cacheServerAddr
	imagePath := req.GetImagePath().GetPath()
	build, err := targetBuild(imagePath)
	if err != nil {
		return nil, err
	}

	return &FoilService{
		DutIp: dut.GetChromeos().GetSsh().GetAddress(),
		// TODO: extract targetBuild from the input imagePath.
		TargetBuild:    build,
		CacheServerUrl: cacheUrl,
		ImagePath: &conf.StoragePath{
			Path: imagePath,
		},
	}, nil
}

func targetBuild(imagePath string) (string, error) {
	pattern := `build_details/(P[0-9]+)/`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(imagePath)
	if len(matches) != 2 {
		return "", fmt.Errorf("could not extract buildId from %s", imagePath)
	}
	return matches[1], nil
}

// CleanupOnFailure is called if one of service's states fails to Execute() and
// should clean up the temporary files, and undo the execution, if feasible.
func (c *FoilService) CleanupOnFailure(states []common_utils.ServiceState, executionErr error) error {
	// TODO: evaluate whether cleanup is needed.
	return nil
}

func ipEndpointToHostPort(i *api1.IpEndpoint) (string, error) {
	if len(i.GetAddress()) == 0 {
		return "", fmt.Errorf("IpEndpoint missing address")
	}
	if i.GetPort() == 0 {
		return "", fmt.Errorf("IpEndpoint missing port")
	}
	return fmt.Sprintf("%v:%v", i.GetAddress(), i.GetPort()), nil
}
