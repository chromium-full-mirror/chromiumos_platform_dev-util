// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package fileutils defines generic file utilities.
package fileutils

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// CopyFile copies the file or directory at srcFilePath to dstPath. If dstPath
// is a directory, the file is placed within it.
func CopyFile(ctx context.Context, srcFilePath string, dstPath string) error {
	cmd := exec.CommandContext(ctx, "cp", "-R", srcFilePath, dstPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy file %q to dir %q: %w", srcFilePath, dstPath, err)
	}
	return nil
}

// ReadFile reads a file's entire contents.
func ReadFile(ctx context.Context, filePath string) ([]byte, error) {
	inFile, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", filePath, err)
	}
	defer func() {
		_ = inFile.Close()
	}()
	inFileReader := NewContextualReaderWrapper(ctx, inFile)
	return io.ReadAll(inFileReader)
}
