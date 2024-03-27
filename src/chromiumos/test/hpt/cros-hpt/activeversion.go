// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"go.chromium.org/chromiumos/test/publish/cmd/publishserver/storage"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const (
	activeVersionPath = "active_version"
	versionFile       = "current.json"
	imageZip          = "image.zip"
)

type reader interface {
	Read(ctx context.Context, gsObject storage.GSObject, destFilePath string) error
}

type activeVersion struct{}

type version struct {
	Board           string `json:"board"`
	Version         string `json:"version"`
	SnapshotVersion string `json:"snapshot_version"`
}

// Get returns the active version.
func (a activeVersion) Get(ctx context.Context, reader reader, artifactsBucket, workDir string) (*version, error) {
	localPath := filepath.Join(workDir, versionFile)
	if err := reader.Read(ctx, storage.GSObject{
		Bucket: artifactsBucket,
		// TODO(b/327275996): Replace brya with board in /etc/os-release file on dut.
		Object: filepath.Join(activeVersionPath, brya, versionFile),
	}, localPath); err != nil {
		return nil, err
	}
	contents, err := os.ReadFile(localPath)
	if err != nil {
		return nil, err
	}
	v := &version{}
	if err := json.Unmarshal(contents, v); err != nil {
		return nil, err
	}
	// v.SnapshotVersion has image.zip in the end. Moving forward,
	// we only need the base path without the image.zip.
	v.SnapshotVersion = strings.TrimSuffix(v.SnapshotVersion, imageZip)
	return v, nil
}
