// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cross_over

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	common_utils "go.chromium.org/chromiumos/test/provision/v2/common-utils"
	"go.chromium.org/chromiumos/test/util/adb"
	"go.chromium.org/chromiumos/test/util/common"

	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/protobuf/types/known/anypb"
)

// ValidateState checks if the correct version was provisioned or not.
type ValidateState struct {
	servodClient api.ServodServiceClient
	params       *CrossOverParameters
}

func NewValidateState(servodClient api.ServodServiceClient, params *CrossOverParameters) common_utils.ServiceState {
	return &ValidateState{
		servodClient: servodClient,
		params:       params,
	}
}

func (s ValidateState) Execute(ctx context.Context, log *log.Logger) (*anypb.Any, api.InstallResponse_Status, error) {
	// TODO: Refactor verfication logic into tow separate commands one for Android and another one for cros.
	log.Println("Executing " + s.Name())
	// TODO: Use TargetImagePath's HostType to identify installation type.
	if strings.Contains(s.params.TargetImagePath.GetPath(), "android-build") {
		log.Println("Performing validation check for AL.")
		dutAddress := fmt.Sprintf("%s:%v", s.params.Dut.GetChromeos().GetSsh().GetAddress(), "22")
		if err := adb.RetrySetupAdb(log, dutAddress, 3*time.Minute); err != nil {
			log.Println("Error while testing adb connection ", err)
			return nil, api.InstallResponse_STATUS_POST_PROVISION_SETUP_FAILED, err
		}

		// As soon as we get the ADB connection, immediately set the keep awake settings.
		c1, err := adb.AdbShellCmd([]string{"svc", "power", "stayon", "true"}, dutAddress, log)
		if err != nil {
			log.Println("Unable to set stay awake:", c1)
			return nil, api.InstallResponse_STATUS_POST_PROVISION_SETUP_FAILED, err
		}
		c2, err := adb.AdbShellCmd([]string{"settings", "put", "global", "stay_on_while_plugged_in", "7"}, dutAddress, log)
		if err != nil {
			log.Println("Unable to set stay awake:", c2)
			return nil, api.InstallResponse_STATUS_POST_PROVISION_SETUP_FAILED, err
		}

		dutVersion, err := adb.AdbShellCmd([]string{"getprop", "ro.system.build.version.incremental"}, dutAddress, log)
		if err != nil {
			return nil, api.InstallResponse_STATUS_POST_PROVISION_SETUP_FAILED, err
		}
		dutVersion = strings.Trim(dutVersion, " \n")
		// TODO: Remove hardcoded version

		if dutVersion != "12445875" {
			err := fmt.Errorf("post provision verification failed. Installed version on dut is %s but want %s", dutVersion, "12445875")
			return nil, api.InstallResponse_STATUS_POST_PROVISION_SETUP_FAILED, err
		}
		if err := adb.TeardownAdb(log, dutAddress); err != nil {
			log.Println("Warning: TeardownAdb failed in ValidateState ", err)
		}
	} else {
		log.Println("Calling force connect on cros-dut")
		if _, err := s.params.DutClient.ForceReconnect(ctx, &api.ForceReconnectRequest{}); err != nil {
			return nil, api.InstallResponse_STATUS_POST_PROVISION_SETUP_FAILED, err
		}

		content, err := common.GetFile(ctx, "/etc/os-release", s.params.DutClient)
		if err != nil {
			log.Printf("Could not fetch /etc/os-release : %s\n", err)
			return nil, api.InstallResponse_STATUS_POST_PROVISION_SETUP_FAILED, err
		}
		log.Println("Content of /etc/os-release ", content)
		// TODO: Remove hardcoded version
		if !strings.Contains(content, "16033.4.0") {
			err = fmt.Errorf("expected version 16033.4.0 but got " + content)
			return nil, api.InstallResponse_STATUS_POST_PROVISION_SETUP_FAILED, err
		}
	}
	return nil, api.InstallResponse_STATUS_SUCCESS, nil
}

func (s ValidateState) Next() common_utils.ServiceState {
	return NewCrossOverTerminateState(s.servodClient, s.params)
}

func (s ValidateState) Name() string {
	return "Validate State"
}
