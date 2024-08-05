// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package build

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/google/uuid"
	labapi "go.chromium.org/chromiumos/config/go/test/lab/api"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/cmd/common"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/dirs"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/fileutils"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/log"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/openwrt"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const ImageBuildInfoFilename = "cros_openwrt_image_build_info.json"

type CrosOpenWrtImageBuilderOptions struct {
	BuilderVersion               string
	ChromiumosDirPath            string
	WorkingDirPath               string
	SdkURL                       string
	ImageBuilderURL              string
	AutoURL                      string
	UseExistingSdk               bool
	UseExistingImageBuilder      bool
	UseExisting                  bool
	ImageProfile                 string
	ImageDisabledServices        []string
	IncludedCustomPackages       []string
	IncludedOfficialPackages     []string
	ImagePackageExcludes         []string
	SdkSourcePackageMakefileDirs []string
	ExtraImageName               string
	AutoRetrySdkCompile          bool
	SdkCompileMaxCPUs            int
	SdkConfigOverrides           map[string]string
	ImageFeatureNames            []string
	ImageFeatures                []labapi.WifiRouterFeature
	SkipSdkCompile               bool
}

// String returns the options as a pretty JSON string that can be logged.
func (o *CrosOpenWrtImageBuilderOptions) String() string {
	optionsJSON, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("failed to marshall CrosOpenWrtImageBuilderOptions to JSON: %v", err))
	}
	return string(optionsJSON)
}

// CrosOpenWrtImageBuilder is a utility that can preform different steps related
// to building OpenWrt OS images customized for ChromeOS testing.
type CrosOpenWrtImageBuilder struct {
	src     *dirs.SrcDirectory
	wd      *dirs.WorkingDirectory
	sdk     *openwrt.SdkRunner
	builder *openwrt.ImageBuilderRunner
	options *CrosOpenWrtImageBuilderOptions
}

// NewCrosOpenWrtImageBuilder initializes a CrosOpenWrtImageBuilder and prepares
// its source and working directories for use.
func NewCrosOpenWrtImageBuilder(options *CrosOpenWrtImageBuilderOptions) (*CrosOpenWrtImageBuilder, error) {
	ib := &CrosOpenWrtImageBuilder{
		options: options,
	}

	// Init src dir.
	src, err := dirs.NewSrcDirectory(ib.options.ChromiumosDirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize src directory using chromiumos dir %q: %w", ib.options.ChromiumosDirPath, err)
	}
	ib.src = src

	// Init and clean working dir.
	wd, err := dirs.NewWorkingDirectory(ib.options.WorkingDirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize working directory at %q: %w", ib.options.WorkingDirPath, err)
	}
	ib.wd = wd

	return ib, nil
}

func (ib *CrosOpenWrtImageBuilder) prepareSdk(ctx context.Context) error {
	log.Logger.Println("Preparing OpenWrt SDK for packaging")

	if ib.options.UseExistingSdk {
		log.Logger.Printf("Using existing sdk at %q\n", ib.wd.SdkDirPath)
	} else {
		// Clean old sdk copy.
		if err := ib.wd.CleanSdk(); err != nil {
			return nil
		}

		// Get sdk archive.
		var sdkArchivePath string
		var err error
		if ib.options.SdkURL == "" {
			log.Logger.Println("No OpenWrt SDK archive download url specified, checking for previously download archive")
			sdkArchivePath, err = fileutils.GetLatestFilePathInDir(ib.wd.SdkDownloadsDirPath)
			if err != nil {
				return fmt.Errorf("failed to get latest sdk archive in download directory %q (Specify a new download URL with --sdk_url): %w", ib.wd.SdkDownloadsDirPath, err)
			}
		} else {
			log.Logger.Printf("Downloading OpenWrt SDK archive from %q\n", ib.options.SdkURL)
			sdkArchivePath, err = fileutils.DownloadFileFromURL(ctx, ib.options.SdkURL, ib.wd.SdkDownloadsDirPath, ib.isSnapshotURL(ib.options.SdkURL))
			if err != nil {
				return fmt.Errorf("failed to download sdk archive from %q: %w", ib.options.SdkURL, err)
			}
		}
		log.Logger.Printf("Using OpenWrt SDK archive at %q\n", sdkArchivePath)

		// Unpack sdk archive.
		log.Logger.Printf("Unpacking OpenWrt SDK archive to %q\n", ib.wd.SdkDirPath)
		if err := fileutils.UnpackTarXz(ctx, sdkArchivePath, ib.wd.SdkDirPath); err != nil {
			return err
		}
	}

	// Init sdk runner.
	log.Logger.Println("Initializing OpenWrt SDK runner")
	var err error
	ib.sdk, err = openwrt.NewSdkRunner(ib.wd.SdkDirPath)
	if err != nil {
		return fmt.Errorf("failed to initialize new sdk runner for sdk at %q: %w", ib.wd.SdkDirPath, err)
	}

	log.Logger.Println("Successfully prepared OpenWrt SDK for packaging")
	return nil
}

