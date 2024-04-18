// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package helpers

import (
	"fmt"

	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/test/ctpv2/common/dynamic_updates/builders"
	"go.chromium.org/chromiumos/test/ctpv2/common/dynamic_updates/common"
)

func NewCrosDutContainer(deviceId *common.DeviceIdentifier) *builders.ContainerBuilder {
	dutContainerBuilder := builders.NewContainerBuilder(
		deviceId.GetCrosDutServer(), common.CrosDut, "", "", "",
	)
	dutContainerBuilder.SetCustomTemplate(&api.Template{
		Container: &api.Template_CrosDut{
			CrosDut: &api.CrosDutTemplate{},
		},
	})
	dutContainerBuilder.SetDynamicDeps(map[string]string{
		"crosDut.cacheServer": common.NewPrimaryDeviceIdentifier().GetDevice("dut", "cacheServer", "address"),
		"crosDut.dutAddress":  deviceId.GetDevice("dutServer"),
	})

	return dutContainerBuilder
}

func NewCrosProvisionContainer(ContainerId *common.TaskIdentifier) *builders.ContainerBuilder {
	provisionContainerBuilder := builders.NewContainerBuilder(
		ContainerId.Id, common.CrosProvision, "",
		"/tmp/provisionservice", fmt.Sprintf("%s server -port 0", common.CrosProvision))

	return provisionContainerBuilder
}

func NewAndroidProvisionContainer(taskId *common.TaskIdentifier) *builders.ContainerBuilder {
	container := builders.NewContainerBuilder(
		taskId.Id, "android-provision", "",
		"/tmp/provision",
		"android-provision server -port 0",
	)

	return container
}
