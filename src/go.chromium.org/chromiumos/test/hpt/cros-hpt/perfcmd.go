// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"go.chromium.org/chromiumos/config/go/test/api"
)

const (
	pgrep         = "pgrep"
	ps            = "ps"
	perf          = "perf"
	mkdir         = "mkdir"
	samplingRate  = "8000"
	kill          = "kill"
	outputPerfDir = "/tmp/perf"
	perfData      = "perf.data"
	nohup         = "nohup"
	perfInitWait  = 1 * time.Second
	exitStatus1   = "status:1"
)

var (
	perfArgs = []string{
		"perf",
		"record",
		"-e", // the Performance Monitoring Unit event
		"cycles",
		"--timestamp",         // record the sample timestamps
		"--switch-output=60s", // emit a perf.data file every 1 minute
		"--all-cpus",
		"-F",
		samplingRate,
		"-o", // will generate /tmp/perf/perf.data.<timestamp> files.
		filepath.Join(outputPerfDir, perfData),
	}
	mkdirArgs = []string{
		"-p",
		outputPerfDir,
	}
)

type execUtil interface {
	RunCmd(ctx context.Context, cmd string, args []string, dut api.DutServiceClient) (string, error)
	RunCmdAsync(ctx context.Context, cmd string, args []string, dut api.DutServiceClient) error
}

type perfCmd struct {
	dut                  api.DutServiceClient
	exec                 execUtil
	perfInitWaitTimeout  time.Duration
	perfInitPollInterval time.Duration
}

func (p *perfCmd) perfId(ctx context.Context) (string, error) {
	// It takes few milliseconds for RunCmdAsync to start the perf process on dut.
	// Need to wait little bit before quering for the perf processID.
	timeout := time.After(p.perfInitWaitTimeout)
	for {
		select {
		case <-time.After(p.perfInitPollInterval):
			pids, err := p.perfPids(ctx)
			if err != nil {
				return "", err
			}
			if len(pids) != 1 {
				return "", fmt.Errorf("expect one perf to be present on dut but got %d pids: %v", len(pids), pids)
			}
			return pids[0], nil
		case <-timeout:
			return "", fmt.Errorf("did not find any perf process on DUT")
		}
	}
}

func (p *perfCmd) startPerf(ctx context.Context) (string, error) {
	// Create a dir where the perf.data will be saved.
	if _, err := p.exec.RunCmd(ctx, mkdir, mkdirArgs, p.dut); err != nil {
		return "", err
	}
	// perf-record is an non-terminating cmd. It blocks the session/terminal
	// and will continue to run until it is signalled to stop.
	// Due to this property we use RunCmdAsync, which is a non blocking way to run the command.
	// When the session used to start perf-record gets terminated(perhaps due to network glich),
	// the corresponding perf process will also get terminated.
	// Using nohup provides robustness as the perf process will continue to run
	// in background even after the session is closed.
	if err := p.exec.RunCmdAsync(ctx, nohup, perfArgs, p.dut); err != nil {
		return "", err
	}
	return p.perfId(ctx)
}

func (p *perfCmd) kill(ctx context.Context, pids ...string) error {
	for _, pid := range pids {
		args := []string{
			"-SIGTERM",
			pid,
		}
		if _, err := p.exec.RunCmd(ctx, kill, args, p.dut); err != nil {
			return err
		}
		log.Printf("Successfully stopped process: %s\n", pid)
	}
	return nil
}

func (p *perfCmd) perfPids(ctx context.Context) ([]string, error) {
	var result []string
	output, err := p.exec.RunCmd(ctx, pgrep, []string{perf, "||", "true"}, p.dut)
	if err != nil && !strings.Contains(err.Error(), exitStatus1) {
		return nil, err
	}
	output = strings.TrimSpace(output)
	for _, pid := range strings.Split(output, "\n") {
		if pid = strings.TrimSpace(pid); pid != "" {
			result = append(result, pid)
		}
	}
	return result, nil
}

func (p *perfCmd) killAll(ctx context.Context) error {
	log.Println("Killing all perf processes on dut.")
	pids, err := p.perfPids(ctx)
	if err != nil {
		return err
	}
	return p.kill(ctx, pids...)
}
