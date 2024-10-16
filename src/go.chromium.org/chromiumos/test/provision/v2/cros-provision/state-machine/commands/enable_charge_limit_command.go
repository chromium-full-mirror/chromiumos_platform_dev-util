// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package commands

import (
	"go.chromium.org/chromiumos/test/provision/v2/cros-provision/service"
	"context"
	"fmt"
	"log"

	"go.chromium.org/chromiumos/config/go/test/api"
)

type EnableChargeLimitCommand struct {
	ctx context.Context
	cs  *service.CrOSService
}

func NewEnableChargeLimitCommand(ctx context.Context, cs *service.CrOSService) *EnableChargeLimitCommand {
	return &EnableChargeLimitCommand{
		ctx: ctx,
		cs:  cs,
	}
}

func (c *EnableChargeLimitCommand) Execute(log *log.Logger) error {
	log.Printf("Start EnableChargeLimitCommand Execute")

	// If this is run on a DUT without a battery (or even just a backup battery
	// for Chromebases), this will effectively do nothing.
	if _, err := c.cs.Connection.RunCmd(c.ctx, "echo", []string{"\"1\"", ">", "/var/lib/power_manager/charge_limit_enabled"}); err != nil {
		return fmt.Errorf("failed to create charge_limit_enabled pref, %s", err)
	}

	if _, err := c.cs.Connection.RunCmd(c.ctx, "echo", []string{"\"3\"", ">", "/var/lib/power_manager/adaptive_charging_hold_delta_percent"}); err != nil {
		return fmt.Errorf("failed to create adaptive_charging_hold_delta_percent pref, %s", err)
	}

	if _, err := c.cs.Connection.RunCmd(c.ctx, "stop", []string{"powerd"}); err != nil {
		log.Printf("failed to stop powerd, %s", err)
	}

	if _, err := c.cs.Connection.RunCmd(c.ctx, "start", []string{"powerd"}); err != nil {
		log_out, log_err := c.cs.Connection.RunCmd(c.ctx, "cat", []string{"/var/log/powerd.out", "/var/log/power_manager/powerd.LATEST"})
		if log_err != nil {
			log.Printf("failed to read powerd logs: %s", log_err)
		} else {
			log.Printf("powerd log contents: %s", log_out)
		}
		return fmt.Errorf("failed to start powerd, %s", err)
	}

	log.Printf("EnableChargeLimitCommand Success")

	return nil
}

func (c *EnableChargeLimitCommand) Revert() error {
	log.Printf("Start EnableChargeLimitCommand Revert")

	if _, err := c.cs.Connection.RunCmd(c.ctx, "rm", []string{"/var/lib/power_manager/charge_limit_enabled"}); err != nil {
		return fmt.Errorf("failed to remove charge_limit_enabled pref, %s", err)
	}

	if _, err := c.cs.Connection.RunCmd(c.ctx, "rm", []string{"/var/lib/power_manager/adaptive_charging_hold_delta_percent"}); err != nil {
		return fmt.Errorf("failed to remove adaptive_charging_hold_delta_percent pref, %s", err)
	}

	if _, err := c.cs.Connection.RunCmd(c.ctx, "stop", []string{"powerd"}); err != nil {
		log.Printf("failed to stop powerd, %s", err)
	}

	if _, err := c.cs.Connection.RunCmd(c.ctx, "start", []string{"powerd"}); err != nil {
		return fmt.Errorf("failed to start powerd, %s", err)
	}

	log.Printf("EnableChargeLimitCommand Revert Success")

	return nil
}

func (c *EnableChargeLimitCommand) GetErrorMessage() string {
	return "failed to get enable Charge Limit"
}

func (c *EnableChargeLimitCommand) GetStatus() api.InstallResponse_Status {
	return api.InstallResponse_STATUS_PROVISIONING_FAILED
}