func (ib *CrosOpenWrtImageBuilder) prepareBuilder(ctx context.Context) error {
	log.Logger.Println("Preparing OpenWrt image builder")

	if ib.options.UseExistingImageBuilder {
		log.Logger.Printf("Using existing image builder at %q\n", ib.wd.ImageBuilderDirPath)
	} else {
		// Get builder archive.
		var builderArchivePath string
		var err error
		if ib.options.ImageBuilderURL == "" {
			log.Logger.Println("No OpenWrt image builder archive download url specified, checking for previously download archive")
			builderArchivePath, err = fileutils.GetLatestFilePathInDir(ib.wd.ImageBuilderDownloadsDirPath)
			if err != nil {
				return fmt.Errorf("failed to get latest image builder archive in download directory %q (Specify a new download URL with --image_builder_url): %w", ib.wd.ImageBuilderDownloadsDirPath, err)
			}
		} else {
			log.Logger.Printf("Downloading OpenWrt image builder archive from %q\n", ib.options.ImageBuilderURL)
			builderArchivePath, err = fileutils.DownloadFileFromURL(ctx, ib.options.ImageBuilderURL, ib.wd.ImageBuilderDownloadsDirPath, ib.isSnapshotURL(ib.options.ImageBuilderURL))
			if err != nil {
				return fmt.Errorf("failed to download image builder archive from %q: %w", ib.options.ImageBuilderURL, err)
			}
		}
		log.Logger.Printf("Using OpenWrt image builder archive at %q\n", builderArchivePath)

		// Unpack builder archive.
		log.Logger.Printf("Unpacking OpenWrt image builder archive to %q\n", ib.wd.ImageBuilderDirPath)
		if err := fileutils.UnpackTarXz(ctx, builderArchivePath, ib.wd.ImageBuilderDirPath); err != nil {
			return fmt.Errorf("failed to unpack image builder archive from %q to %q: %w", builderArchivePath, ib.wd.ImageBuilderDirPath, err)
		}
	}

	// Copy custom-built package files to builder.
	builderPackagesDir := path.Join(ib.wd.ImageBuilderDirPath, "packages")
	if err := fileutils.CopyFilesInDirToDir(ctx, ib.wd.PackagesOutputDirPath, builderPackagesDir); err != nil {
		return err
	}

	// Copy custom files from src to working.
	if err := fileutils.CleanDirectory(ib.wd.ImageBuilderIncludedFilesDirPath); err != nil {
		return err
	}
	if err := fileutils.CopyFilesInDirToDir(ctx, ib.src.IncludedImagesFilesDirPath, ib.wd.ImageBuilderIncludedFilesDirPath); err != nil {
		return err
	}

	// Copy the shared test key to included files as an authorized key so that it
	// may be used for ssh access to the router with openssh-server (sshd).
	sshdUserConfigDir := path.Join(ib.wd.ImageBuilderIncludedFilesDirPath, "root/.ssh")
	if err := os.MkdirAll(sshdUserConfigDir, fileutils.DefaultDirPermissions); err != nil {
		return fmt.Errorf("failed to make dir %q: %w", sshdUserConfigDir, err)
	}
	crosPubKeyPath := path.Join(ib.src.ChromiumosDirPath, "src/third_party/chromiumos-overlay/chromeos-base/chromeos-ssh-testkeys/files/testing_rsa.pub")
	if err := fileutils.CopyFile(ctx, crosPubKeyPath, path.Join(sshdUserConfigDir, "authorized_keys")); err != nil {
		return err
	}

	// Resolve make image run script path.
	runMakeImageScriptPath := path.Join(ib.src.CrosOpenWrtDirPath, "image_builder/run_make_image.sh")

	// Init builder runner.
	log.Logger.Println("Initializing OpenWrt image builder runner")
	var err error
	ib.builder, err = openwrt.NewImageBuilderRunner(ib.wd.ImageBuilderDirPath, runMakeImageScriptPath)
	if err != nil {
		return fmt.Errorf("failed to initialize new image builder runner for image builder at %q: %w", ib.wd.ImageBuilderDirPath, err)
	}

	log.Logger.Println("Successfully prepared OpenWrt image builder")
	return nil
}

