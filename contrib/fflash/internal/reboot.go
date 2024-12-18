// Copyright 2022 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package internal

import (
	"context"
	"fmt"
	"log"
	"time"

	"chromium.googlesource.com/chromiumos/platform/dev-util.git/contrib/fflash/internal/ssh"
)

const BootIdFile = "/proc/sys/kernel/random/boot_id"

// GetBootId returns the unique boot ID of c's remote host.
func GetBootId(ctx context.Context, c *ssh.Client) (string, error) {
	return c.RunSimpleOutput("cat " + BootIdFile)
}

// GetRebootingBootId attempts to connect to host using the system SSH command.
// Then returns the unique boot ID of the host.
func GetRebootingBootId(ctx context.Context, dialer *ssh.Dialer, host string) (string, error) {
	ctx, cancel := context.WithTimeout(
		ctx,
		dialer.SSHOptions().ConnectTimeout+
			2*time.Second, // Give 2 secs for the overhead of the SSH command itself.
	)
	defer cancel()

	cmd := dialer.DefaultCommand(ctx)
	cmd.Args = append(cmd.Args, host, "cat", BootIdFile)
	out, err := cmd.Output()
	return string(out), err
}

// WaitReboot waits the ssh host to reboot
func WaitReboot(ctx context.Context, dialer *ssh.Dialer, host string, oldBootID string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			BootID, err := GetRebootingBootId(ctx, dialer, host)
			if err != nil {
				continue
			}
			if BootID != oldBootID {
				return nil
			}
		case <-ctx.Done():
			return fmt.Errorf("failed to wait for DUT reboot: %s", ctx.Err())
		}
	}
}

// Reboot the device connected with client and wait for the host to be online
func Reboot(ctx context.Context, dialer *ssh.Dialer, client *ssh.Client, host, oldBootID string) error {
	log.Println("rebooting host:", host)
	if _, err := client.RunSimpleOutput("sh -c 'nohup sleep 1 && reboot &'"); err != nil {
		return fmt.Errorf("failed to schedule reboot")
	}
	client.Close()

	return WaitReboot(ctx, dialer, host, oldBootID)
}

// checkBootExpectations checks that the DUT host reboots to wantActiveRootfs.
// On success a new ssh.Client is returned.
func checkBootExpectations(ctx context.Context, dialer *ssh.Dialer, host, wantActiveRootfs string) (*ssh.Client, error) {
	client, err := dialer.DialWithSystemSSH(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("DialWithSystemSSH(): %w", err)
	}

	newParts, err := DetectPartitions(client)
	if err != nil {
		client.Close()
		return nil, err
	}
	log.Println("DUT rebooted to:", newParts.ActiveRootfs())
	if newParts.ActiveRootfs() != wantActiveRootfs {
		client.Close()
		return nil, fmt.Errorf("DUT did not boot to %s", wantActiveRootfs)
	}

	// Check that the DUT is alive for 5 seconds. If it remains alive then assume
	// there are no remaining pending reboots.
	if _, err := client.RunSimpleOutput("sleep 5"); err != nil {
		client.Close()
		return nil, fmt.Errorf("cannot execute ssh command. Maybe DUT rebooted again?")
	}

	return client, nil
}

// CheckedReboot reboots the dut with Reboot() and checkBootExpectations().
// A new SSH client is provided to interact with the rebooted host.
// The passed in client is closed.
func CheckedReboot(ctx context.Context, dialer *ssh.Dialer, client *ssh.Client, host, wantActiveRootfs string) (*ssh.Client, error) {
	defer client.Close()

	bootID, err := GetBootId(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("cannot get current boot ID of DUT %s", bootID)
	}

	if err := Reboot(ctx, dialer, client, host, bootID); err != nil {
		return nil, err
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for i := 1; ; i++ {
		select {
		case <-ticker.C:
			log.Printf("checking boot expectations #%d", i)
			newClient, err := checkBootExpectations(ctx, dialer, host, wantActiveRootfs)
			if err == nil {
				return newClient, nil
			}
			log.Printf("failed checking boot expectations: %s", err)
		case <-ctx.Done():
			return nil, fmt.Errorf("failed to wait for DUT reboot: %s", ctx.Err())
		}
	}
}
