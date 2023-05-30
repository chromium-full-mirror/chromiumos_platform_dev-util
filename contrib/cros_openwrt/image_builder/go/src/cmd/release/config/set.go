// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"fmt"

	"github.com/spf13/cobra"
	labapi "go.chromium.org/chromiumos/config/go/test/lab/api"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/fileutils"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/log"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/release"
	"google.golang.org/protobuf/encoding/protojson"
)

func setCmd() *cobra.Command {

	var srcFilePath string
	var forceUpdate bool

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Updates the release config in storage",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			configJson, err := fileutils.ReadFile(cmd.Context(), srcFilePath)
			if err != nil {
				return fmt.Errorf("failed to read config file %q: %w", srcFilePath, err)
			}
			config := &labapi.WifiRouterConfig{}
			if err := protojson.Unmarshal(configJson, config); err != nil {
				return fmt.Errorf("failed to unmarshal config file %q as a WifiRouterConfig: %w", srcFilePath, err)
			}
			config = &labapi.WifiRouterConfig{
				Openwrt: config.Openwrt,
			}
			configJson, err = fileutils.MarshalPrettyJSON(config)
			if err != nil {
				return fmt.Errorf("failed to re-marshal config unmarshed from %q: %w", srcFilePath, err)
			}
			log.Logger.Printf("Successfully read OpenWrt config from %q:\n%s", srcFilePath, configJson)

			releaseManager := release.SharedManagerInstance()
			if _, err := releaseManager.UpdateOpenWrtConfig(cmd.Context(), config, forceUpdate); err != nil {
				return fmt.Errorf("failed to update OpenWrt config: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(
		&srcFilePath,
		"src",
		defaultLocalConfigFilePath,
		"Path to read new config from",
	)
	cmd.Flags().BoolVar(
		&forceUpdate,
		"force",
		false,
		"Bypass config validation failures and forcefully update the config",
	)

	return cmd
}