func (ib *CrosOpenWrtImageBuilder) isSnapshotURL(openWrtDownloadsURL string) bool {
	return strings.HasPrefix(openWrtDownloadsURL, "https://downloads.openwrt.org/snapshots/")
}

// CompileCustomPackages uses the OpenWrt sdk to compile custom/customized
// OpenWrt packages that may be installed on OpenWrt systems/images.
//
// Note: This also results in all the dependencies of the custom packages to be
// compiled as well, but only specifically chosen packages are saved for image
// building.
//
// Compiled custom package archives are saved to
// dirs.WorkingDirectory.PackagesOutputDirPath.
func (ib *CrosOpenWrtImageBuilder) CompileCustomPackages(ctx context.Context) error {
	if err := ib.prepareSdk(ctx); err != nil {
		return fmt.Errorf("failed to prepare sdk: %w", err)
	}

	log.Logger.Printf("Importing custom package sources from %q\n", ib.src.CustomPackagesDirPath)
	if err := ib.sdk.ImportCustomPackageSources(ctx, ib.src.CustomPackagesDirPath); err != nil {
		return err
	}

	if ib.options.SkipSdkCompile {
		log.Logger.Println("Skipping compilation custom packages (SkipSdkCompile option is true)")
	} else {
		log.Logger.Println("Compiling custom packages and their dependencies")
		if err := ib.sdk.CompileCustomPackages(ctx, ib.options.SdkConfigOverrides, ib.options.SdkSourcePackageMakefileDirs, ib.options.AutoRetrySdkCompile, ib.options.SdkCompileMaxCPUs); err != nil {
			return fmt.Errorf("failed to compile custom packages with sdk: %w", err)
		}
	}

	log.Logger.Printf("Exporting compiled custom package archives to %q\n", ib.wd.PackagesOutputDirPath)
	if err := ib.sdk.ExportCompiledCustomPackageArchives(ctx, ib.options.IncludedCustomPackages, ib.wd.PackagesOutputDirPath); err != nil {
		return fmt.Errorf("failed to export built custom package archives from sdk to %q: %w", ib.wd.PackagesOutputDirPath, err)
	}
	return nil
}

