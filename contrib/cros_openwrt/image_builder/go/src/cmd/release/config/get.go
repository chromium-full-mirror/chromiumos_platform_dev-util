// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/fileutils"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/log"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/release"
)

func getCmd() *cobra.Command {

	var printOnly bool
	var dstFilePath string

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Downloads the release config from storage",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			releaseManager := release.SharedManagerInstance()
			config := releaseManager.OpenWrtConfig()
			configJson, err := fileutils.MarshalPrettyJSON(config)
			if err != nil {
				return fmt.Errorf("failed to marshal config to JSON: %w", err)
			}
			if printOnly {
				log.Logger.Printf("OpenWrt config:\n%s\n", string(configJson))
			} else {
				if err := fileutils.WriteBytesToFile(cmd.Context(), configJson, dstFilePath); err != nil {
					return err
				}
				absDstPath, err := filepath.Abs(dstFilePath)
				if err != nil {
					return fmt.Errorf("failed to get absolute file path dst file %q", dstFilePath)
				}
				log.Logger.Printf("Saved copy of OpenWrt config to \"file://%s\"\n", absDstPath)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(
		&printOnly,
		"print",
		false,
		"Print the config rather than save it to a file",
	)
	cmd.Flags().StringVar(
		&dstFilePath,
		"dst",
		defaultLocalConfigFilePath,
		"Path to save config to",
	)

	return cmd
}
