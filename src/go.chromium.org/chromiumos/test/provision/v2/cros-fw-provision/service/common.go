// Copyright 2022 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package firmwareservice

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	"go.chromium.org/chromiumos/config/go/test/api"
	common_utils "go.chromium.org/chromiumos/test/provision/v2/common-utils"

	"net/url"

	"github.com/pkg/errors"
)

const curlExtractTimeout = 20 * time.Minute

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

var curlErrorRe *regexp.Regexp = regexp.MustCompile(`The requested URL returned error: (\d+)`)

// RunDUTCommand runs a command on the DUT and returns stdout, stderr, and an error if it failed.
// CAUTION: The args are concatenated with no quoting, so beware of spaces and shell chars in filenames.
func RunDUTCommand(ctx context.Context, dut api.DutServiceClient, timeout time.Duration, cmd string, args []string, stdin []byte) (string, string, error) {
	type execCmdResult struct {
		response string
		stderr   string
		err      error
	}

	// Channel used to receive the result from ExecCommand function.
	ch := make(chan execCmdResult, 1)

	// Create a context with the specified timeout.
	ctxTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	log.Printf("RunDUTCommand START: %s %s\n", cmd, args)

	// Start the execCmd function.
	go func() {
		req := api.ExecCommandRequest{
			Command: cmd,
			Args:    args,
			Stdout:  api.Output_OUTPUT_PIPE,
			Stderr:  api.Output_OUTPUT_PIPE,
			Stdin:   stdin,
		}
		stream, err := dut.ExecCommand(ctx, &req)
		if err != nil {
			log.Printf("RunDUTCommand FAILED: %s\n", err)
			ch <- execCmdResult{err: fmt.Errorf("execution fail: %w", err)}
			return
		}
		// Expecting single stream result
		execCmdResponse, err := stream.Recv()
		if err != nil {
			ch <- execCmdResult{err: fmt.Errorf("execution single stream result: %w", err)}
			return
		}
		if execCmdResponse.ExitInfo.Status != 0 {
			err = fmt.Errorf("status:%v message:%v", execCmdResponse.ExitInfo.Status, execCmdResponse.ExitInfo.ErrorMessage)
		}
		if string(execCmdResponse.Stderr) != "" {
			log.Printf("execution finished with stderr: %s\n", string(execCmdResponse.Stderr))
		}
		ch <- execCmdResult{response: string(execCmdResponse.Stdout), stderr: string(execCmdResponse.Stderr), err: err}
	}()

	select {
	case <-ctxTimeout.Done():
		return "", "", fmt.Errorf("timeout %s reached", timeout)
	case result := <-ch:
		return result.response, result.stderr, result.err
	}
}

// ExtractFile calls the cache server to extract a file to the DUT, and retries on 5xx http errors.
// Returns false, nil on 404 errors. Returns an error on all other errors.
func ExtractFile(ctx context.Context, dut api.DutServiceClient, url, destPath string) (bool, error) {
	err := errors.New("unknown error")
	for i := 1; i <= 5; i++ {
		var stderr string
		_, stderr, err = RunDUTCommand(ctx, dut, curlExtractTimeout, "curl", []string{"-f", "-S", "-o", fmt.Sprintf("'%s'", destPath), fmt.Sprintf("'%s'", url)}, nil)

		if err != nil {
			log.Printf("Failed to download %q: %s", url, string(stderr))
			m := curlErrorRe.FindStringSubmatch(stderr)
			log.Printf("result.stderr: %v", stderr)
			log.Printf("Regex matched: %v", m)
			if m != nil && len(m[1]) == 3 && m[1][0] == '5' {
				// retry on 5xx errors
				continue
			}
			if m != nil && string(m[1]) == "404" {
				// return false on 404 errors
				return false, nil
			}
			// Fail on all other errors
			return false, errors.Wrapf(err, "curl failed: %s", stderr)
		}
		log.Printf("Extracted %q as %q", url, destPath)
		return true, nil
	}
	return false, err
}

// PickAndExtractMainImage uses provided list of |filesInArchive| to pick a main
// image to use, extracts only it, and returns a path to extracted image.
// board and model(aka variant) are optional.
func PickAndExtractMainImage(ctx context.Context, dut api.DutServiceClient, imageMetadata ImageArchiveMetadata, gsPath string, fws *FirmwareService) (string, error) {
	log.Printf("Staging %q", gsPath)
	url := createStageURL(ctx, gsPath, fws.CacheServer)
	args := []string{"-f", "-S", fmt.Sprintf("'%s'", url.String())}
	_, out, err := RunDUTCommand(ctx, dut, curlExtractTimeout, "curl", args, nil)
	if err != nil {
		log.Printf("Failed to stage %q: %v\n%s", gsPath, err, string(out))
		return "", errors.Wrapf(err, "failed to stage %q", url.String())
	}
	log.Printf("Stage of %q success: %s", gsPath, string(out))
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
		if ok, err := ExtractFile(ctx, dut, url.String(), destPath); err != nil {
			return "", errors.Wrapf(err, "extract %q", candidates[i])
		} else if !ok {
			log.Printf("%q not found", candidates[i])
			continue
		}
		return destPath, nil
	}
	return "", fmt.Errorf(`could not find an AP image named any of: %v.
Specifying board and model may help`, candidates)
}

// PickAndExtractECImage uses provided list of |filesInArchive| to pick an EC
// image to use, extracts only it, and returns a path to extracted image.
// board and model(aka variant) are optional.
func PickAndExtractECImage(ctx context.Context, dut api.DutServiceClient, imageMetadata ImageArchiveMetadata, gsPath string, fws *FirmwareService) (string, error) {
	log.Printf("Staging %q", gsPath)
	url := createStageURL(ctx, gsPath, fws.CacheServer)
	args := []string{"-f", "-S", fmt.Sprintf("'%s'", url.String())}
	_, out, err := RunDUTCommand(ctx, dut, curlExtractTimeout, "curl", args, nil)
	if err != nil {
		log.Printf("Failed to stage %q: %s", gsPath, string(out))
		return "", errors.Wrapf(err, "failed to stage %q", url.String())
	}
	log.Printf("Stage of %q success: %s", gsPath, string(out))
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
		if ok, err := ExtractFile(ctx, dut, url.String(), destPath); err != nil {
			return "", errors.Wrapf(err, "extract %q", candidates[i])
		} else if !ok {
			log.Printf("%q not found", candidates[i])
			continue
		}
		// Try to get npcx_monitor.bin also
		npcxCandidate := strings.Replace(candidates[i], "ec.bin", "npcx_monitor.bin", 1)
		log.Printf("Trying %q", npcxCandidate)
		url, err = createExtractURL(ctx, gsPath, npcxCandidate, fws.CacheServer)
		if err != nil {
			return "", errors.Wrapf(err, "no url for %q", npcxCandidate)
		}
		npcxPath := fmt.Sprintf("%s/npcx_monitor.bin", imageMetadata.ArchiveDir)
		if ok, err := ExtractFile(ctx, dut, url.String(), npcxPath); err != nil {
			return "", errors.Wrapf(err, "extract %q", npcxCandidate)
		} else if !ok {
			log.Printf("%q not found", npcxCandidate)
		}
		return destPath, nil
	}
	return "", fmt.Errorf(`could not find an EC image named any of: %v.
Specifying board and model may help`, candidates)
}

// createStageURL returns the URL to stage a gsPath. Pass to curl on the DUT or call stageFile() to do in this service.
func createStageURL(ctx context.Context, gsPath string, cacheServer url.URL) url.URL {
	stagingURL := cacheServer
	stagingURL.Path = "/stage/"
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
