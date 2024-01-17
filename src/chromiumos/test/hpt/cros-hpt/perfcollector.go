// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"chromiumos/test/publish/cmd/publishserver/storage"

	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/config/go/test/hpt"
	"google.golang.org/protobuf/proto"
)

// TODO(b:323393089) Create a separate lib for perf collector.
const (
	ls      = "ls"
	rm      = "rm"
	stopped = "stopped"
)

type publisher interface {
	Publish(ctx context.Context, data []byte) error
}

type storageWriter interface {
	Write(ctx context.Context, filePath string, gsObject storage.GSObject) error
}

type file interface {
	Copy(ctx context.Context, file string, dut api.DutServiceClient) (string, error)
}

type tar interface {
	Tar(ctx context.Context, srcFile, destTarFile string) error
}

// perfCollector regularly copies over perf.data from the DUT to cros-hpt host and upload to gcs.
type perfCollector struct {
	fileFetcher           file
	exec                  execUtil
	dut                   api.DutServiceClient
	perfCollectorInterval time.Duration
	tarUtil               tar
	storage               storageWriter
	bucket                string
	publisher             publisher

	// terminated chan is used by the collector to notify that it has successfully terminated.
	terminated chan bool

	// stopCollector chan is used to notify collector to stop collecting.
	stopCollector chan bool
}

// listPerfFiles will list all the perf data files on dut under perfDir.
func (p perfCollector) listPerfFiles(ctx context.Context, perfDir string) ([]string, error) {
	lsResult, err := p.exec.RunCmd(ctx, ls, []string{"-l", perfDir}, p.dut)
	if err != nil {
		return nil, err
	}
	files := strings.Split(lsResult, "\n")
	var result []string
	for _, file := range files {
		file = strings.TrimSpace(file)
		if file != "" {
			result = append(result, file)
		}
	}
	return result, nil
}

func (p perfCollector) tar(ctx context.Context, srcFile, destDir string) (string, error) {
	tarFile := filepath.Join(destDir, filepath.Base(srcFile)+".gz")
	if err := p.tarUtil.Tar(ctx, srcFile, tarFile); err != nil {
		return "", err
	}
	return tarFile, nil
}

func (p perfCollector) publish(ctx context.Context, bucket, ObjectPath string) error {
	record := hpt.PerfRecord{
		Bucket:     &bucket,
		ObjectPath: &ObjectPath,
		// TODO: set Branch, Version and Board
	}
	out, err := proto.Marshal(&record)
	if err != nil {
		return err
	}
	return p.publisher.Publish(ctx, out)
}

func (p perfCollector) gatherPerf(ctx context.Context, perfDir string) error {
	// First list all the perf data files on the DUT under path perfDir.
	files, err := p.listPerfFiles(ctx, perfDir)
	if err != nil {
		return err
	}
	for _, dutFile := range files {
		// Next copy the files to the cros-dut one by one.
		localPath, err := p.fileFetcher.Copy(ctx, dutFile, p.dut)
		if err != nil {
			return err
		}
		// Next zip the perf data.
		tarFilePath, err := p.tar(ctx, localPath, os.TempDir())
		// Next clean up the local file since we dont need it.
		if err := os.Remove(localPath); err != nil {
			return err
		}
		// Next upload to GCS.
		gsObject := storage.GSObject{
			Bucket: p.bucket,
			Object: filepath.Join(time.Now().Format(time.DateOnly), filepath.Base(tarFilePath)),
		}
		if err := p.storage.Write(ctx, tarFilePath, gsObject); err != nil {
			return err
		}
		// Next notify perf processing pipeline to process the perf.data
		if err := p.publish(ctx, p.bucket, gsObject.Object); err != nil {
			return err
		}
		// Next clean up the file on dut.
		if _, err = p.exec.RunCmd(ctx, rm, []string{dutFile}, p.dut); err != nil {
			return err
		}
	}
	return nil
}

func (p perfCollector) start(ctx context.Context, perfDir string) {
	for {
		select {
		case <-p.stopCollector:
			log.Println("Stopping perf collector.")
			p.terminated <- true
			return
		case <-time.After(p.perfCollectorInterval):
			if err := p.gatherPerf(ctx, perfDir); err != nil {
				log.Printf("error in perf collector %v", err)
				p.terminated <- true
				return
			}
		}
	}
}
