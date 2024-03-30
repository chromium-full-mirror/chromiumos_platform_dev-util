// Copyright 2022 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"

	"chromium.googlesource.com/chromiumos/platform/dev-util.git/contrib/fflash/internal/artifact"
)

func tarDecompressionFlag(compression artifact.Compression) string {
	switch compression {
	case artifact.Gzip:
		return "--gzip"
	case artifact.Zstd:
		return "--zstd"
	default:
		log.Panicf("unknown compression %q", compression)
		panic("notreached")
	}
}

// unpackStateful unpacks the compressed stateful partition.
func unpackStateful(ctx context.Context, r io.Reader, compression artifact.Compression) error {
	cmd := exec.CommandContext(ctx,
		"tar",
		"--ignore-command-error",
		"--overwrite",
		"--selinux",
		"--directory", statefulDir,
		tarDecompressionFlag(compression),
		"-xf",
		"-",
	)
	cmd.Stdin = r
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tar unpack failed: %w: %s", err, out)
	}
	return nil
}
