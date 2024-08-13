// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package commands

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	conf "go.chromium.org/chromiumos/config/go"
	"go.chromium.org/chromiumos/test/provision/v2/foil-provision/service"

	"go.chromium.org/chromiumos/config/go/test/api"

	"go.chromium.org/chromiumos/test/util/adb"
)

type Install struct {
	ctx            context.Context
	cs             *service.FoilService
	RebootRequired bool
}

func NewInstall(ctx context.Context, cs *service.FoilService) *Install {
	return &Install{
		ctx: ctx,
		cs:  cs,
	}
}

func (c *Install) Execute(log *log.Logger) error {

	log.Printf("Start Install Execute, with reboot")
	if c.cs.ImagePath == nil || c.cs.ImagePath.HostType != conf.StoragePath_GS {
		return fmt.Errorf("only support installing from GCS")
	}
	// Remove gs:// from the image path before pulling from cache server.
	localImagePath, err := c.pullFromCache(log, c.cs.ImagePath.GetPath()[5:])
	if err != nil {
		return fmt.Errorf("Unable to pull image from cache %v", err)
	}
	defer os.Remove(localImagePath)
	log.Println("Image pulled from cache and stored at %v", localImagePath)
	//TODO: Consume localImagePath inside the install function below.
	installErr := install(log, c.cs.UpdateEnginePid, c.cs.DutIp)

	if installErr != nil {
		return fmt.Errorf("Unable to install over 1 atempts.")
	}
	return nil
}

// pullFromCache downloads the image from cache server on cft container and
// returns the local path.
func (c *Install) pullFromCache(log *log.Logger, path string) (string, error) {

	var client http.Client
	cacheUrl := c.cs.CacheServerUrl
	cacheUrl.Path = filepath.Join("download", path)
	log.Println("pulling from cache url : %v", cacheUrl.String())
	resp, err := client.Get(cacheUrl.String())
	if err != nil {
		return "", fmt.Errorf("failed to pull from cache server %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error %s to download %q", resp.Status, cacheUrl.String())
	}
	out, err := os.CreateTemp(os.TempDir(), "image-*")
	if err != nil {
		return "", fmt.Errorf("failed to create a temp file %v", err)
	}
	log.Printf("created tmp file %v to store the pulled file", out.Name())

	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to copy request data to temp file %v", err)
	}
	return out.Name(), nil
}

func (c *Install) Revert() error {
	return nil
}

func (c *Install) GetErrorMessage() string {
	return "failed to install"
}

func (c *Install) GetStatus() api.InstallResponse_Status {
	return api.InstallResponse_STATUS_PROVISIONING_FAILED
}

func TestScanner(stream io.Reader, logger *log.Logger, harness string) bool {
	const maxCapacity = 4096 * 1024
	scanner := bufio.NewScanner(stream)
	completion := false
	// Expand the buffer size to avoid deadlocks on heavy logs
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {

		txt := scanner.Text()
		logger.Printf("[%v] %v", harness, txt)
		if strings.Contains(txt, "onPayloadApplicationComplete(ErrorCode::kSuccess") {
			logger.Println("found status in: ", txt)
			completion = true
		}
	}
	if scanner.Err() != nil {
		logger.Println("Failed to read pipe: ", scanner.Err())
	}
	logger.Println("Final completion status: ", completion)
	return completion
}

func logcat(log *log.Logger, pid string, addr string) {
	outStr, _ := adb.AdbCmd([]string{"-s", addr, "logcat", "-d", "--pid", pid}, log)

	log.Println("Finished co")
	log.Println("logcat out", outStr)

	return
}

// Helper function to read output from a pipe
func readOutput(r io.Reader, ch chan string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		ch <- scanner.Text()
	}
	close(ch)
}

func install(log *log.Logger, lcpid string, addr string) error {
	log.Printf("Install start")
	// logcmd, logchan, err := logcat()

	cmd := exec.Command("python3", []string{"/usr/local/update_device.py", "/otapackage.zip", "-s", adb.FmtAddr(addr)}...)

	log.Println("Running ADB OTA: ", cmd.String())

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("StderrPipe failed")
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("StdoutPipe failed")
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to run Cmd: %v", err)
	}
	var wg sync.WaitGroup
	wg.Add(2)

	const maxCapacity = 4096 * 1024
	found1 := false
	found2 := false
	go func() {
		defer wg.Done()
		found1 = TestScanner(stderr, log, "foil-prov:")
	}()

	go func() {
		defer wg.Done()
		found2 = TestScanner(stdout, log, "foil-prov")
	}()
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		log.Println("Failed to run ADB UPDATE: ", err)
	}
	log.Printf("DONE. Checking completion status")

	logcat(log, lcpid, addr)
	log.Println("LOGCAT CLOSED!")

	if found1 == true || found2 == true {
		log.Println("Found success status!")
		return nil
	}

	return fmt.Errorf("unable to find successful status.")
}
