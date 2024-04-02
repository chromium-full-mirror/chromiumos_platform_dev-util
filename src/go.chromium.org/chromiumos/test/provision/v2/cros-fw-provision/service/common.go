// Copyright 2022 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package firmwareservice

import (
	common_utils "go.chromium.org/chromiumos/test/provision/v2/common-utils"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"strings"

	"net/url"

	"github.com/pkg/errors"
)

// ImageArchiveMetadata will be the value of the map in which the key is the
// gsPath, so we can avoid downloading/reprocessing same archives.
type ImageArchiveMetadata struct {
	ArchiveDir string
}

// GetFlashECScript finds flash_ec script locally and returns path to it.
// If flash_ec is not found, download the latest version with git to |prefix|,
// and return path to downloaded flash_ec.
func GetFlashECScript(ctx context.Context, s common_utils.ServiceAdapterInterface, prefix string) (string, error) {
	// flash_ec within checkout will have access to the dependencies/config files
	preferredFlashEC := "~/chromiumos/src/platform/ec/util/flash_ec"
	if preferredExists, err := s.PathExists(ctx, preferredFlashEC); preferredExists && err == nil {
		return preferredFlashEC, nil
	}

	// find any other flash_ec
	flashEC, err := s.RunCmd(ctx, "which", []string{"flash_ec"})
	if len(flashEC) > 0 && err == nil {
		// `which` found the script
		return strings.TrimRight(flashEC, "\n"), nil
	}

	// donwload the platform/ec repo to get the flash_ec script
	log.Println("flash_ec script not found, downloading")
	_, err = s.RunCmd(ctx, "", []string{"cd " + prefix + ";", "git", "clone",
		"https://chromium.googlesource.com/chromiumos/platform/ec", "ec-repo"})
	if err != nil {
		return "", errors.Wrap(err, "falied to checkout platform/ec repo")
	}

	// TODO: mv ec-repo to some location and try it before downloading.

	return path.Join(prefix, "ec-repo", "util", "flash_ec"), nil
}

// PickAndExtractMainImage uses provided list of |filesInArchive| to pick a main
// image to use, extracts only it, and returns a path to extracted image.
// board and model(aka variant) are optional.
func PickAndExtractMainImage(ctx context.Context, s common_utils.ServiceAdapterInterface, imageMetadata ImageArchiveMetadata, gsPath string, fws *FirmwareService) (string, error) {
	log.Printf("Staging %q", gsPath)
	url := createStageURL(ctx, gsPath, fws.CacheServer)
	args := []string{"-f", "-s", "-S", url.String()}
	out, err := s.RunCmd(ctx, "curl", args)
	if err != nil {
		log.Printf("Failed to stage %q: %s", gsPath, string(out))
		return "", errors.Wrapf(err, "failed to stage %q", url.String())
	}
	candidates := []string{}
	if fws.CorebootName != "" {
		filePath := path.Join(fmt.Sprintf("image-%v.bin", fws.CorebootName))
		candidates = append(candidates, filePath)
	}
	if len(fws.GetModel()) > 0 {
		filePath := path.Join(fmt.Sprintf("image-%v.bin", fws.GetModel()))
		candidates = append(candidates, filePath)
	}
	if len(fws.GetBoard()) > 0 {
		filePath := path.Join(fmt.Sprintf("image-%v.bin", fws.GetBoard()))
		candidates = append(candidates, filePath)
	}
	candidates = append(candidates, path.Join("image.bin"))
	candidates = append(candidates, path.Join("bios.bin"))

	for i := 0; i < len(candidates); i++ {
		log.Printf("Trying %q", candidates[i])
		url, err := createExtractURL(ctx, gsPath, candidates[i], fws.CacheServer)
		if err != nil {
			return "", errors.Wrapf(err, "no url for %q", candidates[i])
		}
		destPath := fmt.Sprintf("%s/bios.bin", imageMetadata.ArchiveDir)
		args := []string{"-f", "-s", "-S", "-o", destPath, url.String()}
		out, err := s.RunCmd(ctx, "curl", args)
		if err != nil {
			log.Printf("Failed to download %q: %s", candidates[i], string(out))
			continue
		}
		log.Printf("Extracted %q as %q", candidates[i], destPath)
		return destPath, nil
	}
	return "", fmt.Errorf(`could not find an AP image named any of: %v.
Specifying board and model may help`, candidates)
}

