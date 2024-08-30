// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"go.chromium.org/chromiumos/config/go/test/api"
)

const (
	// This is possible when the DUT has rebooted.
	noPerfStatus = "status:1"

	// Error status 42 means the DUT connection was not alive.
	// which is possible when:
	// 1. DUT is in sleep, suspended or rebooting.
	// 2. transient lab network issues.
	// These are probably intermittent states and not a cause for Error.
	noConnStatus = "status:42"
	needRestart  = "perf need restart"
)

type perfStarter interface {
	startPerf(ctx context.Context) (string, error)
}

type monitor struct {
	exec                execUtil
	perf                perfStarter
	dut                 api.DutServiceClient
	healthCheckInterval time.Duration
	// stopMonitoring chan is used to notify monitor to stop monitoring.
	stopMonitoring chan bool
	// terminated chan is used by the monitor to notify that it has successfully terminated.
	terminated chan bool
}

func (m *monitor) startPerf(ctx context.Context) (string, error) {
	p, err := m.perf.startPerf(ctx)
	if err != nil {
		err = fmt.Errorf("error while starting perf: %v", err)
		log.Println(err)
		return "", err
	}
	return p, nil
}

func (m *monitor) checkStatus(ctx context.Context, pid string) error {
	psResult, err := m.exec.RunCmd(ctx, ps, []string{"-p", pid, "-o", "comm="}, m.dut)
	if err == nil {
		log.Println("perf is healthy.")
		return nil
	} else if strings.Contains(err.Error(), noPerfStatus) {
		log.Println("No running perf found on the DUT. Will need to restart it.")
		return fmt.Errorf(needRestart)
	} else if strings.Contains(err.Error(), noConnStatus) {
		log.Println("Warning received status 42 while during health check.")
		return nil
	} else {
		return fmt.Errorf("error: ps command failed with unexpected error: %v, command result %v", psResult, err)
	}
}

// start will start monitoring the health of perf process.
// If the process is not found, it will restart it.
func (m *monitor) start(pid string) {
	log.Println("Starting to monitoring.")
	ctx := context.Background()
	if pid == "" {
		if p, err := m.startPerf(ctx); err != nil {
			m.terminated <- true
			return
		} else {
			pid = p
		}
	}
	for {
		select {
		case <-m.stopMonitoring:
			log.Println("Stopping monitoring.")
			m.terminated <- true
			return
		case <-time.After(m.healthCheckInterval):
			if err := m.checkStatus(ctx, pid); err != nil {
				if strings.Contains(err.Error(), needRestart) {
					if p, err := m.startPerf(ctx); err != nil {
						m.terminated <- true
						return
					} else {
						pid = p
					}
				} else {
					log.Println(err)
					m.terminated <- true
					return
				}
			}
		}
	}
}
