// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package commands

import (
	"context"
	"log"

	"go.chromium.org/chromiumos/test/provision/v2/foil-provision/service"

	"go.chromium.org/chromiumos/config/go/test/api"

	"go.chromium.org/chromiumos/test/util/adb"
)

type AdbRoot struct {
	ctx            context.Context
	cs             *service.FoilService
	RebootRequired bool
}

func NewAdbRootSetup(ctx context.Context, cs *service.FoilService) *AdbRoot {
	return &AdbRoot{
		ctx: ctx,
		cs:  cs,
	}
}

func (c *AdbRoot) Execute(log *log.Logger) error {
	log.Printf("Start AdbRoot Execute, with reboot")

	addr := c.cs.DutIp
	adb.SetupAdb(log, addr)

	_, err := adb.AdbCmd([]string{"root"}, log)
	if err != nil {
		return err
	}
	adb.SetupAdb(log, addr)
	return adb.SetupAdb(log, addr)
}
func (c *AdbRoot) Revert() error {
	return nil
}

func (c *AdbRoot) GetErrorMessage() string {
	return "failed to AdbRoot"
}

func (c *AdbRoot) GetStatus() api.InstallResponse_Status {
	return api.InstallResponse_STATUS_PROVISIONING_FAILED
}
