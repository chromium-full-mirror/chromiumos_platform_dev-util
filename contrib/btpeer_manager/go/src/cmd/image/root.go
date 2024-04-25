// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package image

import (
	"github.com/spf13/cobra"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/btpeer_manager/cmd/image/release"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/btpeer_manager/dirs"
)

func RootCmd(dirContext *dirs.DirContext, initDirContext func() error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image",
		Short: "Commands related to managing images on btpeers",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(
		release.RootCmd(dirContext, initDirContext),
	)

	return cmd
}
