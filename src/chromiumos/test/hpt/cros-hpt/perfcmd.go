// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"log"
	"path/filepath"
	"strings"

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
)

type execUtil interface {
	RunCmd(ctx context.Context, cmd string, args []string, dut api.DutServiceClient) (string, error)
}

type perfCmd struct {
	dut  api.DutServiceClient
	exec execUtil
}

func (p *perfCmd) startPerf(ctx context.Context) (string, error) {
	// Create a dir where the perf.data will be saved.
	args := []string{
		"-p",
		outputPerfDir,
	}
	if _, err := p.exec.RunCmd(ctx, mkdir, args, p.dut); err != nil {
		return "", err
	}
	outputPerfFile := filepath.Join(outputPerfDir, perfData)

	args = []string{
		"-e", // the Performance Monitoring Unit event
		"cycles",
		"--timestamp", // record the sample timestamps
		"-k",          // clock id to use
		"CLOCK_MONOTONIC_RAW",
		"--switch-output=300s", // emit a perf.data file every 5 minute
		"--all-cpus",
		"-F",
		samplingRate,
		"-g", // capture call stack
		"-o", // will generate /tmp/perf/perf.data.<timestamp> files.
		outputPerfFile,
		"&", // run perf record and don't wait for it to return
	}
	pid, err := p.exec.RunCmd(ctx, perf, args, p.dut)
	if err != nil {
		return "", err
	}
	log.Printf("<cros-hpt> perf started with pid %s\n", pid)
	return pid, nil
}

func (p *perfCmd) kill(ctx context.Context, pid string) error {
	args := []string{
		"-SIGTERM",
		pid,
	}
	result, err := p.exec.RunCmd(ctx, kill, args, p.dut)
	if err != nil {
		return err
	}
	log.Printf("<cros-hpt> Successfully stopped perf: %s\n", result)
	return nil
}

func (p *perfCmd) killAll(ctx context.Context) error {
	result, err := p.exec.RunCmd(ctx, pgrep, []string{perf}, p.dut)
	if err != nil {
		return err
	}
	result = strings.TrimSpace(result)
	if result == "" {
		return nil
	}
	for _, pid := range strings.Split(result, "\n") {
		pid = strings.TrimSpace(pid)
		if pid == "" {
			continue
		}
		if err := p.kill(ctx, pid); err != nil {
			return err
		}
	}
	return nil
}
