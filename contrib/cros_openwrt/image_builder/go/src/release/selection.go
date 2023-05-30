// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package release

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	labapi "go.chromium.org/chromiumos/config/go/test/lab/api"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/log"
)

func SelectImageByUUID(deviceConfig *labapi.OpenWrtWifiRouterDeviceConfig, imageUUID string) (*labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage, error) {
	for _, imageConfig := range deviceConfig.Images {
		if strings.EqualFold(imageConfig.ImageUuid, imageUUID) {
			return imageConfig, nil
		}
	}
	return nil, fmt.Errorf("found no image with ImageUUID %q configured for device", imageUUID)
}

func SelectCurrentImage(deviceConfig *labapi.OpenWrtWifiRouterDeviceConfig) (*labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage, error) {
	if deviceConfig.CurrentImageUuid == "" {
		return nil, errors.New("no current image configured for device")
	}
	imageConfig, err := SelectImageByUUID(deviceConfig, deviceConfig.CurrentImageUuid)
	if err != nil {
		return nil, fmt.Errorf("failed to select image set as current image: %w", err)
	}
	return imageConfig, nil
}
func SelectNextImage(deviceConfig *labapi.OpenWrtWifiRouterDeviceConfig) (*labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage, error) {
	if deviceConfig.NextImageUuid == "" {
		return nil, errors.New("no next image configured for device")
	}
	imageConfig, err := SelectImageByUUID(deviceConfig, deviceConfig.NextImageUuid)
	if err != nil {
		return nil, fmt.Errorf("failed to select image set as next image: %w", err)
	}
	return imageConfig, nil
}

func SelectImageByCrosReleaseVersion(deviceConfig *labapi.OpenWrtWifiRouterDeviceConfig, dutCrosReleaseVersion string, useCurrentIfNoMatches bool) (*labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage, error) {
	if len(deviceConfig.Images) == 0 {
		return nil, fmt.Errorf("no images are configured for this device")
	}
	dutVersion, err := ParseChromeOSReleaseVersion(dutCrosReleaseVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid CHROMEOS_RELEASE_VERSION %q: %w", dutCrosReleaseVersion, err)
	}

	// Collect all matching versions.
	var allMatchingImageVersions []ChromeOSReleaseVersion
	imageVersionToConfig := make(map[string]*labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage)
	for i, imageConfig := range deviceConfig.Images {
		imageMinVersion, err := ParseChromeOSReleaseVersion(imageConfig.MinDutReleaseVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to parse deviceConfig.Images[%d].MinDutReleaseVersion: %w", i, err)
		}
		if !IsChromeOSReleaseVersionLessThan(dutVersion, imageMinVersion) {
			allMatchingImageVersions = append(allMatchingImageVersions, imageMinVersion)
			imageVersionToConfig[imageMinVersion.String()] = imageConfig
		}
	}
	if len(allMatchingImageVersions) == 0 {
		if useCurrentIfNoMatches {
			log.Logger.Printf("WARNING: none of the %d images configured for this device have a MinDutReleaseVersion greater than or equal to %q; selecting current version instead", len(deviceConfig.Images), dutVersion.String())
			return SelectCurrentImage(deviceConfig)
		}
		return nil, fmt.Errorf("none of the %d images configured for this device have a MinDutReleaseVersion greater than or equal to %q", len(deviceConfig.Images), dutVersion.String())
	}

	// Sort them and use the highest matching min version.
	sort.SliceStable(allMatchingImageVersions, func(i, j int) bool {
		return IsChromeOSReleaseVersionLessThan(allMatchingImageVersions[i], allMatchingImageVersions[j])
	})
	highestMatchingVersion := allMatchingImageVersions[len(allMatchingImageVersions)-1]
	if len(allMatchingImageVersions) > 1 {
		secondHighestMatchingVersion := allMatchingImageVersions[len(allMatchingImageVersions)-2]
		if !IsChromeOSReleaseVersionLessThan(secondHighestMatchingVersion, highestMatchingVersion) {
			// Versions are the same, and thus we cannot pick between the two images
			// they belong to (this is a config error we'd need to fix manually).
			return nil, fmt.Errorf("config error: unable to choose image for CHROMEOS_RELEASE_VERSION %q, as multiple matching images were found with the same MinDutReleaseVersion %q", dutVersion.String(), highestMatchingVersion.String())
		}
	}
	return imageVersionToConfig[highestMatchingVersion.String()], nil
}

func SelectImageForDut(deviceConfig *labapi.OpenWrtWifiRouterDeviceConfig, dutHostname, dutCrosReleaseVersion string) (*labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage, error) {
	var dutVersion ChromeOSReleaseVersion
	if dutCrosReleaseVersion != "" {
		var err error
		dutVersion, err = ParseChromeOSReleaseVersion(dutCrosReleaseVersion)
		if err != nil {
			return nil, fmt.Errorf("invalid CHROMEOS_RELEASE_VERSION %q: %w", dutCrosReleaseVersion, err)
		}
	}
	for _, hostname := range deviceConfig.NextImageVerificationDutPool {
		if hostname == dutHostname {
			log.Logger.Printf("DUT %q is in the NextImageVerificationDutPool, selecting next image\n", dutHostname)
			if dutVersion == nil {
				return SelectNextImage(deviceConfig)
			}
			nextImage, err := SelectNextImage(deviceConfig)
			if err != nil {
				return nil, err
			}
			nextImageMinVersion, err := ParseChromeOSReleaseVersion(nextImage.MinDutReleaseVersion)
			if err != nil {
				return nil, fmt.Errorf("failed to parse MinDutReleaseVersion for next image with UUID %q: %w", nextImage.ImageUuid, err)
			}
			if IsChromeOSReleaseVersionLessThan(dutVersion, nextImageMinVersion) {
				log.Logger.Printf("WARNING: The DUT CHROMEOS_RELEASE_VERSION provided, %q, is less than the selected next OpenWrt OS image MinDutReleaseVersion %q\n", dutVersion.String(), nextImage.String())
			}
			return nextImage, nil
		}
	}
	if dutVersion == nil {
		log.Logger.Printf("DUT %q is not in the NextImageVerificationDutPool and no DUT CHROMEOS_RELEASE_VERSION specified, selecting current image\n", dutHostname)
		return SelectCurrentImage(deviceConfig)
	}
	currentImage, err := SelectNextImage(deviceConfig)
	if err != nil {
		return nil, err
	}
	currentImageMinVersion, err := ParseChromeOSReleaseVersion(currentImage.MinDutReleaseVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MinDutReleaseVersion for current image with UUID %q: %w", currentImage.ImageUuid, err)
	}
	if IsChromeOSReleaseVersionLessThan(dutVersion, currentImageMinVersion) {
		log.Logger.Printf("WARNING: DUT %q is not in the NextImageVerificationDutPool and the current image's MinDutReleaseVersion is higher than requested, selecting image by CHROMEOS_RELEASE_VERSION from available images\n", dutHostname)
		return SelectImageByCrosReleaseVersion(deviceConfig, dutCrosReleaseVersion, true)
	}
	return currentImage, nil
}
