// Copyright 2022 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"chromium.googlesource.com/chromiumos/platform/dev-util.git/contrib/fflash/internal/artifact"
	"chromium.googlesource.com/chromiumos/platform/dev-util.git/contrib/fflash/internal/logging"
	"chromium.googlesource.com/chromiumos/platform/dev-util.git/contrib/fflash/internal/progress"
)

func Main() error {
	var r Request
	if err := gob.NewDecoder(os.Stdin).Decode(&r); err != nil {
		return fmt.Errorf("cannot decode flash request: %w", err)
	}
	logging.SetUp(time.Now().Add(-r.ElapsedTimeWhenSent))
	log.SetPrefix("[dut-agent] ")

	ctx := context.Background()

	partState, err := ActivePartitions(ctx)
	if err != nil {
		return fmt.Errorf("cannot get active partitions: %w", err)
	}
	log.Printf("active kernel/rootfs: %s, %s", partState.ActiveKernel(), partState.ActiveRootfs())
	log.Printf("flashing to: %s, %s", partState.InactiveKernel(), partState.InactiveRootfs())

	payload, err := r.Payload.Instance()
	if err != nil {
		return fmt.Errorf("r.Payload.Instance(): %w", err)
	}
	if err := payload.InitClient(ctx); err != nil {
		return fmt.Errorf("payload.InitClient(): %w", err)
	}
	defer payload.CloseClient()

	// Prevent the system from suspending or shutting down.
	err = os.WriteFile(powerdLockFilePath, []byte(""), 0755)
	if err != nil {
		return err
	}
	defer os.Remove(powerdLockFilePath)

	pr := progress.NewProgressReporter()
	ch := make(chan error)

	flashCtx, cancelFlash := context.WithCancel(ctx)
	defer cancelFlash()

	go func(rw *progress.ReportingWriter) {
		ch <- r.Flash(flashCtx, payload, rw, artifact.Kernel, partState.InactiveKernel(),
			// Flush the page cache for the kernel partition to avoid aliasing between
			// the partition and the device. See also https://crrev.com/c/5934991.
			true,
		)
	}(pr.NewWriter("kernel"))
	go func(rw *progress.ReportingWriter) {
		ch <- r.Flash(flashCtx, payload, rw, artifact.Rootfs, partState.InactiveRootfs(), false)
	}(pr.NewWriter("rootfs"))
	go func(rw *progress.ReportingWriter) {
		ch <- r.FlashStateful(flashCtx, payload, rw, r.ClobberStateful)
	}(pr.NewWriter("stateful"))

	var failed bool
	completed := 0
	ticker := time.NewTicker(time.Second)
	for completed < 3 {
		select {
		case err := <-ch:
			if err != nil {
				failed = true
				cancelFlash()
				if !errors.Is(err, context.Canceled) {
					log.Println(err)
				}
			}
			completed += 1
		case <-ticker.C:
			log.Println("flash", pr.Report())
		}
	}
	ticker.Stop()
	if failed {
		return fmt.Errorf("flash failed")
	}
	log.Println("flash", pr.Report())

	log.Println("running postinst")
	if err := RunPostinst(ctx, partState.InactiveRootfs()); err != nil {
		return fmt.Errorf("postinst failed: %w", err)
	}

	result := &Result{}

	if r.DisableRootfsVerification {
		log.Println("disabling rootfs verification")
		if err := DisableRootfsVerification(ctx, partState.InactiveKernelNum); err != nil {
			log.Printf("disable rootfs verification failed (will retry after reboot): %s", err)
			result.RetryDisableRootfsVerification = true
		}
	}

	if r.ClearTpmOwner {
		log.Println("clearing tpm owner")
		if err := ClearTpmOwner(ctx); err != nil {
			log.Printf("clear tpm owner request failed (will retry after reboot): %s", err)
			result.RetryClearTpmOwner = true
		}
	}

	if err := gob.NewEncoder(os.Stdout).Encode(result); err != nil {
		return fmt.Errorf("failed to encode result: %w", err)
	}

	return nil
}
