// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"github.com/spf13/cobra"
)

const defaultLocalConfigFilePath = "./wifi_router_config_openwrt.json"

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Commands for reading and updating the release config",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(
		getCmd(),
		setCmd(),
	)

	return cmd
}
