// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package release

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	labapi "go.chromium.org/chromiumos/config/go/test/lab/api"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/cmd/common"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/fileutils"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/log"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/release"
)

func downloadCmd() *cobra.Command {
	var deviceName string
	var dstDir string
	var selectNextImage bool
	var selectByDutHostname string
	var selectByDutCrosReleaseVersion string
	var selectByImageUUID string

	cmd := &cobra.Command{
		Use:   "download <device_name>",
		Short: "Downloads a released image",
		Long:  "Downloads a released image configured for the specified device. The current image is downloaded by default, but specific images may be downloaded by using the --uuid, --dut, ---cros_version, or --next flags.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deviceName = args[0]

			// Select the corresponding device config.
			releaseManager := release.SharedManagerInstance()
			deviceConfig, err := releaseManager.OpenWrtRouterDeviceConfig(deviceName)
			if err != nil {
				return err
			}
			deviceConfigJSON, err := fileutils.MarshalPrettyJSON(deviceConfig)
			if err != nil {
				return fmt.Errorf("failed to marshal device config to JSON: %w", err)
			}
			log.Logger.Printf("OpenWrtRouterDeviceConfig for device %q:\n%s", deviceName, deviceConfigJSON)

			// Select the image.
			requireUserConfirmation := false
			var imageConfig *labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage
			if selectByImageUUID != "" {
				imageConfig, err = release.SelectImageByUUID(deviceConfig, selectByImageUUID)
			} else if selectByDutHostname != "" {
				requireUserConfirmation = true
				imageConfig, err = release.SelectImageForDut(deviceConfig, selectByDutHostname, selectByDutCrosReleaseVersion)
			} else if selectByDutCrosReleaseVersion != "" {
				requireUserConfirmation = true
				imageConfig, err = release.SelectImageByCrosReleaseVersion(deviceConfig, selectByDutCrosReleaseVersion, false)
			} else if selectNextImage {
				imageConfig, err = release.SelectNextImage(deviceConfig)
			} else {
				imageConfig, err = release.SelectCurrentImage(deviceConfig)
			}
			if err != nil {
				return fmt.Errorf("failed to select image for device %q: %w", deviceName, err)
			}
			imageConfigJSON, err := fileutils.MarshalPrettyJSON(imageConfig)
			if err != nil {
				return fmt.Errorf("failed to marshal image config to JSON: %w", err)
			}
			log.Logger.Printf("Selected image:\n%s", imageConfigJSON)
			if requireUserConfirmation {
				answer, err := common.PromptYesNo(cmd.Context(), "Download the selected image?", true)
				if err != nil {
					return err
				}
				if !answer {
					return errors.New("aborted by user")
				}
			}

			// Download the selected image.
			if _, err := releaseManager.DownloadImageArchive(cmd.Context(), imageConfig, dstDir); err != nil {
				return fmt.Errorf("failed to download image %q: %w", imageConfig.ImageUuid, err)
			}
			return err
		},
	}

	cmd.Flags().StringVar(
		&dstDir,
		"dst",
		".",
		"Directory to download image archive file to",
	)
	cmd.Flags().StringVar(
		&selectByImageUUID,
		"uuid",
		"",
		"Download the image configured for this device with a matching ImageUUID (case-insensitive)",
	)
	cmd.Flags().StringVar(
		&selectByDutHostname,
		"dut",
		"",
		"Download the next image if the dut is in the verification pool, or the current image otherwise",
	)
	cmd.Flags().StringVar(
		&selectByDutCrosReleaseVersion,
		"cros_version",
		"",
		"Download the image that has the highest MinDutCrosReleaseVersion that is equal to or less than this CHROMEOS_RELEASE_VERSION (compatible with --dut)",
	)
	cmd.Flags().BoolVar(
		&selectNextImage,
		"next",
		false,
		"Download the next image for this device",
	)
	return cmd
}
