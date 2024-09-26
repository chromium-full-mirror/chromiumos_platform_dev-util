// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// DLC constants and helpers
package cross_over

import (
	"context"
	"fmt"
	"log"
	"strings"

	"go.chromium.org/chromiumos/test/dut/cmd/cros-dut/dutssh"
	common_utils "go.chromium.org/chromiumos/test/provision/v2/common-utils"
	"golang.org/x/crypto/ssh"

	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/protobuf/types/known/anypb"
)

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
	config := dutssh.GetSSHConfig()
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
	if strings.Contains(s.params.TargetImagePath.GetPath(), "android-build") {
		installCommand = "al-install chromeos-throw-away-bucket/kimjae/al.bin.img " + cacheServerAdd
	} else {
		installCommand = "cros-install chromeos-image-archive/brya-release/R130-16033.4.0 " + cacheServerAdd
	}
	log.Println("Running command on host now ", installCommand)
	if output, err := session.CombinedOutput(installCommand); err != nil {
		log.Println("Failed to run installCommand: ", installCommand, err)
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
