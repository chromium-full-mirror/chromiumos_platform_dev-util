// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"github.com/spf13/cobra"
)

func RootCmd() (*cobra.Command, error) {
	rootCmd := &cobra.Command{
		Use:           "btpeer_manager",
		Version:       "0.0.0",
		Short:         "Utility for managing btpeers",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	// TODO Add subcommands

	return rootCmd, nil
}
