// Copyright 2022 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package firmwareservice

import (
	common_utils "chromiumos/test/provision/v2/common-utils"
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

const FirmwarePathTmp = "/tmp/cros-fw-provisioning-service/"
const CurlWithRetriesArgsFW = "-S -s -v -# -C - --retry 3 --retry-delay 60"

// ImageArchiveMetadata will be the value of the map in which the key is the
// gsPath, so we can avoid downloading/reprocessing same archives.
type ImageArchiveMetadata struct {
	ArchiveDir   string
	CorebootName string
	ECName       string
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
func PickAndExtractMainImage(ctx context.Context, s common_utils.ServiceAdapterInterface, imageMetadata ImageArchiveMetadata, gsPath, board, model string, cacheServer url.URL) (string, error) {
	candidates := []string{}
	if imageMetadata.CorebootName != "" {
		filePath := path.Join(fmt.Sprintf("image-%v.bin", imageMetadata.CorebootName))
		candidates = append(candidates, filePath)
	}
	if len(model) > 0 {
		filePath := path.Join(fmt.Sprintf("image-%v.bin", model))
		candidates = append(candidates, filePath)
	}
	if len(board) > 0 {
		filePath := path.Join(fmt.Sprintf("image-%v.bin", board))
		candidates = append(candidates, filePath)
	}
	candidates = append(candidates, path.Join("image.bin"))
	candidates = append(candidates, path.Join("bios.bin"))

	for i := 0; i < len(candidates); i++ {
		log.Printf("Trying %q", candidates[i])
		url, err := createExtractURL(ctx, gsPath, candidates[i], cacheServer)
		if err != nil {
			return "", errors.Wrapf(err, "no url for %q", candidates[i])
		}
		destPath := fmt.Sprintf("%s/bios.bin", imageMetadata.ArchiveDir)
		args := []string{"-S", "-o", destPath, url.String()}
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
func PickAndExtractECImage(ctx context.Context, s common_utils.ServiceAdapterInterface, imageMetadata ImageArchiveMetadata, gsPath, board, model string, cacheServer url.URL) (string, error) {
	candidates := []string{}
	if imageMetadata.ECName != "" {
		filePath := path.Join(fmt.Sprintf("%v/ec.bin", imageMetadata.ECName))
		candidates = append(candidates, filePath)
	}
	if len(model) > 0 {
		filePath := path.Join(model, "ec.bin")
		candidates = append(candidates, filePath)
	}
	if len(board) > 0 {
		filePath := path.Join(board, "ec.bin")
		candidates = append(candidates, filePath)
	}
	candidates = append(candidates, path.Join("ec.bin"))

	for i := 0; i < len(candidates); i++ {
		log.Printf("Trying %q", candidates[i])
		url, err := createExtractURL(ctx, gsPath, candidates[i], cacheServer)
		if err != nil {
			return "", errors.Wrapf(err, "no url for %q", candidates[i])
		}
		destPath := fmt.Sprintf("%s/ec.bin", imageMetadata.ArchiveDir)
		args := []string{"-S", "-o", destPath, url.String()}
		out, err := s.RunCmd(ctx, "curl", args)
		if err != nil {
			log.Printf("Failed to download %q: %s", candidates[i], string(out))
			continue
		}
		log.Printf("Extracted %q as %q", candidates[i], destPath)
		// Try to get npcx_monitor.bin also
		npcxCandidate := strings.Replace(candidates[i], "ec.bin", "npcx_monitor.bin", 1)
		log.Printf("Trying %q", npcxCandidate)
		url, err = createExtractURL(ctx, gsPath, npcxCandidate, cacheServer)
		if err != nil {
			return "", errors.Wrapf(err, "no url for %q", npcxCandidate)
		}
		npcxPath := fmt.Sprintf("%s/npcx_monitor.bin", imageMetadata.ArchiveDir)
		args = []string{"-S", "-o", npcxPath, url.String()}
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

func stageFile(ctx context.Context, gsPath string, cacheServer url.URL) error {
	stagingURL := cacheServer
	stagingURL.Path = "/stage"
	v := url.Values{}
	v.Set("archive_url", path.Dir(gsPath))
	v.Set("files", path.Base(gsPath))
	stagingURL.RawQuery = v.Encode()

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
