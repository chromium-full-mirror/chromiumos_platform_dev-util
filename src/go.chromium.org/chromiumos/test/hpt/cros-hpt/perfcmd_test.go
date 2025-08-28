// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.chromium.org/chromiumos/config/go/test/api"
)

type mockExecUtil struct {
	fail   map[string]string
	output map[string]string
	cmds   []string
	args   [][]string
}

func (m *mockExecUtil) RunCmdAsync(ctx context.Context, cmd string, args []string, dut api.DutServiceClient) error {
	if m.fail != nil {
		if v, ok := m.fail[cmd]; ok && v != "" {
			return fmt.Errorf("%s", "failing " + cmd + " " + v)
		}
	}
	return nil
}

func (m *mockExecUtil) RunCmd(ctx context.Context, cmd string, args []string, dut api.DutServiceClient) (string, error) {
	if m.cmds != nil {
		m.cmds = append(m.cmds, cmd)
	}
	if m.args != nil {
		m.args = append(m.args, args)
	}
	if m.output != nil {
		if v, ok := m.output[cmd]; ok && v != "" {
			return v, nil
		}
	}
	if m.fail != nil {
		if v, ok := m.fail[cmd]; ok && v != "" {
			return "", errors.New("failing " + cmd + " " + v)
		}
	}
	return "", nil
}

func TestStart(t *testing.T) {
	testCases := []struct {
		name        string
		wantErr     bool
		wantCmds    []string
		pgrepOutput string
		mkdirFail   string
		nohupFail   string
	}{
		{
			name:      "Fail mkdir",
			mkdirFail: "fail mkdir",
			wantErr:   true,
			wantCmds:  []string{mkdir},
		}, {

			name:      "Fail nohup",
			wantErr:   true,
			wantCmds:  []string{mkdir},
			nohupFail: "nohupFail",
		},
		{
			name:        "Multiple perf",
			wantCmds:    []string{mkdir, pgrep},
			wantErr:     true,
			pgrepOutput: "1234 \n 987",
		},
		{
			name:        "Pass",
			wantCmds:    []string{mkdir, pgrep},
			pgrepOutput: pid,
		},
	}
	for _, tc := range testCases {
		t.Log("TestCase " + tc.name)
		m := mockExecUtil{
			fail: map[string]string{
				mkdir: tc.mkdirFail,
				nohup: tc.nohupFail,
			},
			cmds: []string{},
			output: map[string]string{
				pgrep: tc.pgrepOutput,
			},
		}
		p := perfCmd{
			exec:                 &m,
			perfInitPollInterval: 10 * time.Millisecond,
			perfInitWaitTimeout:  1 * time.Second,
		}
		if _, got := p.startPerf(context.Background()); tc.wantErr && got == nil {
			t.Error("got none want error")
		} else if !tc.wantErr && got != nil {
			t.Errorf("got %v want none", got)
		}
		if diff := cmp.Diff(m.cmds, tc.wantCmds); diff != "" {
			t.Errorf("got diff %v, want no diff", diff)
		}
	}
}

func TestKill(t *testing.T) {
	testCases := []struct {
		name     string
		wantErr  bool
		wantCmds []string
		fail     map[string]string
	}{
		{
			name:     "Pass",
			wantCmds: []string{kill},
		},
		{
			name:     "Fail",
			wantErr:  true,
			wantCmds: []string{kill},
			fail: map[string]string{
				kill: "kill failed",
			},
		},
	}
	for _, tc := range testCases {
		m := mockExecUtil{
			cmds: []string{},
			fail: tc.fail,
		}
		p := perfCmd{
			exec:                 &m,
			perfInitPollInterval: 10 * time.Millisecond,
			perfInitWaitTimeout:  1 * time.Second,
		}
		if err := p.kill(context.Background(), "1"); tc.wantErr && err == nil {
			t.Error("got no error want error")
		} else if !tc.wantErr && err != nil {
			t.Errorf("got error %v want no error", err)
		}
		if diff := cmp.Diff(m.cmds, tc.wantCmds); diff != "" {
			t.Errorf("got diff %v, want no diff", diff)
		}
	}
}

func TestKillAllFail(t *testing.T) {
	testCases := []struct {
		name     string
		wantErr  bool
		wantCmds []string
		fail     map[string]string
		output   map[string]string
		wantArgs [][]string
	}{
		{

			name: "Fail",
			fail: map[string]string{
				pgrep: "pgrep failed",
			},
			wantErr:  true,
			wantCmds: []string{pgrep},
			wantArgs: [][]string{
				{perf, "||", "true"},
			},
		},
		{
			name: "Pass",
			output: map[string]string{
				pgrep: `
				123
				456
				789

				`,
			},
			wantCmds: []string{pgrep, kill, kill, kill},
			wantArgs: [][]string{
				{perf, "||", "true"},
				{"-SIGTERM", "123"},
				{"-SIGTERM", "456"},
				{"-SIGTERM", "789"},
			},
		},
	}
	for _, tc := range testCases {
		t.Log("TestCase " + tc.name)
		m := mockExecUtil{
			fail:   tc.fail,
			cmds:   []string{},
			args:   [][]string{},
			output: tc.output,
		}
		p := perfCmd{
			exec: &m,
		}
		if got := p.killAll(context.Background()); tc.wantErr && got == nil {
			t.Errorf("got none want error")
		} else if !tc.wantErr && got != nil {
			t.Errorf("got error %v want no error", got)
		}
		if diff := cmp.Diff(m.cmds, tc.wantCmds); diff != "" {
			t.Errorf("got diff %v, want no diff", diff)
		}
		if diff := cmp.Diff(m.args, tc.wantArgs); diff != "" {
			t.Errorf("got diff %v, want no diff", diff)
		}
	}
}
