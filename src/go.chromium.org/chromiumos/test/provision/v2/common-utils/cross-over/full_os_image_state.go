// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// DLC constants and helpers
package cross_over

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	common_utils "go.chromium.org/chromiumos/test/provision/v2/common-utils"
	"golang.org/x/crypto/ssh"

	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/protobuf/types/known/anypb"
)

var launchTargetPatterns = []*regexp.Regexp{
	regexp.MustCompile(`build_details/P?[0-9]+/([a-z_\-]+)`),
	regexp.MustCompile(`artifacts_list/P?[0-9]+/([a-z_\-]+)`)}

var buildIdPatterns = []*regexp.Regexp{
	regexp.MustCompile(`build_details/(P?[0-9]+)/`),
	regexp.MustCompile(`artifacts_list/(P?[0-9]+)/`)}

// FullOSImageState copies the full Android/Cros image onto DUT disk.
type FullOSImageState struct {
	servodClient api.ServodServiceClient
	params       *CrossOverParameters
}

func NewFullOSImageState(servodClient api.ServodServiceClient, params *CrossOverParameters) common_utils.ServiceState {
	return &FullOSImageState{
		servodClient: servodClient,
		params:       params,
	}
}

func (s FullOSImageState) Execute(ctx context.Context, log *log.Logger) (*anypb.Any, api.InstallResponse_Status, error) {
	log.Println("Executing " + s.Name())
	config := GetSSHConfig()
	dutAddress := fmt.Sprintf("%s:%v", s.params.Dut.GetChromeos().GetSsh().GetAddress(), s.params.Dut.GetChromeos().GetSsh().GetPort())
	var client *ssh.Client
	log.Println("Establishing ssh connection to dut on address: ", dutAddress)
	client, err := ssh.Dial("tcp", dutAddress, config)
	if err != nil {
		log.Println("Failed to tcp dial: ", err)
		return nil, api.InstallResponse_STATUS_DOWNLOADING_IMAGE_FAILED, err
	}
	defer client.Close()
	var session *ssh.Session
	session, err = client.NewSession()
	if err != nil {
		log.Println("Failed to create session: ", err)
		return nil, api.InstallResponse_STATUS_DOWNLOADING_IMAGE_FAILED, err
	}
	cacheServerAdd := s.params.Dut.GetCacheServer().GetAddress().GetAddress()
	installCommand := ""
	targetImgPath := s.params.TargetImagePath.GetPath()
	if strings.Contains(targetImgPath, "android-build") {
		buildId, err := targetBuild(targetImgPath)
		if err != nil {
			return nil, api.InstallResponse_STATUS_INVALID_REQUEST, err
		}
		launchTarget, err := extractTargetLaunch(targetImgPath)
		if err != nil {
			return nil, api.InstallResponse_STATUS_INVALID_REQUEST, err
		}
		installCommand = fmt.Sprintf("al-install android-build/builds/%s/%s/attempts/latest/artifacts/android-desktop_image.bin.gz %s", buildId, launchTarget, cacheServerAdd)
	} else {
		installCommand = fmt.Sprintf("cros-install %s %s", targetImgPath[:5], cacheServerAdd)
	}
	log.Println("Running command on host now ", installCommand)
	if output, err := session.CombinedOutput(installCommand); err != nil {
		log.Println("Failed to run installCommand: ", installCommand, err)
		log.Println(string(output))
		return nil, api.InstallResponse_STATUS_DOWNLOADING_IMAGE_FAILED, err
	} else {
		log.Println(string(output))
	}
	return nil, api.InstallResponse_STATUS_SUCCESS, nil
}

func (s FullOSImageState) Next() common_utils.ServiceState {
	return NewRebootFromDiskState(s.servodClient, s.params)
}

func (s FullOSImageState) Name() string {
	return "Full OS Image State"
}

func extractTargetLaunch(path string) (string, error) {
	for _, re := range launchTargetPatterns {
		matches := re.FindStringSubmatch(path)
		if len(matches) == 2 {
			return matches[1], nil
		}
	}
	return "", fmt.Errorf("could not extract launch from %s", path)
}

func targetBuild(imagePath string) (string, error) {
	for _, re := range buildIdPatterns {
		matches := re.FindStringSubmatch(imagePath)
		if len(matches) == 2 {
			return matches[1], nil
		}
	}
	return "", fmt.Errorf("could not extract buildId from %s", imagePath)
}
