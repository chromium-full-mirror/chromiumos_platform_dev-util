// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type mockGSClient struct {
	wantErr     bool
	invalidJSON bool
}

func (m *mockGSClient) DownloadFile(ctx context.Context, gsURL string, localPath string) error {
	if !strings.HasPrefix(gsURL, "gs://") || m.wantErr {
		return fmt.Errorf("error.")
	}

	if m.invalidJSON {
		os.WriteFile(localPath, []byte("foo"), 0644)
		return nil
	}

	content := `{
		"board": "board",
		"version": "R123",
		"snapshot_version": "R123/456"
	}`
	os.WriteFile(localPath, []byte(content), 0644)
	return nil
}

func TestGetVersion(t *testing.T) {
	testCases := []struct {
		name        string
		wantErr     bool
		wantAv      *ActiveVersion
		invalidJSON bool
	}{
		{
			name: "success",
			wantAv: &ActiveVersion{
				Version:         "R123",
				Board:           "board",
				SnapshotVersion: "R123/456",
			},
		}, {
			name:        "invalid_version",
			invalidJSON: true,
			wantErr:     true,
		}, {
			name:    "json_error",
			wantErr: true,
		},
	}
	for _, tc := range testCases {
		t.Logf("Running %s", tc.name)
		m := &mockGSClient{
			wantErr:     tc.wantErr,
			invalidJSON: tc.invalidJSON,
		}
		path := filepath.Join(t.TempDir(), "foo.json")
		av, err := getActiveVersion(context.Background(), m, path)
		if err != nil && !tc.wantErr {
			t.Errorf("Unexpected error: %v", err)
		}

		if !cmp.Equal(av, tc.wantAv) {
			t.Errorf("TestGetVersion() = %+v, want %+v", av, tc.wantAv)
		}
	}
}