// BuildCustomChromeOSTestImage builds an OpenWrt OS image using the OpenWrt
// image builder that includes customizations for ChromeOS test APs.
//
// These customizations include:
//   - Inclusion of custom-built packages.
//   - Inclusion/exclusion of specific official packages.
//   - Inclusion of custom image files.
//   - A customized image name.
//   - The disabling of some core OpenWrt services that are not needed.
//
// This does not compile custom packages, it just uses precompiled package
// files saved to dirs.WorkingDirectory.PackagesOutputDirPath as overrides
// for packages and then makes adjustments to the package list for the images.
// If an included custom package file is provided, the image builder will use
// that file over an official file for with the same package name, allowing for
// overriding official package builds with customized versions. To compile
// custom packages, call CompileCustomPackages first in either this run or in
// a previous run.
//
// Built images are saved to dirs.WorkingDirectory.ImagesOutputDirPath.
func (ib *CrosOpenWrtImageBuilder) BuildCustomChromeOSTestImage(ctx context.Context) error {
	if err := ib.prepareBuilder(ctx); err != nil {
		return fmt.Errorf("failed to prepare builder: %w", err)
	}

	// Prepare make image args.
	log.Logger.Printf("Preparing image builder arguments")
	makeImageArgs := &openwrt.MakeImageArgs{
		Profile:          ib.options.ImageProfile,
		IncludePackages:  append(ib.options.IncludedCustomPackages, ib.options.IncludedOfficialPackages...),
		ExcludePackages:  ib.options.ImagePackageExcludes,
		Files:            ib.wd.ImageBuilderIncludedFilesDirPath,
		DisabledServices: ib.options.ImageDisabledServices,
		ExtraImageName:   "cros-" + ib.options.BuilderVersion,
	}
	if ib.options.ExtraImageName != "" {
		makeImageArgs.ExtraImageName += "-" + ib.options.ExtraImageName
	}
	if makeImageArgs.Profile == "" {
		log.Logger.Println("Image builder profile not specified with --image_profile, prompting user for selection")
		var err error
		makeImageArgs.Profile, err = ib.promptForImageProfile(ctx)
		if err != nil {
			return fmt.Errorf("failed to prompt user for image profile: %w", err)
		}
	}

	// Collect build info and validate profile selection.
	standardBuildInfo, err := ib.builder.FetchStandardBuildInfoForProfile(ctx, makeImageArgs.Profile)
	if err != nil {
		return fmt.Errorf("failed to fetch standard build info for profile %q: %w", makeImageArgs.Profile, err)
	}

	// Make preliminary image so that we can parse its build files for build summary.
	log.Logger.Println("Making preliminary image")
	err = ib.builder.MakeImage(ctx, makeImageArgs)
	if err != nil {
		return fmt.Errorf("failed to make preliminary image: %w", err)
	}

	// Prepare a build summary and add to included files.
	log.Logger.Println("Preparing image build summary")
	osRelease, err := ib.builder.FetchOSReleaseFromLastImageBuild()
	if err != nil {
		return fmt.Errorf("failed to fetch OSRelease from preliminary image build files: %w", err)
	}
	buildInfo, err := ib.prepareBuildInfo(ctx, makeImageArgs, ib.options.ImageFeatures, osRelease, standardBuildInfo)
	if err != nil {
		return fmt.Errorf("failed to prepare build info: %w", err)
	}
	buildInfoJSON, err := ib.marshallBuildInfoJSON(buildInfo)
	if err != nil {
		return fmt.Errorf("failed to marshall build info: %w", err)
	}
	relativeImageBuildInfoFilePath := "etc/cros/" + ImageBuildInfoFilename
	log.Logger.Printf("Image build info (available on image install at \"/%s\":\n%s\n", relativeImageBuildInfoFilePath, buildInfoJSON)
	if err := fileutils.WriteStringToFile(ctx, buildInfoJSON, path.Join(ib.wd.ImageBuilderIncludedFilesDirPath, relativeImageBuildInfoFilePath)); err != nil {
		return fmt.Errorf("failed to save build info in included image files: %w", err)
	}

	// Generate the ssh banner with build info.
	sshBannerPath := path.Join(ib.wd.ImageBuilderIncludedFilesDirPath, "etc/cros/ssh_banner.txt")
	sshBanner := ib.buildSSHBanner(buildInfo)
	if err := fileutils.WriteStringToFile(ctx, sshBanner, sshBannerPath); err != nil {
		return fmt.Errorf("failed to save ssh banner in included image files: %w", err)
	}

	// Make final image with build summary included.
	log.Logger.Println("Making final image")
	err = ib.builder.MakeImage(ctx, makeImageArgs)
	if err != nil {
		return fmt.Errorf("failed to make image: %w", err)
	}

	// Export image.
	log.Logger.Println("Exporting built image")
	imageDirPath, err := ib.builder.ExportBuiltImage(ctx, ib.wd.ImagesOutputDirPath, map[string][]byte{
		ImageBuildInfoFilename: []byte(buildInfoJSON),
	})
	if err != nil {
		return fmt.Errorf("failed to export built image: %w", err)
	}
	log.Logger.Printf("New %q OpenWrt OS image available at file://%s\n", standardBuildInfo.DeviceName, imageDirPath)

	return nil
}