// PickAndExtractECImage uses provided list of |filesInArchive| to pick an EC
// image to use, extracts only it, and returns a path to extracted image.
// board and model(aka variant) are optional.
func PickAndExtractECImage(ctx context.Context, s common_utils.ServiceAdapterInterface, imageMetadata ImageArchiveMetadata, gsPath string, fws *FirmwareService) (string, error) {
	log.Printf("Staging %q", gsPath)
	url := createStageURL(ctx, gsPath, fws.CacheServer)
	args := []string{"-f", "-s", "-S", url.String()}
	out, err := s.RunCmd(ctx, "curl", args)
	if err != nil {
		log.Printf("Failed to stage %q: %s", gsPath, string(out))
		return "", errors.Wrapf(err, "failed to stage %q", url.String())
	}
	candidates := []string{}
	if fws.ECName != "" {
		filePath := path.Join(fmt.Sprintf("%v/ec.bin", fws.ECName))
		candidates = append(candidates, filePath)
	}
	if len(fws.GetModel()) > 0 {
		filePath := path.Join(fws.GetModel(), "ec.bin")
		candidates = append(candidates, filePath)
	}
	if len(fws.GetBoard()) > 0 {
		filePath := path.Join(fws.GetBoard(), "ec.bin")
		candidates = append(candidates, filePath)
	}
	candidates = append(candidates, path.Join("ec.bin"))

	for i := 0; i < len(candidates); i++ {
		log.Printf("Trying %q", candidates[i])
		url, err := createExtractURL(ctx, gsPath, candidates[i], fws.CacheServer)
		if err != nil {
			return "", errors.Wrapf(err, "no url for %q", candidates[i])
		}
		destPath := fmt.Sprintf("%s/ec.bin", imageMetadata.ArchiveDir)
		args := []string{"-f", "-s", "-S", "-o", destPath, url.String()}
		out, err := s.RunCmd(ctx, "curl", args)
		if err != nil {
			log.Printf("Failed to download %q: %s", candidates[i], string(out))
			continue
		}
		log.Printf("Extracted %q as %q", candidates[i], destPath)
		// Try to get npcx_monitor.bin also
		npcxCandidate := strings.Replace(candidates[i], "ec.bin", "npcx_monitor.bin", 1)
		log.Printf("Trying %q", npcxCandidate)
		url, err = createExtractURL(ctx, gsPath, npcxCandidate, fws.CacheServer)
		if err != nil {
			return "", errors.Wrapf(err, "no url for %q", npcxCandidate)
		}
		npcxPath := fmt.Sprintf("%s/npcx_monitor.bin", imageMetadata.ArchiveDir)
		args = []string{"-f", "-s", "-S", "-o", npcxPath, url.String()}
		out, err = s.RunCmd(ctx, "curl", args)
		if err != nil {
			log.Printf("Failed to download %q: %s", npcxCandidate, string(out))
		} else {
			log.Printf("Extracted %q as %q", npcxCandidate, npcxPath)
		}
		return destPath, nil
	}
	return "", fmt.Errorf(`could not find an EC image named any of: %v.
Specifying board and model may help`, candidates)
}

// createStageURL returns the URL to stage a gsPath. Pass to curl on the DUT or call stageFile() to do in this service.
func createStageURL(ctx context.Context, gsPath string, cacheServer url.URL) url.URL {
	stagingURL := cacheServer
	stagingURL.Path = "/stage"
	v := url.Values{}
	v.Set("archive_url", path.Dir(gsPath))
	v.Set("files", path.Base(gsPath))
	stagingURL.RawQuery = v.Encode()
	return stagingURL
}

// createExtractURL returns the URL to extract a file from a gsPath. Pass to curl on the DUT or call extractFile() to do in this service.
func createExtractURL(ctx context.Context, gsPath, fileInArchive string, cacheServer url.URL) (url.URL, error) {
	gsPathURL, err := url.Parse(gsPath)
	if err != nil {
		return url.URL{}, errors.Wrapf(err, "failed to parse %q", gsPath)
	}
	extractURL := cacheServer
	extractURL.Path = fmt.Sprintf("/extract/%s%s", gsPathURL.Host, gsPathURL.Path)
	v := url.Values{}
	v.Set("file", fileInArchive)
	extractURL.RawQuery = v.Encode()
	return extractURL, nil
}

// stageFile stages a gsPath on the cache server.
func stageFile(ctx context.Context, gsPath string, cacheServer url.URL) error {
	stagingURL := createStageURL(ctx, gsPath, cacheServer)
	log.Printf("[FW Provisioning: Stage FW] staging %q\n", stagingURL.String())
	var client http.Client
	resp, err := client.Get(stagingURL.String())
	if err != nil {
		return errors.Wrapf(err, "failed to stage %q", stagingURL.String())
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("error %s to stage %q", resp.Status, stagingURL.String())
	}

	return nil
}

// extractFile extracts a file from an archive using the cache server.
func extractFile(ctx context.Context, gsPath, fileInArchive string, cacheServer url.URL) (io.ReadCloser, error) {
	extractURL, err := createExtractURL(ctx, gsPath, fileInArchive, cacheServer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate url")
	}
	log.Printf("[FW Provisioning: Stage FW] extract %q\n", extractURL.String())

	var client http.Client
	resp, err := client.Get(extractURL.String())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract %q", extractURL.String())
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("error %s to extract %q", resp.Status, extractURL.String())
	}

	return resp.Body, nil
}
