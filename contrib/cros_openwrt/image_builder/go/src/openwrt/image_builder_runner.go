// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package openwrt

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	labapi "go.chromium.org/chromiumos/config/go/test/lab/api"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/fileutils"
)

type ImageBuilderRunner struct {
	builderDirPath         string
	runMakeImageScriptPath string
	binDirPath             string
	buildDirPath           string
}

type MakeImageArgs struct {
	// Profile specifies the target image to build.
	Profile string

	// IncludePackages is a list of packages to embed into the image.
	IncludePackages []string

	// ExcludePackages is a list of packages to exclude from the image.
	ExcludePackages []string

	// Files is a path to a	directory with custom files to include in the image.
	Files string

	// ExtraImageName is added to the output image filename.
	ExtraImageName string

	// DisabledServices is a list of service names from /etc/init.d to disable.
	DisabledServices []string
}

type ImageBuilderProfileInfo struct {
	OpenWrtRevision  string
	BuildTarget      string
	ProfileName      string
	DeviceName       string
	DefaultPackages  []string
	ProfilePackages  []string
	SupportedDevices []string
}

func NewImageBuilderRunner(builderDirPath, runMakeImageScriptPath string) (*ImageBuilderRunner, error) {
	ib := &ImageBuilderRunner{
		builderDirPath:         builderDirPath,
		runMakeImageScriptPath: runMakeImageScriptPath,
		binDirPath:             path.Join(builderDirPath, "bin"),
		buildDirPath:           path.Join(builderDirPath, "build_dir"),
	}
	if err := fileutils.AssertDirectoriesExist(ib.builderDirPath); err != nil {
		return nil, err
	}
	return ib, nil
}

// MakeImage runs "make image" in the image builder directory with imageArgs.
//
// The "make image" command is run through the run_make_image.sh bash script
// so that its arguments are passed correctly, as it relies on bash to pass
// them in a way that is not supported with golang's exec.
func (ib *ImageBuilderRunner) MakeImage(ctx context.Context, imageArgs *MakeImageArgs) error {
	// Clean local image output and build directories, if they exist.
	binDirExists, err := fileutils.DirectoryExists(ib.binDirPath)
	if err != nil {
		return err
	}
	if binDirExists {
		if err := fileutils.CleanDirectory(ib.binDirPath); err != nil {
			return err
		}
	}

	var runMakeImageArgs []string

	// BUILDER_DIR run script arg, where to run "make image".
	runMakeImageArgs = append(runMakeImageArgs, ib.builderDirPath)

	// PROFILE arg.
	if imageArgs.Profile == "" {
		return errors.New("target image profile is required")
	}
	runMakeImageArgs = append(runMakeImageArgs, imageArgs.Profile)

	// PACKAGES arg.
	packagesArg := imageArgs.IncludePackages
	for _, packageName := range imageArgs.ExcludePackages {
		packagesArg = append(packagesArg, "-"+packageName)
	}
	runMakeImageArgs = append(runMakeImageArgs, strings.Join(packagesArg, " "))

	// FILES arg.
	runMakeImageArgs = append(runMakeImageArgs, imageArgs.Files)

	// EXTRA_IMAGE_NAME arg.
	runMakeImageArgs = append(runMakeImageArgs, imageArgs.ExtraImageName)

	// DISABLED_SERVICES arg.
	runMakeImageArgs = append(runMakeImageArgs, strings.Join(imageArgs.DisabledServices, " "))

	cmd := exec.CommandContext(ctx, "bash", append([]string{ib.runMakeImageScriptPath}, runMakeImageArgs...)...)
	cmd.Dir = ib.builderDirPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run image builder 'make image' with args %v: %w", runMakeImageArgs, err)
	}
	return nil
}

func (ib *ImageBuilderRunner) runMakeInfo(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "make", "info")
	cmd.Dir = ib.builderDirPath
	cmd.Stderr = os.Stderr
	stdout, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run image builder make info: %w", err)
	}
	return string(stdout), nil
}

