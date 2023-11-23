// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package build

import (
	"errors"
	"fmt"
	"os/user"
	"path"
	"strings"

	"github.com/spf13/cobra"
	labapi "go.chromium.org/chromiumos/config/go/test/lab/api"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/cmd/common"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/log"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/openwrt"
)

func RootCmd(rootCmdVersion string) (*cobra.Command, error) {
	builderOptions := &CrosOpenWrtImageBuilderOptions{
		BuilderVersion: rootCmdVersion,
	}
	var possibleImageFeatureNames []string

	usr, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory path: %w", err)
	}

	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "Commands for building custom OpenWrt images.",
		Args:  cobra.NoArgs,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if builderOptions.UseExisting {
				builderOptions.UseExistingSdk = true
				builderOptions.UseExistingImageBuilder = true
			}
			for _, name := range builderOptions.ImageFeatureNames {
				value, ok := labapi.WifiRouterFeature_value[name]
				if !ok {
					return fmt.Errorf(
						"invalid WifiRouterFeature %q, must be one of [%s]",
						name,
						strings.Join(possibleImageFeatureNames, ", "),
					)
				}
				builderOptions.ImageFeatures = append(builderOptions.ImageFeatures, labapi.WifiRouterFeature(value))
			}
			if builderOptions.AutoURL != "" && (!builderOptions.UseExistingSdk && !builderOptions.UseExistingImageBuilder) {
				// Resolve URLs.
				resolvedSdkURL, resolvedImageBuilderURL, err := openwrt.AutoResolveDownloadURLs(cmd.Context(), builderOptions.AutoURL)
				if err != nil {
					return fmt.Errorf("failed to auto resolve download URLs from %q: %w", builderOptions.AutoURL, err)
				}
				if builderOptions.SdkURL == "" {
					builderOptions.SdkURL = resolvedSdkURL
				}
				if builderOptions.ImageBuilderURL == "" {
					builderOptions.ImageBuilderURL = resolvedImageBuilderURL
				}

				// Prompt user for verification.
				promptMsg := fmt.Sprintf(
					"Resolved sdk download URL: %s\nResolved image builder download URL: %s\n\nUse these download URLs?",
					builderOptions.SdkURL,
					builderOptions.ImageBuilderURL,
				)
				answer, err := common.PromptYesNo(cmd.Context(), promptMsg, true)
				if err != nil {
					return err
				}
				if !answer {
					return errors.New("aborted by user")
				}
			}
			log.Logger.Printf("CrosOpenWrtImageBuilderOptions: %s", builderOptions)
			return nil
		},
	}

	buildCmd.PersistentFlags().StringVar(
		&builderOptions.ChromiumosDirPath,
		"chromiumos_src_dir",
		path.Join(usr.HomeDir, "chromiumos"),
		"Path to local chromiumos source directory.",
	)
	buildCmd.PersistentFlags().StringVar(
		&builderOptions.WorkingDirPath,
		"working_dir",
		"/tmp/cros_openwrt",
		"Path to working directory to store downloads, sdk, image builder, and built packages and images.",
	)
	buildCmd.PersistentFlags().StringVar(
		&builderOptions.SdkURL,
		"sdk_url",
		"",
		"URL to download the sdk archive from. Leave unset to use the last downloaded sdk.",
	)
	buildCmd.PersistentFlags().StringVar(
		&builderOptions.ImageBuilderURL,
		"image_builder_url",
		"",
		"URL to download the image builder archive from. Leave unset to use the last downloaded image builder.",
	)
	buildCmd.PersistentFlags().StringVar(
		&builderOptions.AutoURL,
		"auto_url",
		"",
		"Download URL to use to auto resolve unset --sdk_url and --image_builder_url values from.",
	)
	buildCmd.PersistentFlags().BoolVar(
		&builderOptions.UseExistingSdk,
		"use_existing_sdk",
		false,
		"Use sdk in working directory as-is (must exist).",
	)
	buildCmd.PersistentFlags().BoolVar(
		&builderOptions.UseExistingSdk,
		"use_existing_image_builder",
		false,
		"Use image builder in working directory as-is (must exist).",
	)
	buildCmd.PersistentFlags().BoolVar(
		&builderOptions.UseExisting,
		"use_existing",
		false,
		"Shortcut to set both --use_existing_sdk and --use_existing_image_builder.",
	)

	// Optimization options.
	buildCmd.PersistentFlags().BoolVar(
		&builderOptions.AutoRetrySdkCompile,
		"disable_auto_sdk_compile_retry",
		true,
		"Include to disable the default behavior of retrying the compilation of custom packages once if the first attempt fails.",
	)
	buildCmd.PersistentFlags().IntVar(
		&builderOptions.SdkCompileMaxCPUs,
		"sdk_compile_max_cpus",
		-1,
		"The maximum number of CPUs to use for custom package compilation. Values less than 1 indicate that all available CPUs may be used.",
	)

	// OpenWrt OS customization options.
	buildCmd.PersistentFlags().StringVar(
		&builderOptions.ExtraImageName,
		"extra_image_name",
		"",
		"A custom name to add to the image, added as a suffix to existing names.",
	)
	buildCmd.PersistentFlags().StringVar(
		&builderOptions.ImageProfile,
		"image_profile",
		"",
		"The profile to use with the image builder when making images. Leave unset to prompt for selection based off of available profiles.",
	)
	buildCmd.PersistentFlags().StringToStringVar(
		&builderOptions.SdkConfigOverrides,
		"sdk_config",
		map[string]string{},
		"Config options to set for the sdk when compiling custom packages.",
	)
	buildCmd.PersistentFlags().StringSliceVar(
		&builderOptions.SdkSourcePackageMakefileDirs,
		"sdk_make",
		[]string{},
		"The sdk package makefile paths to use to compile custom IPKs. Making a package with the sdk will build all the IPKs that package depends upon, but only need to be included if they are expected to differ from official versions.",
	)
	buildCmd.PersistentFlags().StringSliceVar(
		&builderOptions.ImageDisabledServices,
		"disable_service",
		[]string{},
		"Services to disable in the built image.",
	)
	buildCmd.PersistentFlags().StringSliceVar(
		&builderOptions.IncludedCustomPackages,
		"include_custom_package",
		[]string{},
		"Names of packages that should be included in built images that are built using a local sdk and included in the image builder as custom IPKs. Only custom packages in this list are saved from sdk package compilation.",
	)
	buildCmd.PersistentFlags().StringSliceVar(
		&builderOptions.IncludedOfficialPackages,
		"include_official_package",
		[]string{
			"openssh-server",
		},
		"Names of packages that should be included in built images that are downloaded from official OpenWrt repositories.",
	)
	buildCmd.PersistentFlags().StringSliceVar(
		&builderOptions.ImagePackageExcludes,
		"exclude_package",
		[]string{
			"dropbear",
		},
		"Packages to exclude from the built image.",
	)

	possibleImageFeatureNames = make([]string, len(labapi.WifiRouterFeature_name))
	for i, name := range labapi.WifiRouterFeature_name {
		possibleImageFeatureNames[i] = name
	}
	buildCmd.PersistentFlags().StringSliceVar(
		&builderOptions.ImageFeatureNames,
		"image_feature",
		[]string{},
		fmt.Sprintf("Wifi router features this image supports for testing (possible features: [%s])", strings.Join(possibleImageFeatureNames, ", ")),
	)
	buildCmd.PersistentFlags().BoolVar(
		&builderOptions.SkipSdkCompile,
		"skip_sdk_compile",
		false,
		"Include to skip the sdk compilation step. Useful when IPKs have been compiled in a previous run and you are just debugging which need to be exported.",
	)

	buildCmd.AddCommand(
		allCmd(builderOptions),
		imageCmd(builderOptions),
		packagesCmd(builderOptions),
		cleanCmd(builderOptions),
	)

	return buildCmd, nil
}
