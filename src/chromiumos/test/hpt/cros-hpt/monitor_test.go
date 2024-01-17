// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

const (
	pid   = "123"
	ONE   = 1
	THREE = 3
	FOUR  = 4
	ZERO  = 0
)

type mockMonitorPerfCmd struct {
	startPerfCount int
}

func (m *mockMonitorPerfCmd) startPerf(ctx context.Context) (string, error) {
	m.startPerfCount++
	return pid, nil
}

func TestStartPerf(t *testing.T) {
	testCases := []struct {
		name               string
		wantStartPerfCount int
		mockExecUtil       mockExecUtil
		stopDataCollector  bool
		wantExecCmds       []string
	}{
		{
			name: "perf terminated on DUT",
			mockExecUtil: mockExecUtil{
				fail: map[string]string{
					ps: noPerfStatus,
				},
			},
			wantExecCmds: []string{ps},

			wantStartPerfCount: ONE,
			stopDataCollector:  true,
		},
		{
			name: "DUT unreachable",
			mockExecUtil: mockExecUtil{
				fail: map[string]string{
					ps: noConnStatus,
				},
			},
			wantExecCmds: []string{ps},

			wantStartPerfCount: ZERO,
			stopDataCollector:  true,
		},
		{
			name: "ps failed with unexpected err",
			mockExecUtil: mockExecUtil{
				fail: map[string]string{
					ps: "unexpected error",
				},
			},
			wantExecCmds:       []string{ps},
			wantStartPerfCount: ZERO,
		},
		{
			name: "perf is healthy",
			mockExecUtil: mockExecUtil{
				output: map[string]string{
					ps: pid,
				},
			},
			wantExecCmds:       []string{ps},
			wantStartPerfCount: ZERO,
			stopDataCollector:  true,
		},
	}
	for _, tc := range testCases {
		log.Printf("\nTestCase %s \n", tc.name)
		mExecUtil := mockExecUtil{
			cmds:   []string{},
			output: tc.mockExecUtil.output,
			fail:   tc.mockExecUtil.fail,
		}
		mPerf := mockMonitorPerfCmd{}
		m := monitor{
			healthCheckInterval: time.Second * ONE,
			exec:                &mExecUtil,
			perf:                &mPerf,
			stopMonitoring:      make(chan bool),
			terminated:          make(chan bool),
		}
		go m.start(context.Background(), pid)
		if tc.stopDataCollector {
			time.Sleep(1 * time.Second)
			m.stopMonitoring <- true
		}
		_ = <-m.terminated
		if mPerf.startPerfCount != tc.wantStartPerfCount {
			t.Errorf("want startPerfCount %d got %d", tc.wantStartPerfCount, mPerf.startPerfCount)
		}
		if diff := cmp.Diff(mExecUtil.cmds, tc.wantExecCmds); diff != "" {
			t.Errorf("got diff %v, want no diff", diff)
		}
	}
}
