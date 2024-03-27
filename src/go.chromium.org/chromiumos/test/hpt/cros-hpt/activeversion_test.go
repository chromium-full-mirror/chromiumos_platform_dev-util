// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"go.chromium.org/chromiumos/test/publish/cmd/publishserver/storage"
	"context"
	"fmt"
	"os"
	"testing"
)

const (
	board          = "brya"
	artifactBucket = "e2e-coverage-artifacts"
)

type mreader struct {
	fail bool
}

func (m mreader) Read(ctx context.Context, gsObject storage.GSObject, destFilePath string) error {
	if m.fail {
		return fmt.Errorf("erroring reader as requested")
	}
	f, err := os.Create(destFilePath)
	if err != nil {
		return err
	}

	f.WriteString(`{"version":"R124-15797.0.0-95364","board":"brya","snapshot_version":"gs://snapshot_version/image.zip"}`)
	return nil
}

func TestGet(t *testing.T) {
	testCases := []struct {
		failReader  bool
		name        string
		wantVersion *version
	}{
		{
			name:       "fail reader",
			failReader: true,
		}, {
			name: "pass",
			wantVersion: &version{
				Board:           "brya",
				Version:         "R124-15797.0.0-95364",
				SnapshotVersion: "gs://snapshot_version/",
			},
		},
	}

	for _, tc := range testCases {
		t.Logf("UserCase %s", tc.name)
		av := activeVersion{}
		got, err := av.Get(context.Background(), mreader{
			fail: tc.failReader,
		}, artifactBucket, os.TempDir())
		if tc.failReader && err == nil {
			t.Error("want err got none")
		}
		if !tc.failReader && tc.wantVersion == got {
			t.Errorf("want %s got %v", tc.wantVersion, got)
		}
	}
}