// AvailableProfiles runs "make info" to retrieve available image profiles this
// image builder supports as a map of profile to its description.
func (ib *ImageBuilderRunner) AvailableProfiles(ctx context.Context) (map[string]string, error) {
	makeInfoOutput, err := ib.runMakeInfo(ctx)
	if err != nil {
		return nil, err
	}

	// Parse out available profiles and device names.
	availProfilesLine := "Available Profiles:"
	profilesStartIndex := strings.Index(makeInfoOutput, availProfilesLine) + len(availProfilesLine)
	if profilesStartIndex >= len(makeInfoOutput) {
		return nil, fmt.Errorf("failed to parse image builder make info output for profiles:\n%s", makeInfoOutput)
	}
	availProfilesOutput := makeInfoOutput[profilesStartIndex:]
	profileAndDescriptionRegex := regexp.MustCompile(`\n([^:]+):\n\s+([^\n]+)\n\s+Packages:`)
	matches := profileAndDescriptionRegex.FindAllStringSubmatch(availProfilesOutput, -1)
	if matches == nil {
		return nil, fmt.Errorf("found no available profiles from image builder make info output:\n%s", availProfilesOutput[0:1000])
	}
	availProfilesToDescription := make(map[string]string)
	for _, profileMatch := range matches {
		availProfilesToDescription[strings.TrimSpace(profileMatch[1])] = strings.TrimSpace(profileMatch[2])
	}
	return availProfilesToDescription, nil
}

func (ib *ImageBuilderRunner) FetchStandardBuildInfoForProfile(ctx context.Context, profileName string) (*labapi.CrosOpenWrtImageBuildInfo_StandardBuildConfig, error) {
	validProfileNameRegex := regexp.MustCompile(`[a-zA-Z-_\d]+`)
	if !validProfileNameRegex.MatchString(profileName) {
		return nil, fmt.Errorf("invalid image profile name %q, must match %s", profileName, validProfileNameRegex.String())
	}
	makeInfoOutput, err := ib.runMakeInfo(ctx)
	if err != nil {
		return nil, err
	}
	return ib.parseStandardBuildInfo(makeInfoOutput, profileName)
}

