// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package release

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/spf13/cobra"
	labapi "go.chromium.org/chromiumos/config/go/test/lab/api"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/cmd/build"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/fileutils"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/log"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/release"
	"google.golang.org/protobuf/encoding/protojson"
)

func uploadCmd() *cobra.Command {
	var overwriteExistingImage bool
	var setAsCurrent bool
	var setAsNext bool
	var overwriteCrosVersion bool

	cmd := &cobra.Command{
		Use:          "upload <min_cros_version> <path_to_builder_output_dir_with_image_archive_and_build_config>",
		Short:        "Uploads a local image to GCS and adds it to the release config",
		Args:         cobra.ExactArgs(2),
		SilenceUsage: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Resolve the min dut version for the new image.
			minCrosVersion := args[0]
			parsedVersion, err := release.ParseChromeOSReleaseVersion(minCrosVersion)
			if err != nil {
				return fmt.Errorf("invalid min_cros_version: %w", err)
			}
			minCrosVersion = parsedVersion.String()

			// Identify local files.
			newLocalImageDir := args[1]
			if err := fileutils.AssertDirectoriesExist(newLocalImageDir); err != nil {
				return err
			}
			buildInfoFilePath := path.Join(newLocalImageDir, build.ImageBuildInfoFilename)
			imageArchiveFilePath, err := findImageArchiveFileInDir(newLocalImageDir)
			if err != nil {
				return err
			}

			// Parse build info file to get info for config.
			buildInfoFileContents, err := fileutils.ReadFile(cmd.Context(), buildInfoFilePath)
			if err != nil {
				return err
			}
			buildInfo := &labapi.CrosOpenWrtImageBuildInfo{}
			if err := protojson.Unmarshal(buildInfoFileContents, buildInfo); err != nil {
				return fmt.Errorf("failed to unmarshal build info from file %q into a CrosOpenWrtImageBuildInfo object: %w", buildInfoFilePath, err)
			}
			if buildInfo.ImageUuid == "" {
				return fmt.Errorf("CrosOpenWrtImageBuildInfo.ImageUuid is empty")
			}
			if buildInfo.StandardBuildConfig == nil || buildInfo.StandardBuildConfig.DeviceName == "" {
				return fmt.Errorf("CrosOpenWrtImageBuildInfo.StandardBuildConfig.DeviceName is empty")
			}

			// Select the corresponding device config.
			releaseManager := release.SharedManagerInstance()
			config := releaseManager.OpenWrtConfig()
			deviceName := buildInfo.StandardBuildConfig.DeviceName
			deviceConfig, ok := config.Openwrt[deviceName]
			if !ok {
				deviceConfig = &labapi.OpenWrtWifiRouterDeviceConfig{}
				config.Openwrt[deviceName] = deviceConfig
			}

			// Check existing images for any incompatibilities, then remove any as allowed.
			for _, existingImageConfig := range deviceConfig.Images {
				if existingImageConfig.MinDutReleaseVersion == minCrosVersion {
					if !overwriteCrosVersion {
						return fmt.Errorf("existing image found (%s) already configured with the same MinDutReleaseVersion (%q); Add the --overwrite_cros_version flag to remove this existing image config or specify a different version", existingImageConfig.ImageUuid, minCrosVersion)
					}
					if deviceConfig.NextImageUuid != "" && existingImageConfig.ImageUuid == deviceConfig.NextImageUuid && !setAsNext {
						return fmt.Errorf("existing image found (%s) already configured with the same MinDutReleaseVersion (%q) and it is the next image; Add the --next flag to remove this existing image config and set this new image as next", existingImageConfig.ImageUuid, minCrosVersion)
					}
					if deviceConfig.CurrentImageUuid != "" && existingImageConfig.ImageUuid == deviceConfig.CurrentImageUuid && !setAsCurrent {
						return fmt.Errorf("existing image found (%s) already configured with the same MinDutReleaseVersion (%q) and it is the current image; Add the --current flag to remove this existing image config and set this new image as current", existingImageConfig.ImageUuid, minCrosVersion)
					}
				}
				if !overwriteExistingImage && existingImageConfig.ImageUuid == buildInfo.ImageUuid {
					imageConfigJSON, err := fileutils.MarshalPrettyJSON(existingImageConfig)
					if err != nil {
						return fmt.Errorf("failed to marshal prexisting image config to JSON: %w", err)
					}
					log.Logger.Printf("Existing image with matching ImageUUID found:\n%s", imageConfigJSON)
					return fmt.Errorf("image found to exist in config already (include --overwrite_existing_image to overwrite it)")
				}
			}
			var newImageConfigs []*labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage
			for _, existingImageConfig := range deviceConfig.Images {
				if existingImageConfig.ImageUuid != buildInfo.ImageUuid && existingImageConfig.MinDutReleaseVersion != minCrosVersion {
					newImageConfigs = append(newImageConfigs, existingImageConfig)
				}
			}
			deviceConfig.Images = newImageConfigs

			// Upload the image.
			imageConfig, err := releaseManager.UploadImageArchive(
				cmd.Context(),
				imageArchiveFilePath,
				deviceName,
				buildInfo.ImageUuid,
				minCrosVersion,
			)
			if err != nil {
				return fmt.Errorf("failed to upload image archive: %w", err)
			}

			// Update the config with new image.
			deviceConfig.Images = append(deviceConfig.Images, imageConfig)
			if setAsCurrent || (deviceConfig.CurrentImageUuid == "" && len(deviceConfig.Images) == 1) {
				deviceConfig.CurrentImageUuid = imageConfig.ImageUuid
			}
			if setAsNext {
				deviceConfig.NextImageUuid = imageConfig.ImageUuid
			}
			if _, err := releaseManager.UpdateOpenWrtConfig(cmd.Context(), config, false); err != nil {
				return fmt.Errorf("failed to save new image updates to config: %w", err)
			}

			// Log image info.
			imageConfigJSON, err := fileutils.MarshalPrettyJSON(imageConfig)
			if err != nil {
				return fmt.Errorf("failed to marshal image config to JSON: %w", err)
			}
			log.Logger.Printf("Successfully uploaded new image for %q OpenWrt devices:\n%s", deviceName, imageConfigJSON)

			log.Logger.Println("See 'cros_openwrt_image_builder release config --help' for ways to update the device config further if needed")

			return nil
		},
	}

	cmd.Flags().BoolVar(
		&overwriteExistingImage,
		"overwrite_existing_image",
		false,
		"Allows overwriting of an image that already exists with the same ImageUUID",
	)
	cmd.Flags().BoolVar(
		&setAsCurrent,
		"current",
		false,
		"Sets the current image to this new image (will be set as current if no other images are configured for device)",
	)
	cmd.Flags().BoolVar(
		&setAsNext,
		"next",
		false,
		"Sets the next image to this new image",
	)
	cmd.Flags().BoolVar(
		&overwriteCrosVersion,
		"overwrite_cros_version",
		false,
		"Allows for replacing other images in the image config with matching min_cros_version values",
	)

	return cmd
}

func findImageArchiveFileInDir(srcDir string) (string, error) {
	globMatcher := "*.tar.xz"
	matchingArchiveFiles, err := filepath.Glob(path.Join(srcDir, globMatcher))
	if err != nil {
		return "", fmt.Errorf("failed to search for an image archive file (matching %s) file in dir %q", globMatcher, srcDir)
	}
	if len(matchingArchiveFiles) != 1 {
		return "", fmt.Errorf("expected to find exactly one image archive file in dir matching %s, but found %d", globMatcher, len(matchingArchiveFiles))
	}
	return matchingArchiveFiles[0], nil
}
