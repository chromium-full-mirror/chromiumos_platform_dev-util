// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"chromiumos/test/publish/cmd/publishserver/storage"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.chromium.org/chromiumos/config/go/test/api"
)

const bucket = "bucket"

type mockFileFetcher struct {
	copyCount   int
	copyFile    bool
	failOnCount int
}

func (m *mockFileFetcher) Copy(ctx context.Context, file string, dut api.DutServiceClient) (string, error) {
	m.copyCount++
	if m.copyFile || m.failOnCount == m.copyCount {
		return "", fmt.Errorf("failing Copy")
	}

	localFilePath := filepath.Join(os.TempDir(), filepath.Base(file))
	err := os.WriteFile(localFilePath, []byte("content"), 0644)
	if err != nil {
		fmt.Printf("\nError in osWriteFile localFilePath %s: %v", localFilePath, err)
		return "", err
	}
	return localFilePath, nil
}

type mockTar struct {
	tarCount int
}

func (m *mockTar) Tar(ctx context.Context, srcFile, destTarFile string) error {
	m.tarCount++
	return nil
}

type mockStorage struct {
	failWrite bool
}

func (m *mockStorage) Write(ctx context.Context, filePath string, gsObject storage.GSObject) error {
	if m.failWrite {
		return fmt.Errorf("failing write")
	}
	return nil
}

type mockPublisher struct {
	failPublish  bool
	publishCount int
}

func (m *mockPublisher) Publish(ctx context.Context, data []byte) error {
	m.publishCount++
	if m.failPublish {
		return fmt.Errorf("failing publish")
	}
	return nil
}

func TestDataCollectorStart(t *testing.T) {
	testCases := []struct {
		name               string
		execFail           map[string]string
		execOutput         map[string]string
		wantExecCmds       []string
		wantCopyCount      int
		failGetFile        bool
		wantTarCount       int
		failWrite          bool
		failPublish        bool
		wantPublishCount   int
		getFileFailOnCount int
		stop               bool
	}{
		{
			name: "ls fail",
			execFail: map[string]string{
				ls: "ls failed",
			},
			wantExecCmds:  []string{ls},
			wantCopyCount: 0,
		},
		{
			name: "GetFile fail",
			execOutput: map[string]string{
				ls: `
				/tmp/perf.data.1
				/tmp/perf.data.2

				`,
			},
			execFail:      map[string]string{},
			wantExecCmds:  []string{ls},
			wantCopyCount: 1,
			failGetFile:   true,
			wantTarCount:  0,
		},
		{
			name: "fail storage write",
			execOutput: map[string]string{
				ls: `
				/tmp/perf.data.1
				/tmp/perf.data.2

				`,
			},
			execFail:      map[string]string{},
			wantExecCmds:  []string{ls},
			wantCopyCount: 1,
			wantTarCount:  1,
			failWrite:     true,
		},
		{
			name: "fail publish",
			execOutput: map[string]string{
				ls: `
				/tmp/perf.data.1
				/tmp/perf.data.2

				`,
			},
			execFail:         map[string]string{},
			wantExecCmds:     []string{ls},
			wantCopyCount:    1,
			wantTarCount:     1,
			failPublish:      true,
			wantPublishCount: 1,
		},
		{
			name: "fail rm",
			execOutput: map[string]string{
				ls: `
				/tmp/perf.data.1
				/tmp/perf.data.2

				`,
			},
			execFail: map[string]string{
				rm: "rm failed",
			},
			wantExecCmds:     []string{ls, rm},
			wantCopyCount:    1,
			wantTarCount:     1,
			wantPublishCount: 1,
		},
		{
			name: "One Successful loop",
			execOutput: map[string]string{
				ls: `
				/tmp/perf.data.1
				/tmp/perf.data.2

				`,
			},
			wantExecCmds:       []string{ls, rm, rm, ls},
			wantCopyCount:      3,
			wantTarCount:       2,
			wantPublishCount:   2,
			getFileFailOnCount: 3,
		},
		{
			name: "Request to Stop",
			execOutput: map[string]string{
				ls: `
				/tmp/perf.data.1
				/tmp/perf.data.2

				`,
			},
			wantExecCmds:     []string{ls, rm, rm},
			wantCopyCount:    2,
			wantTarCount:     2,
			wantPublishCount: 2,
			stop:             true,
		},
	}
	for _, tc := range testCases {
		fmt.Printf("\n\ntestcase : %v", tc.name)
		mFileFetcher := mockFileFetcher{
			copyFile:    tc.failGetFile,
			failOnCount: tc.getFileFailOnCount,
		}
		mExecUtil := mockExecUtil{
			fail:   tc.execFail,
			cmds:   []string{},
			args:   [][]string{},
			output: tc.execOutput,
		}
		mTarUtil := mockTar{}
		mStorage := mockStorage{
			failWrite: tc.failWrite,
		}
		mPublisher := mockPublisher{failPublish: tc.failPublish}
		dc := perfCollector{
			fileFetcher:           &mFileFetcher,
			exec:                  &mExecUtil,
			perfCollectorInterval: time.Second * 1,
			tarUtil:               &mTarUtil,
			storage:               &mStorage,
			bucket:                bucket,
			publisher:             &mPublisher,
			stopCollector:         make(chan bool),
			terminated:            make(chan bool),
		}
		go dc.start(context.Background(), "perfDir")
		if tc.stop {
			time.Sleep(1 * time.Second)
			dc.stopCollector <- true
		}
		_ = <-dc.terminated
		if mFileFetcher.copyCount != tc.wantCopyCount {
			t.Errorf("\nCopy want %d got %d", tc.wantCopyCount, mFileFetcher.copyCount)
		}
		if diff := cmp.Diff(mExecUtil.cmds, tc.wantExecCmds); diff != "" {
			t.Errorf("\ngot diff %v, want no diff", diff)
		}
		if mTarUtil.tarCount != tc.wantTarCount {
			t.Errorf("\nTarCount want %d got %d", tc.wantTarCount, mTarUtil.tarCount)
		}
		if mPublisher.publishCount != tc.wantPublishCount {
			t.Errorf("\nPublishCount want %d got %d", tc.wantPublishCount, mPublisher.publishCount)
		}
	}
}
