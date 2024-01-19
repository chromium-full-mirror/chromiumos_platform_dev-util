// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"chromiumos/test/publish/cmd/publishserver/storage"
	"chromiumos/test/util/common"
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"

	"go.chromium.org/chromiumos/config/go/test/api"
)

const (
	perfCollectorInterval = time.Minute * 1
	healthCheckInterval   = time.Second * 30
	waitTimout            = time.Second * 30
)

type perfServer struct {
	exec          execUtil
	logger        *log.Logger
	dut           *api.DutServiceClient
	storageClient *storage.StorageClient
	healthMonitor *monitor
	perfCollector *perfCollector
	perfCmd       *perfCmd
	stopMonitor   chan bool
}

type dutFileFetcher struct{}

func (dutFileFetcher) Copy(ctx context.Context, file string, dut api.DutServiceClient) (string, error) {
	return common.GetFile(ctx, file, dut)
}

type dutExecUtil struct{}

func (dutExecUtil) RunCmd(ctx context.Context, cmd string, args []string, dut api.DutServiceClient) (string, error) {
	return common.RunCmd(ctx, cmd, args, dut)
}

func NewServer(ctx context.Context, logger *log.Logger, dut *api.DutServiceClient, credentialsFile, projectId, topicId, bucket string) (*perfServer, func(), error) {
	storageClient, err := storage.NewStorageClientWithCredsFile(ctx, credentialsFile)
	if err != nil {
		return nil, nil, err
	}
	pubsubClient, err := pubsub.NewClient(ctx, projectId, option.WithCredentialsFile(credentialsFile))
	topic := pubsubClient.Topic(topicId)
	publisher := &common.Publisher{
		Topic: topic,
	}
	exec := dutExecUtil{}
	dc := perfCollector{
		fileFetcher:           dutFileFetcher{},
		exec:                  exec,
		dut:                   *dut,
		perfCollectorInterval: perfCollectorInterval,
		tarUtil:               common.Zip{},
		storage:               storageClient,
		bucket:                bucket,
		publisher:             publisher,
		stopCollector:         make(chan bool),
		terminated:            make(chan bool),
	}
	perfCmd := perfCmd{
		dut:  *dut,
		exec: exec,
	}
	healthMonitor := monitor{
		exec:                exec,
		dut:                 *dut,
		healthCheckInterval: healthCheckInterval,
		perf:                &perfCmd,
		stopMonitoring:      make(chan bool),
		terminated:          make(chan bool),
	}
	server := &perfServer{
		exec:          exec,
		logger:        logger,
		dut:           dut,
		storageClient: storageClient,
		healthMonitor: &healthMonitor,
		perfCollector: &dc,
		perfCmd:       &perfCmd,
		stopMonitor:   make(chan bool),
	}
	destructor := func() {
		server.storageClient.Close()
	}
	return server, destructor, nil
}

func waitForTermination(ch chan bool, processName string) error {
	log.Printf("\nWaiting for the %s to stop.", processName)
	for {
		select {
		case val := <-ch:
			if val {
				log.Printf("\n%s terminated.", processName)
				return nil
			} else {
				return fmt.Errorf("did not expect false value for terminate %s", processName)
			}
		case <-time.After(1 * time.Second):
			log.Print(".")
		case <-time.After(waitTimout):
			return fmt.Errorf("timedout while waiting for %s to stop", processName)
		}
	}
}

func (ps *perfServer) cleanPerfDir(ctx context.Context) error {
	_, err := ps.exec.RunCmd(ctx, rm, []string{filepath.Join(outputPerfDir, "*")}, *ps.dut)
	return err
}

// Start is called by TestRunner before cros-test
func (ps *perfServer) Start(ctx context.Context, _ *api.GenericStartRequest) (*api.GenericStartResponse, error) {

	// In rare event, previous runs might miss terminating perf and cleaning perf dir.
	// Lets do that before starting a new perf cmd.
	if err := ps.perfCmd.killAll(ctx); err != nil {
		return nil, err
	}
	if err := ps.cleanPerfDir(ctx); err != nil {
		return nil, err
	}

	pid, err := ps.perfCmd.startPerf(ctx)
	if err != nil {
		return nil, err
	}
	go ps.healthMonitor.start(ctx, pid)
	go ps.perfCollector.start(ctx, filepath.Join(outputPerfDir, perfData))
	return &api.GenericStartResponse{}, nil
}

// Run is called by TestRunner after calling Start, but before cros-test
func (ps *perfServer) Run(ctx context.Context, _ *api.GenericRunRequest) (*api.GenericRunResponse, error) {
	return &api.GenericRunResponse{}, nil
}

// Stop is called by TestRunner after cros-test
func (ps *perfServer) Stop(ctx context.Context, _ *api.GenericStopRequest) (*api.GenericStopResponse, error) {
	// Stop monitoring.
	ps.healthMonitor.stopMonitoring <- true
	if err := waitForTermination(ps.healthMonitor.terminated, "Health Monitor"); err != nil {
		return nil, err
	}

	// Stop the Perf collector.
	ps.perfCollector.stopCollector <- true
	if err := waitForTermination(ps.healthMonitor.terminated, "Perf Collector"); err != nil {
		return nil, err
	}

	// Kill the perf process.
	if err := ps.perfCmd.killAll(ctx); err != nil {
		return nil, err
	}

	// Previous step(killAll) resulted in new perf.data dumped on DUT.
	// Lets process them as well.
	if err := ps.perfCollector.gatherPerf(ctx, filepath.Join(outputPerfDir, perfData)); err != nil {
		return nil, err
	}
	if err := ps.cleanPerfDir(ctx); err != nil {
		return nil, err
	}
	return &api.GenericStopResponse{}, nil
}
