// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

const (
	pid = "123"
)

type mockPerfCmd struct {
	startPerfCount int
}

func (m *mockPerfCmd) startPerf(ctx context.Context) (string, error) {
	m.startPerfCount++
	return pid, nil
}

func TestStartPerf(t *testing.T) {
	testCases := []struct {
		name               string
		stopDataCollector  bool
		wantStartPerfCount int
		failps             string
		outputps           string
	}{
		{
			name:               "perf terminated on DUT",
			failps:             noPerfStatus,
			wantStartPerfCount: 1,
			stopDataCollector:  true,
		},
		{
			name:              "DUT unreachable",
			failps:            noConnStatus,
			stopDataCollector: true,
		},
		{
			name:   "ps failed with unexpected err",
			failps: "unexpected error",
		},
		{
			name:              "perf is healthy",
			outputps:          pid,
			stopDataCollector: true,
		},
	}
	for _, tc := range testCases {
		t.Logf("\nTestCase %s \n", tc.name)
		mExecUtil := mockExecUtil{
			cmds: []string{},
			output: map[string]string{
				ps: tc.outputps,
			},
			fail: map[string]string{
				ps: tc.failps,
			},
		}
		mPerf := mockPerfCmd{}
		m := monitor{
			healthCheckInterval: time.Second * 1,
			exec:                &mExecUtil,
			perf:                &mPerf,
			stopMonitoring:      make(chan bool),
			terminated:          make(chan bool),
		}
		go m.start(pid)
		if tc.stopDataCollector {
			time.Sleep(1500 * time.Millisecond)
			m.stopMonitoring <- true
		}
		<-m.terminated
		if mPerf.startPerfCount != tc.wantStartPerfCount {
			t.Errorf("want startPerfCount %d got %d", tc.wantStartPerfCount, mPerf.startPerfCount)
		}
		if diff := cmp.Diff(mExecUtil.cmds, []string{ps}); diff != "" {
			t.Errorf("got diff %v, want no diff", diff)
		}
	}
}
