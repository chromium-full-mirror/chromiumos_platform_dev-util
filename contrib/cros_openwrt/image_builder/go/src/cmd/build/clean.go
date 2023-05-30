// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package build

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/dirs"
)

func cleanCmd(builderOptions *CrosOpenWrtImageBuilderOptions) *cobra.Command {
	var cleanAll bool

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Deletes temporary files.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stat(builderOptions.WorkingDirPath); os.IsNotExist(err) {
				return fmt.Errorf("working directory %q does not exist", builderOptions.WorkingDirPath)
			}
			wd, err := dirs.NewWorkingDirectory(builderOptions.WorkingDirPath)
			if err != nil {
				return fmt.Errorf("failed to initialize working directory at %q: %w", builderOptions.WorkingDirPath, err)
			}
			if cleanAll {
				if err := wd.CleanAll(); err != nil {
					return fmt.Errorf("failed to fully remove working directory at %q: %w", builderOptions.WorkingDirPath, err)
				}
			} else {
				if err := wd.CleanIntermediaryFiles(); err != nil {
					return fmt.Errorf("failed to remove intermediary files from working directory at %q: %w", builderOptions.WorkingDirPath, err)
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(
		&cleanAll,
		"all",
		false,
		"Fully delete working directory (sdk and image builder downloads, intermediary files, built packages, built images)",
	)

	return cmd
}