func (ib *ImageBuilderRunner) parseStandardBuildInfo(makeInfoOutput, profileName string) (*labapi.CrosOpenWrtImageBuildInfo_StandardBuildConfig, error) {
	if !strings.Contains(makeInfoOutput, profileName) {
		return nil, fmt.Errorf("unknown profile %q", profileName)
	}
	matchRegex, err := regexp.Compile(fmt.Sprintf(
		`(?s)^\s*Current Target: "(?P<currentTarget>[^"]+)"\s+`+
			`(Current Architecture: "(?P<currentArchitecture>[^"]+)"\s+)?`+
			`Current Revision: "(?P<currentRevision>[^"]+)"\s+`+
			`Default Packages: (?P<defaultPackagesList>[^\n]*)\n`+
			`Available Profiles:\s+`+
			`.*%s:\n`+
			`\s+(?P<deviceName>[^\n]+)\n`+
			`\s*Packages: (?P<imagePackagesList>[^\n]*)\n`+
			`\s*hasImageMetadata: \d+\n`+
			`\s*SupportedDevices: (?P<supportedDevicesList>[^\n]*)\n`+
			`.*$`,
		profileName,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to compile image info match regex: %w", err)
	}
	match := matchRegex.FindStringSubmatch(makeInfoOutput)
	if match == nil {
		return nil, fmt.Errorf("failed to parse image info from 'make info' output with regex %s", matchRegex.String())
	}
	namedMatchResults := make(map[string]string)
	for i, name := range matchRegex.SubexpNames() {
		if i != 0 && name != "" {
			namedMatchResults[name] = match[matchRegex.SubexpIndex(name)]
		}
	}
	return &labapi.CrosOpenWrtImageBuildInfo_StandardBuildConfig{
		OpenwrtBuildTarget:  namedMatchResults["currentTarget"],
		OpenwrtRevision:     namedMatchResults["currentRevision"],
		BuildTargetPackages: splitAndTrimStringList(namedMatchResults["defaultPackagesList"]),
		BuildProfile:        profileName,
		DeviceName:          namedMatchResults["deviceName"],
		ProfilePackages:     splitAndTrimStringList(namedMatchResults["imagePackagesList"]),
		SupportedDevices:    splitAndTrimStringList(namedMatchResults["supportedDevicesList"]),
	}, nil
}

// ExportBuiltImage exports the image files created by the OpenWrt image builder
// to the specified directory. An archive of the files is also included for
// easy distribution.
//
// Additional data may be saved to the export directory as additional files with
// the extraFileNamesToContent param. These are also included in the archive.
func (ib *ImageBuilderRunner) ExportBuiltImage(ctx context.Context, dstDir string, extraFileNamesToContent map[string][]byte) (string, error) {
	binDirExists, err := fileutils.DirectoryExists(ib.binDirPath)
	if err != nil {
		return "", err
	}
	if !binDirExists {
		return "", fmt.Errorf("image builder bin directory not found at %q", ib.binDirPath)
	}

	// Find directory of the image (i.e. the first directory with a *.manifest file).
	var localImageDir string
	var localImageName string
	const manifestFileSuffix = ".manifest"
	if err := filepath.Walk(ib.binDirPath, func(filePath string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() && strings.HasSuffix(info.Name(), manifestFileSuffix) {
			// Found manifest file.
			localImageDir = path.Dir(filePath)
			localImageName = strings.TrimSuffix(info.Name(), manifestFileSuffix)
		}
		return nil
	}); err != nil {
		return "", err
	}
	if localImageDir == "" || localImageName == "" {
		return "", fmt.Errorf("failed to find local built image in image builder bin dir %q", ib.binDirPath)
	}

	// Build the export directory subdir for this image.
	timestamp := fileutils.BuildTimestampForFilePath(time.Now())
	exportImageDir := path.Join(dstDir, fmt.Sprintf("%s_%s", localImageName, timestamp))
	if err := os.MkdirAll(exportImageDir, fileutils.DefaultDirPermissions); err != nil {
		return "", fmt.Errorf("failed to build image export dir %q: %w", exportImageDir, err)
	}

	// Copy image files to export dir (for local use).
	if err := fileutils.CopyFilesInDirToDir(ctx, localImageDir, exportImageDir); err != nil {
		return "", err
	}

	// Save any extra files.
	for filename, contents := range extraFileNamesToContent {
		filePath := path.Join(exportImageDir, filename)
		if err := fileutils.WriteBytesToFile(ctx, contents, filePath); err != nil {
			return "", fmt.Errorf("failed to write %d bytes to new extra file %q", len(contents), filePath)
		}
	}

	// Create and export an archive of the image (for distribution).
	archivePath := path.Join(exportImageDir, path.Base(exportImageDir)+".tar.xz")
	if err := fileutils.PackageTarXz(ctx, exportImageDir, archivePath); err != nil {
		return "", err
	}

	return exportImageDir, nil
}

func (ib *ImageBuilderRunner) FetchOSReleaseFromLastImageBuild() (*labapi.CrosOpenWrtImageBuildInfo_OSRelease, error) {
	buildDirExists, err := fileutils.DirectoryExists(ib.buildDirPath)
	if err != nil {
		return nil, err
	}
	if !buildDirExists {
		return nil, fmt.Errorf("image build dir %q not found", ib.buildDirPath)
	}
	matchingFiles, err := filepath.Glob(ib.buildDirPath + "/*/*/etc/os-release")
	if err != nil {
		return nil, fmt.Errorf("failed to search for an os-release file in image build dir %q", ib.buildDirPath)
	}
	if len(matchingFiles) == 0 {
		return nil, fmt.Errorf("failed to find an os-release file in image build dir")
	}
	osReleaseFilePath := matchingFiles[0]
	osReleaseFileContents, err := os.ReadFile(osReleaseFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read os-release file %q", osReleaseFilePath)
	}
	return ib.parseOSReleaseFile(string(osReleaseFileContents))
}

func (ib *ImageBuilderRunner) parseOSReleaseFile(osReleaseFileContents string) (*labapi.CrosOpenWrtImageBuildInfo_OSRelease, error) {
	osReleaseEntries := make(map[string]string)
	keyValueRegex := regexp.MustCompile(`(\w+)="([^"]+)"`)
	for _, line := range strings.Split(osReleaseFileContents, "\n") {
		line = strings.TrimSpace(line)
		match := keyValueRegex.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		osReleaseEntries[match[1]] = match[2]
	}
	osRelease := &labapi.CrosOpenWrtImageBuildInfo_OSRelease{}
	var ok bool
	osRelease.Version, ok = osReleaseEntries["VERSION"]
	if !ok {
		return nil, errors.New("missing os-release VERSION")
	}
	osRelease.BuildId, ok = osReleaseEntries["BUILD_ID"]
	if !ok {
		return nil, errors.New("missing os-release BUILD_ID")
	}
	osRelease.OpenwrtBoard, ok = osReleaseEntries["OPENWRT_BOARD"]
	if !ok {
		return nil, errors.New("missing os-release OPENWRT_BOARD")
	}
	osRelease.OpenwrtArch, ok = osReleaseEntries["OPENWRT_ARCH"]
	if !ok {
		return nil, errors.New("missing os-release OPENWRT_ARCH")
	}
	osRelease.OpenwrtRelease, ok = osReleaseEntries["OPENWRT_RELEASE"]
	if !ok {
		return nil, errors.New("missing os-release OPENWRT_RELEASE")
	}
	return osRelease, nil
}

func splitAndTrimStringList(s string) []string {
	items := strings.Split(s, " ")
	var result []string
	for _, item := range items {
		item := strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}
