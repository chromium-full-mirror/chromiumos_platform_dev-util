// Copyright 2022 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package openwrt

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/fileutils"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/log"
)

type SdkRunner struct {
	sdkDirPath     string
	packageDirPath string
	scriptsDirPath string
	config         map[string]string
}

func NewSdkRunner(sdkDirPath string) (*SdkRunner, error) {
	sdk := &SdkRunner{
		sdkDirPath:     sdkDirPath,
		packageDirPath: path.Join(sdkDirPath, "package"),
		scriptsDirPath: path.Join(sdkDirPath, "scripts"),
	}
	if err := fileutils.AssertDirectoriesExist(
		sdk.sdkDirPath,
		sdk.packageDirPath,
		sdk.scriptsDirPath,
	); err != nil {
		return nil, err
	}
	return sdk, nil
}

func (sdk *SdkRunner) RunScript(ctx context.Context, scriptName string, args ...string) error {
	scriptPath := path.Join(sdk.scriptsDirPath, scriptName)
	cmd := exec.CommandContext(ctx, scriptPath, args...)
	cmd.Dir = sdk.sdkDirPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run sdk script %q with args %v: %w", scriptPath, args, err)
	}
	return nil
}

func (sdk *SdkRunner) RunScriptForOutput(ctx context.Context, scriptName string, args ...string) (string, error) {
	scriptPath := path.Join(sdk.scriptsDirPath, scriptName)
	cmd := exec.CommandContext(ctx, scriptPath, args...)
	cmd.Dir = sdk.sdkDirPath
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run sdk script %q with args %v: %w", scriptPath, args, err)
	}
	return string(output), nil
}

func (sdk *SdkRunner) CompileCustomPackages(ctx context.Context, configOptions map[string]string, sdkSourcePackageMakefileDirs []string, autoRetrySdkCompile bool, maxCPUs int) error {
	// Resolve build flags.
	const verbosityFlag = "V=sw"
	if maxCPUs <= 0 {
		var err error
		maxCPUs, err = sdk.availableSystemCPUs(ctx)
		if err != nil {
			return fmt.Errorf("failed to determine system cpu core count: %w", err)
		}
	}
	maxCPUsFlag := fmt.Sprintf("-j%d", maxCPUs)

	// Download dependencies.
	log.Logger.Println("Downloading sdk dependencies")
	if err := sdk.Feeds(ctx, "update", "-a"); err != nil {
		return err
	}
	if err := sdk.Feeds(ctx, "install", "-a"); err != nil {
		return err
	}

	// Update sdk config now that options are valid.
	if len(configOptions) > 0 {
		if err := sdk.applyConfigChanges(ctx, configOptions); err != nil {
			return fmt.Errorf("failed to apply custom %d config options: %w", len(configOptions), err)
		}
	}

	// Build custom package files by running the specified makefiles.
	log.Logger.Printf("Making %d source packages with sdk using a maximum of %d CPUs\n", len(sdkSourcePackageMakefileDirs), maxCPUs)
	for _, makefile := range sdkSourcePackageMakefileDirs {
		log.Logger.Printf("Preparing source package %q for compilation\n", makefile)
		if err := sdk.Make(ctx, fmt.Sprintf("package/%s/prepare", makefile), maxCPUsFlag, verbosityFlag); err != nil {
			return fmt.Errorf("failed to prepare package %q for compilation: %w", makefile, err)
		}
		log.Logger.Printf("Compiling source package %q\n", makefile)
		compileArgs := []string{fmt.Sprintf("package/%s/compile", makefile), maxCPUsFlag, verbosityFlag}
		if err := sdk.Make(ctx, compileArgs...); err != nil {
			if !autoRetrySdkCompile {
				return fmt.Errorf("failed to compile package %q: %w", makefile, err)
			}
			// Try one more time, as it can be a flaky process.
			log.Logger.Printf("Failed to compile package %q on first attempt: %s\n", makefile, err)
			log.Logger.Printf("Retrying compilation of package once %q\n", makefile)
			if err := sdk.Make(ctx, compileArgs...); err != nil {
				return fmt.Errorf("failed to compile package %q after two attempts: %w", makefile, err)
			}
		}
	}

	log.Logger.Println("Package building via sdk complete")
	return nil
}

// Feeds runs the sdk feeds script.
func (sdk *SdkRunner) Feeds(ctx context.Context, args ...string) error {
	return sdk.RunScript(ctx, "feeds", args...)
}

// Make runs the make command in the sdk directory.
func (sdk *SdkRunner) Make(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "make", args...)
	cmd.Dir = sdk.sdkDirPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run sdk make with args %v: %w", args, err)
	}
	return nil
}

