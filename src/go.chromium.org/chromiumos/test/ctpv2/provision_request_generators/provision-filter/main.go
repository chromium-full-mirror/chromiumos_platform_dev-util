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

	server "go.chromium.org/chromiumos/test/ctpv2/common/server_template"

	conf "go.chromium.org/chromiumos/config/go"
)

func executor(req *api.InternalTestplan, log *log.Logger, commonParams *server.CommonFilterParams) (*api.InternalTestplan, error) {
	// Will be the map of hardware which we can make provisionInfo for directly
	foundHW := make(map[string]bool)

	var swRequirements *api.LegacySW

	for _, target := range req.GetSuiteInfo().GetSuiteMetadata().GetSchedulingUnits() {
		primaryHwTarget := target.GetPrimaryTarget().GetSwarmingDef()
		addToFoundCache(primaryHwTarget, foundHW)
		generateProvisionInfo(primaryHwTarget, target.GetPrimaryTarget().GetSwReq(), log)

		for _, t := range target.GetCompanionTargets() {
			generateProvisionInfo(t.GetSwarmingDef(), t.GetSwReq(), log)
			addToFoundCache(t.GetSwarmingDef(), foundHW)
		}
	}

	// TODO (oldProto-azrahman): remove old proto stuffs when schedulingUnits are fully rolled in.
	for _, legacyTarget := range req.GetSuiteInfo().GetSuiteMetadata().GetTargetRequirements() {

		for _, innerTarget := range legacyTarget.GetHwRequirements().GetHwDefinition() {
			swRequirements = legacyTarget.GetSwRequirement()
			generateProvisionInfo(innerTarget, swRequirements, log)
			addToFoundCache(innerTarget, foundHW)

		}
	}

	// Create Dynamic Updates and lookup information.
	if err := GenerateDynamicInfo(req); err != nil {
		return req, err
	}

	return req, nil
}

func addToFoundCache(hwTarget *api.SwarmingDefinition, foundHW map[string]bool) {
	switch hw := hwTarget.GetDutInfo().GetDutType().(type) {
	case *dut_api.Dut_Chromeos:
		board := hw.Chromeos.GetDutModel().GetBuildTarget()
		variant := hwTarget.GetVariant()
		if variant != "" {
			board = fmt.Sprintf("%s-%s", board, variant)
		}
		foundHW[board] = true
	}
}

func generateCrosImageProvisionInfo(target *api.SwarmingDefinition, swReq *api.LegacySW, log *log.Logger) {
	current := target.GetProvisionInfo()
	path := swReq.GetGcsPath()
	log.Printf("generateCrosImageProvisionInfo PATH: %s\n", path)
	log.Printf("generateCrosImageProvisionInfo sw: %s\n", swReq)

	if path != "" {
		installInfo := &api.InstallRequest{
			ImagePath: &conf.StoragePath{
				Path:     path,
				HostType: conf.StoragePath_GS,
			},
		}

		info := &api.ProvisionInfo{
			Type:           api.ProvisionInfo_CROS,
			InstallRequest: installInfo,
		}
		target.ProvisionInfo = append(current, info)

	}
}

// Currently this is simple... just append the new provision info onto the existing.
// We are appending it in case another provsion filter has run and populated info.
func generateProvisionInfo(target *api.SwarmingDefinition, swReq *api.LegacySW, log *log.Logger) {
	switch target.GetDutInfo().GetDutType().(type) {
	case *dut_api.Dut_Chromeos:
		log.Println("chromeos DUT")
		generateCrosImageProvisionInfo(target, swReq, log)
	}

}

func main() {
	err := server.Server(executor, "provision_filter")
	if err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}
