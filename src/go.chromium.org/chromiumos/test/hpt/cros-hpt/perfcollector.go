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

	"go.chromium.org/chromiumos/test/publish/cmd/publishserver/storage"

	"github.com/google/uuid"
	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/config/go/test/hpt"
	"google.golang.org/protobuf/encoding/protojson"
)

// TODO(b:323393089) Create a separate lib for perf collector.
const (
	ls           = "ls"
	rm           = "rm"
	stopped      = "stopped"
	noDirMsg     = "No such file or directory"
	perfDataWild = perfData + ".*"
	gz           = ".tar.gz"
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
	version               string
	snapshotVersion       string
	board                 string
	// terminated chan is used by the collector to notify that it has successfully terminated.
	terminated chan bool

	// stopCollector chan is used to notify collector to stop collecting.
	stopCollector chan bool
}

// listPerfFiles will list all the perf data files on dut under perfDir.
func (p perfCollector) listPerfFiles(ctx context.Context, perfDir string) ([]string, error) {
	perfFilePath := filepath.Join(perfDir, perfDataWild)
	lsResult, err := p.exec.RunCmd(ctx, ls, []string{perfFilePath}, p.dut)
	if err != nil && !strings.Contains(err.Error(), noDirMsg) {
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
	tarFile := filepath.Join(destDir, filepath.Base(srcFile)+gz)
	if err := p.tarUtil.Tar(ctx, srcFile, tarFile); err != nil {
		return "", err
	}
	return tarFile, nil
}

func (p perfCollector) publish(ctx context.Context, bucket, ObjectPath string) error {
	board := strings.Clone(brya)
	record := hpt.PerfRecord{
		Bucket:            &bucket,
		ObjectPath:        &ObjectPath,
		Version:           &p.version,
		SnapshotBuildPath: &p.snapshotVersion,
		Board:             &board,
	}
	out, err := protojson.Marshal(&record)
	if err != nil {
		return err
	}
	return p.publisher.Publish(ctx, out)
}

func (p perfCollector) gatherPerf(ctx context.Context, perfDir, uuid, workDir string) error {
	// First list all the perf data files on the DUT under path perfDir.
	files, err := p.listPerfFiles(ctx, perfDir)
	if err != nil {
		return err
	}
	for _, dutFile := range files {
		// Next copy the files from cros-dut one by one.
		localPath, err := p.fileFetcher.Copy(ctx, dutFile, p.dut)
		if err != nil {
			return err
		}
		// Next zip the perf data.
		tarFilePath, err := p.tar(ctx, localPath, workDir)
		if err != nil {
			return err
		}

		// Next clean up the local file since we dont need it.
		if err := os.Remove(localPath); err != nil {
			return err
		}
		gsObject := storage.GSObject{
			Bucket: p.bucket,
			Object: filepath.Join(p.board, p.version, time.Now().Format(time.DateOnly), uuid, filepath.Base(tarFilePath)),
		}
		if err := p.storage.Write(ctx, tarFilePath, gsObject); err != nil {
			return err
		}
		// Next notify perf processing pipeline to process the perf.data
		if err := p.publish(ctx, p.bucket, gsObject.Object); err != nil {
			return err
		}
		// Next clean up the file on dut.
		log.Println("Deleting the perf file on dut " + dutFile)
		if _, err = p.exec.RunCmd(ctx, rm, []string{dutFile}, p.dut); err != nil {
			return err
		}
		// Next clean up the tar file on host.
		if err = os.Remove(tarFilePath); err != nil {
			return err
		}
	}
	return nil
}

func (p perfCollector) start(perfDir, workDir string) {
	log.Println("Starting to perf collection.")
	ctx := context.Background()
	uuid := uuid.New().String()
	for {
		select {
		case <-p.stopCollector:
			log.Println("Stopping perf collector.")
			p.terminated <- true
			return
		case <-time.After(p.perfCollectorInterval):
			if err := p.gatherPerf(ctx, perfDir, uuid, workDir); err != nil {
				log.Printf("error in perf collector %v", err)
			}
		}
	}
}
