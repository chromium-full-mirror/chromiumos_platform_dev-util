// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package commands

import (
	"context"
	"log"
	"strings"

	"go.chromium.org/chromiumos/test/provision/v2/foil-provision/service"

	"go.chromium.org/chromiumos/config/go/test/api"

	"go.chromium.org/chromiumos/test/util/adb"
)

type GetEnginePid struct {
	ctx            context.Context
	cs             *service.FoilService
	RebootRequired bool
}

func NewGetEnginePidSetup(ctx context.Context, cs *service.FoilService) *GetEnginePid {
	return &GetEnginePid{
		ctx: ctx,
		cs:  cs,
	}
}

func (c *GetEnginePid) Execute(log *log.Logger) error {
	log.Printf("Start GetEnginePid Execute, with reboot")
	lcpid := lcPid(log)
	c.cs.UpdateEnginePid = lcpid

	return nil
}

func lcPid(log *log.Logger) string {
	pidOut, _ := adb.AdbCmd([]string{"shell", "pgrep", "update_engine"}, log)

	pf := strings.ReplaceAll(strings.ReplaceAll(pidOut, " ", ""), "\n", "")
	log.Println("PID FOUND: ", pf)
	return pf

}

func (c *GetEnginePid) Revert() error {
	return nil
}

func (c *GetEnginePid) GetErrorMessage() string {
	return "failed to GetEnginePid"
}

func (c *GetEnginePid) GetStatus() api.InstallResponse_Status {
	return api.InstallResponse_STATUS_PROVISIONING_FAILED
}
