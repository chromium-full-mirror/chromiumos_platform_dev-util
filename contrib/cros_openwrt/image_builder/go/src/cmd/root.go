// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"github.com/spf13/cobra"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/cmd/build"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/cmd/release"
)

func RootCmd() (*cobra.Command, error) {

	rootCmd := &cobra.Command{
		Use:           "cros_openwrt_image_builder",
		Version:       "1.4.0",
		Short:         "Utility for building custom OpenWrt OS images with custom compiled packages",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	// Add subcommands
	buildCmd, err := build.RootCmd(rootCmd.Version)
	if err != nil {
		return nil, err
	}
	rootCmd.AddCommand(
		buildCmd,
		release.RootCmd(),
	)

	return rootCmd, nil
}
