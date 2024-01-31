// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"os"

	"go.chromium.org/chromiumos/config/go/test/api"

	server "chromiumos/test/ctpv2/common/server_template"

	conf "go.chromium.org/chromiumos/config/go"
)

func executor(req *api.InternalTestplan, log *log.Logger) (*api.InternalTestplan, error) {
	for _, target := range req.GetSuiteInfo().GetSuiteMetadata().GetTargetRequirements() {
		// Replace with a call to GCS to get these dynamically.
		swBuild := "R123-15764.0.0-94145"
		gcsPath := "gs://chromeos-image-archive/brya-snapshot/R123-15764.0.0-94145-8757387554446140817/image.zip"
		var variant string

		for _, swRequirements := range target.GetSwRequirements() {

			// If there is a keyvalue, its not for cros, so continue.
			if len(swRequirements.KeyValues) != 0 {
				continue
			}
			variant = swRequirements.GetVariant()
		}

		for _, hwTarget := range target.HwRequirements.HwDefinition {
			board := "brya"
			fmt.Printf("Build: %s\nImagePath: %s\nVariant: %s\nBoard: %s\n", swBuild, gcsPath, variant, board)
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
	err := server.Server(executor, "hpt_filter")
	if err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}