func (ib *CrosOpenWrtImageBuilder) promptForImageProfile(ctx context.Context) (string, error) {
	// Collect available profiles and sort by profile name.
	log.Logger.Println("Retrieving available profiles for this image builder")
	availProfiles, err := ib.builder.AvailableProfiles(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve available profiles: %w", err)
	}
	var availProfileNamesSorted []string
	for name := range availProfiles {
		availProfileNamesSorted = append(availProfileNamesSorted, name)
	}
	sort.Strings(availProfileNamesSorted)

	// Prompt selection from user until a valid answer is provided or the user
	// aborts. Options are only printed once.
	log.Logger.Printf("Prompting user to choose from one of the %d available image builder profiles\n", len(availProfiles))
	var selectedProfile string
	promptMsg := "\nAvailable image builder profiles:\n\n"
	promptMsg += fmt.Sprintf("%-30s  %-30s\n\n", "PROFILE NAME", "DEVICE DESCRIPTION")
	for _, name := range availProfileNamesSorted {
		description := availProfiles[name]
		promptMsg += fmt.Sprintf("%-30s  %-30s\n", name, description)
	}
	promptMsg += "\nPlease choose from one of the above available image builder profiles.\n"
	fmt.Print(promptMsg)
	for true {
		selectedProfile, err = common.ContextualPrompt(ctx, "Profile: ")
		if err != nil {
			return "", err
		}
		selectedProfile = strings.TrimSpace(selectedProfile)

		if _, ok := availProfiles[selectedProfile]; !ok {
			fmt.Printf("Invalid profile %q\n", selectedProfile)
		} else {
			break
		}
	}
	log.Logger.Printf("Selected image builder profile %q for device %q\n", selectedProfile, availProfiles[selectedProfile])
	return selectedProfile, nil
}

func (ib *CrosOpenWrtImageBuilder) prepareBuildInfo(ctx context.Context, makeImageArgs *openwrt.MakeImageArgs, routerFeatures []labapi.WifiRouterFeature, osRelease *labapi.CrosOpenWrtImageBuildInfo_OSRelease, standardBuildConfig *labapi.CrosOpenWrtImageBuildInfo_StandardBuildConfig) (*labapi.CrosOpenWrtImageBuildInfo, error) {
	buildInfo := &labapi.CrosOpenWrtImageBuildInfo{
		ImageUuid:                      uuid.New().String(),
		CustomImageName:                makeImageArgs.ExtraImageName,
		OsRelease:                      osRelease,
		StandardBuildConfig:            standardBuildConfig,
		RouterFeatures:                 routerFeatures,
		BuildTime:                      timestamppb.Now(),
		CrosOpenwrtImageBuilderVersion: ib.options.BuilderVersion,
		ExtraIncludedPackages:          makeImageArgs.IncludePackages,
		ExcludedPackages:               makeImageArgs.ExcludePackages,
		DisabledServices:               makeImageArgs.DisabledServices,
	}
	// Packages customizations.
	packagesChecksums, err := fileutils.CollectFileChecksums(ctx, path.Join(ib.wd.ImageBuilderDirPath, "packages"))
	if err != nil {
		return nil, err
	}
	buildInfo.CustomPackages = packagesChecksums
	// File customizations.
	includedImageFileChecksums, err := fileutils.CollectFileChecksums(ctx, makeImageArgs.Files)
	if err != nil {
		return nil, err
	}
	buildInfo.CustomIncludedFiles = includedImageFileChecksums
	return buildInfo, nil
}

// marshallBuildInfoJSON returns the buildInfo as a pretty marshalled JSON
// string to allow for easy script parsing and human reading from a terminal.
func (ib *CrosOpenWrtImageBuilder) marshallBuildInfoJSON(buildInfo *labapi.CrosOpenWrtImageBuildInfo) (string, error) {
	marshaller := protojson.MarshalOptions{
		Indent:          "  ",
		Multiline:       true,
		EmitUnpopulated: true,
	}
	buildInfoJSON, err := marshaller.Marshal(buildInfo)
	if err != nil {
		return "", fmt.Errorf("failed to marshal build info to JSON: %w", err)
	}
	return string(buildInfoJSON) + "\n", nil
}

func (ib *CrosOpenWrtImageBuilder) buildSSHBanner(buildInfo *labapi.CrosOpenWrtImageBuildInfo) string {
	return fmt.Sprintf(`
------------------------------------------------------------
ChromeOS OpenWrt Test Router
 * Documentation: go/cros-openwrt
 * Device: %q
 * ImageUUID: %s
------------------------------------------------------------
`,
		buildInfo.GetStandardBuildConfig().GetDeviceName(),
		buildInfo.GetImageUuid(),
	)
}
