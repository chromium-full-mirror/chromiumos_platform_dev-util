// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package builders

import (
	"fmt"

	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/test/ctpv2/common/dynamic_updates/helpers"
)

func CommonContainerVolumes() []string {
	return []string{
		"/creds:/creds",
	}
}

func NewCrosDutContainer(deviceId *helpers.DeviceIdentifier) *ContainerBuilder {
	dutContainerBuilder := NewContainerBuilder(
		deviceId.GetCrosDutServer(), helpers.CrosDut, "", "", "",
	)
	dutContainerBuilder.SetCustomTemplate(&api.Template{
		Container: &api.Template_CrosDut{
			CrosDut: &api.CrosDutTemplate{},
		},
	})
	dutContainerBuilder.SetDynamicDeps(map[string]string{
		"crosDut.cacheServer": helpers.NewPrimaryDeviceIdentifier().GetDevice("dut", "cacheServer", "address"),
		"crosDut.dutAddress":  deviceId.GetDevice("dutServer"),
	})

	return dutContainerBuilder
}

func NewCrosProvisionContainer(ContainerId *helpers.TaskIdentifier) *ContainerBuilder {
	provisionContainerBuilder := NewContainerBuilder(
		ContainerId.Id, helpers.CrosProvision, "",
		"/tmp/provisionservice", fmt.Sprintf("%s server -port 0", helpers.CrosProvision))

	return provisionContainerBuilder
}
