// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package release

import (
	"errors"
	"fmt"

	"cloud.google.com/go/storage"
	"github.com/spf13/cobra"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/cmd/common"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/cmd/release/config"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/release"
)

func RootCmd() *cobra.Command {
	managerConfig := &release.ManagerConfig{}

	cmd := &cobra.Command{
		Use:   "release",
		Short: "Commands for managing, releasing, and retrieving released custom OpenWrt images",
		Args:  cobra.NoArgs,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			releaseManager, err := release.NewManager(cmd.Context(), managerConfig)
			if err != nil {
				return fmt.Errorf("failed to initialize a new release manager: %w", err)
			}
			if err := releaseManager.FetchConfig(cmd.Context()); err != nil {
				if err == storage.ErrObjectNotExist {
					configPath, _ := releaseManager.ConfigObject()
					answer, err := common.PromptYesNo(cmd.Context(), fmt.Sprintf("No existing wifi router config found at %q.\n\nWant to initialize a new config file?", configPath), false)
					if err != nil {
						return err
					}
					if !answer {
						return errors.New("aborted by user")
					}
					if err := releaseManager.InitializeNewWifiRouterConfig(cmd.Context(), false); err != nil {
						return fmt.Errorf("failed to initialize new wifi router config: %w", err)
					}
				} else {
					return fmt.Errorf("failed to fetch config: %w", err)
				}
			}
			release.SetSharedManagerInstance(releaseManager)
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(
		&managerConfig.StorageBucketName,
		"bucket",
		release.ChromeOSConnectivityTestArtifactsStorageBucket,
		"GCS storage bucket to use (both prod and test use the same bucket)",
	)
	cmd.PersistentFlags().BoolVarP(
		&managerConfig.UseProdConfig,
		"prod",
		"p",
		false,
		"Uses the production wifi router config when present, or the test config when not present",
	)
	cmd.PersistentFlags().StringVar(
		&managerConfig.GCSCredentialsFilePath,
		"gcloud_cred_file",
		"",
		"The gcloud credential file to use with the GCS API (uses gcloud CLI application-default credentials when unset)",
	)

	cmd.AddCommand(
		config.RootCmd(),
		uploadCmd(),
		downloadCmd(),
	)

	return cmd
}
