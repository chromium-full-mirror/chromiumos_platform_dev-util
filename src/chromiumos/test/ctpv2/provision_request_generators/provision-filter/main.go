// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"os"

	"go.chromium.org/chromiumos/config/go/test/api"
	dut_api "go.chromium.org/chromiumos/config/go/test/lab/api"

	server "chromiumos/test/ctpv2/common/server_template"

	conf "go.chromium.org/chromiumos/config/go"
)

func executor(req *api.InternalTestplan, log *log.Logger) (*api.InternalTestplan, error) {
	for _, target := range req.GetSuiteInfo().GetSuiteMetadata().GetTargetRequirements() {
		var swBuild string
		var gcsPath string
		var variant string

		for _, swRequirements := range target.GetSwRequirements() {

			// If there is a keyvalue, its not for cros, so continue.
			if len(swRequirements.KeyValues) != 0 {
				continue
			}
			swBuild = swRequirements.GetBuild()
			gcsPath = swRequirements.GetGcsPath()
			variant = swRequirements.GetVariant()
		}

		for _, hwTarget := range target.HwRequirements.HwDefinition {
			switch hw := hwTarget.GetDutInfo().GetDutType().(type) {
			case *dut_api.Dut_Chromeos:
				board := hw.Chromeos.GetDutModel().GetBuildTarget()
				fmt.Println(board)
			default:
				continue
			}
			hwTarget.ProvisionInfo = generateProvisionInfo(hwTarget.GetProvisionInfo(), swBuild, gcsPath, variant)
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
	err := server.Server(executor, "provision_filter")
	if err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}