func (sdk *SdkRunner) makeConfig(ctx context.Context) error {
	return sdk.Make(ctx, "defconfig")
}

// ImportCustomPackageSources copies the contents of srcDir to the sdk packages
// dir.
func (sdk *SdkRunner) ImportCustomPackageSources(ctx context.Context, srcDir string) error {
	return fileutils.CopyFilesInDirToDir(ctx, srcDir, sdk.packageDirPath)
}

// ExportCompiledCustomPackageArchives copies built custom package files matching
// names in packageNames to dstDir.
func (sdk *SdkRunner) ExportCompiledCustomPackageArchives(ctx context.Context, packageNames []string, dstDir string) error {
	// Search the for desired package archives built by the sdk.
	log.Logger.Println("Collecting custom built package files from sdk")
	desiredPackageToPath := make(map[string]string)
	for _, packageName := range packageNames {
		desiredPackageToPath[packageName] = ""
	}
	err := filepath.Walk(path.Join(sdk.sdkDirPath, "bin"), func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() && (strings.HasSuffix(info.Name(), ".ipk") || strings.HasSuffix(info.Name(), ".apk")) {
			// APK package names no longer follow <pkg_name>_<version>_arch.ipk
			// and instead are just <pkg_name>-<version>.apk, do a naive search
			// for the package prefix to avoid errors splitting the string on the version which
			// has no defined format.
			for k := range desiredPackageToPath {
				if strings.HasPrefix(info.Name(), k) {
					if desiredPackageToPath[k] != "" {
						return fmt.Errorf("Unable to determine correct package for package: %q, duplicate found: %q", k, desiredPackageToPath[k])
					}
					desiredPackageToPath[k] = path
					log.Logger.Printf("Found custom archive of desired package %q at %q\n", k, path)
				}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to search for package files built by the local sdk: %w", err)
	}
	var missingPackages []string
	for packageName, filePath := range desiredPackageToPath {
		if filePath == "" {
			missingPackages = append(missingPackages, packageName)
		}
	}
	if len(missingPackages) > 0 {
		sort.Strings(missingPackages)
		return fmt.Errorf("failed to find custom built archives for desired packages: %s", strings.Join(missingPackages, " "))
	}

	// Copy desired package files to dstDir.
	log.Logger.Printf("Cleaning old package files from %q\n", dstDir)
	if err := fileutils.CleanDirectory(dstDir); err != nil {
		return err
	}
	log.Logger.Printf("Copying %d custom package files to %q\n", len(desiredPackageToPath), dstDir)
	for packageName, filePath := range desiredPackageToPath {
		if err := fileutils.CopyFile(ctx, filePath, dstDir); err != nil {
			return fmt.Errorf("failed to copy package %q file %q to dir %q: %w", packageName, filePath, dstDir, err)
		}
	}

	return nil
}

// applyConfigChanges updates the sdk .config with the provided configOptions.
// This must be done after feeds/downloaded are download that make the options
// valid or the option settings will be ignored.
func (sdk *SdkRunner) applyConfigChanges(ctx context.Context, configOptions map[string]string) error {
	// Make initial config.
	log.Logger.Println("Processing current sdk config")
	if err := sdk.makeConfig(ctx); err != nil {
		return err
	}

	// Append changes to config file.
	configOptionsStr := ""
	for k, v := range configOptions {
		configOptionsStr += fmt.Sprintf("%s=%s\n", k, v)
	}
	log.Logger.Printf("Applying sdk config options:\n%s", configOptionsStr)
	configFilePath := path.Join(sdk.sdkDirPath, ".config")
	configFile, err := os.OpenFile(configFilePath, os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("failed to open sdk config file %q: %w", configFilePath, err)
	}
	defer func() {
		_ = configFile.Close()
	}()
	if _, err := configFile.WriteString(configOptionsStr); err != nil {
		return fmt.Errorf("failed to append config options to config file %q: %w", configFilePath, err)
	}

	// Apply changes.
	log.Logger.Println("Processing updated sdk config")
	if err := sdk.makeConfig(ctx); err != nil {
		return err
	}
	return nil
}

func (sdk *SdkRunner) availableSystemCPUs(ctx context.Context) (int, error) {
	cmd := exec.CommandContext(ctx, "nproc")
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get available CPUs with nproc command: %w", err)
	}
	cpus, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 0, fmt.Errorf("failed to parse output of nproc command for available CPU count: %w", err)
	}
	return cpus, nil
}
