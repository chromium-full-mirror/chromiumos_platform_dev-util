// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"fmt"
	"os/user"
	"path"

	"github.com/spf13/cobra"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/btpeer_manager/dirs"
)

func RootCmd() (*cobra.Command, error) {
	dirContext := &dirs.DirContext{}

	var chromiumosDirPath string
	var workingDirPath string

	usr, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory path: %w", err)
	}

	cmd := &cobra.Command{
		Use:           "btpeer_manager",
		Version:       "0.0.0",
		Short:         "Utility for managing btpeers",
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			srcDir, err := dirs.NewSrcDirectory(chromiumosDirPath)
			if err != nil {
				return fmt.Errorf("failed to get source directies from ChromeOS src dir at %q: %w", chromiumosDirPath, err)
			}
			dirContext.Src = srcDir
			if workingDirPath == "" {
				workingDirPath = dirContext.Src.BTPeerManagerDirPath
			}
			workingDir, err := dirs.NewWorkingDirectory(workingDirPath)
			if err != nil {
				return fmt.Errorf("failed to create working directory at %q: %w", workingDirPath, err)
			}
			dirContext.Working = workingDir
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(
		&chromiumosDirPath,
		"chromiumos_src_dir",
		path.Join(usr.HomeDir, "chromiumos"),
		"Path to local chromiumos source directory",
	)
	cmd.PersistentFlags().StringVar(
		&workingDirPath,
		"working_dir",
		"",
		"Path to base working directory create build directory (defaults to btpeer_manager source dir)",
	)

	cmd.AddCommand(cleanCmd(dirContext))

	return cmd, nil
}
