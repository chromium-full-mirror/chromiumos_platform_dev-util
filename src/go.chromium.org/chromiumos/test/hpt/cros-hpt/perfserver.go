// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go.chromium.org/chromiumos/test/publish/cmd/publishserver/storage"
	"go.chromium.org/chromiumos/test/util/common"

	"cloud.google.com/go/pubsub"
	"github.com/google/uuid"
	"google.golang.org/api/option"
	"google.golang.org/grpc"

	"go.chromium.org/chromiumos/config/go/test/api"
)

const (
	perfCollectorInterval = time.Second * 10
	healthCheckInterval   = time.Second * 30
	waitTimout            = time.Second * 30
	perfInitPollInterval  = time.Millisecond * 100
	perfInitWaitTimeout   = time.Second * 1
)

type perfServer struct {
	exec          execUtil
	logger        *log.Logger
	dut           *api.DutServiceClient
	storageClient storageWriter
	healthMonitor *monitor
	perfCollector *perfCollector
	perfCmd       *perfCmd
	stopMonitor   chan bool
	workDir       string
}

type dutFileFetcher struct{}

func (dutFileFetcher) Copy(ctx context.Context, file string, dut api.DutServiceClient) (string, error) {
	return common.GetFile(ctx, file, dut)
}

type dutExecUtil struct{}

func (dutExecUtil) RunCmd(ctx context.Context, cmd string, args []string, dut api.DutServiceClient) (string, error) {
	return common.RunCmd(ctx, cmd, args, dut)
}

func (dutExecUtil) RunCmdAsync(ctx context.Context, cmd string, args []string, dut api.DutServiceClient) error {
	log.Printf("Run cmd: %s, %s\n", cmd, args)
	req := api.ExecCommandRequest{
		Command: cmd,
		Args:    args,
	}
	// TODO(b:326288495) Use appropriate Async API when its available.
	// Commands like perf-record are non-terminating in nature. They will
	// block the session and keep on running until a kill signal is given.
	// For these commands, there is no point in calling Recv() on ExecCommand's response.
	// Therefore just ignore the retun value of the ExecCommand.
	if _, err := dut.ExecCommand(ctx, &req); err != nil {
		log.Printf("Run cmd FAILED: %s\n", err)
		return err
	}
	return nil
}

func NewServer(ctx context.Context, logger *log.Logger, workDir string) *perfServer {
	return &perfServer{
		logger:  logger,
		workDir: workDir,
	}
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
	_, err := ps.exec.RunCmd(ctx, rm, []string{"-rf", outputPerfDir}, *ps.dut)
	if err != nil && strings.Contains(err.Error(), noDirMsg) {
		return nil
	}
	return err
}

func (ps *perfServer) setupDUTConnection(dutIP string) (api.DutServiceClient, error) {
	conn, err := grpc.Dial(dutIP, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	defer conn.Close()

	return api.NewDutServiceClient(conn), nil
}

func (ps *perfServer) setupPublisher(ctx context.Context, topic, project, credsFile string) (*common.Publisher, error) {
	pubsubClient, err := pubsub.NewClient(ctx, project, option.WithCredentialsFile(credsFile))
	if err != nil {
		return nil, err
	}
	return &common.Publisher{
		Topic: pubsubClient.Topic(topic),
	}, nil
}

// setupPerfCollector sets up all the necessary plumbing before we can actually start perf.
// This includes setting up the clients so that we can store and publish data.
// Additionally, this makes sure we have an active connection to the DUT.
func (ps *perfServer) setupPerfCollector(ctx context.Context, msg *api.GenericMessage) (*perfCollector, error) {
	args := msg.GetValues()
	credentialsFile := string(args["authTokenFilePath"].GetValue())

	sc, err := storage.NewStorageClientWithCredsFile(ctx, credentialsFile)
	if err != nil {
		return nil, err
	}

	project := string(args["projectId"].GetValue())
	topic := string(args["topicId"].GetValue())
	pub, err := ps.setupPublisher(ctx, topic, project, credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("could not setup publisher for project: %s and topic: %s due to err: %w", project, topic, err)
	}

	dutIP := string(args["dutIP"].GetValue())
	dut, err := ps.setupDUTConnection(dutIP)
	if err != nil {
		return nil, fmt.Errorf("could not set up connection to DUT at IP %s: %w", dutIP, err)
	}

	exec := dutExecUtil{}
	// TODO(b/327285753): Throw error if DUT's version does not match active version.
	pc := perfCollector{
		fileFetcher:           dutFileFetcher{},
		exec:                  exec,
		dut:                   dut,
		perfCollectorInterval: perfCollectorInterval,
		tarUtil:               common.Zip{},
		storage:               sc,
		bucket:                string(args["bucket"].GetValue()),
		publisher:             pub,
		stopCollector:         make(chan bool),
		terminated:            make(chan bool),
		board:                 string(args["board"].GetValue()),
		version:               string(args["version"].GetValue()),
		snapshotVersion:       string(args["snapshotVersion"].GetValue()),
	}

	return &pc, nil
}

func (ps *perfServer) setupServer(ctx context.Context, msg *api.GenericMessage) error {
	pc, err := ps.setupPerfCollector(ctx, msg)
	if err != nil {
		return fmt.Errorf("could not set up perf collector: %w", err)
	}
	ps.perfCollector = pc
	ps.storageClient = pc.storage
	ps.stopMonitor = make(chan bool)
	ps.exec = pc.exec
	ps.dut = &pc.dut

	ps.perfCmd = &perfCmd{
		dut:                  *ps.dut,
		exec:                 ps.exec,
		perfInitPollInterval: perfInitPollInterval,
		perfInitWaitTimeout:  perfInitWaitTimeout,
	}

	ps.healthMonitor = &monitor{
		exec:                ps.exec,
		dut:                 *ps.dut,
		healthCheckInterval: healthCheckInterval,
		perf:                ps.perfCmd,
		stopMonitoring:      make(chan bool),
		terminated:          make(chan bool),
	}
	return nil
}

// Start is called by TestRunner before cros-test
func (ps *perfServer) Start(ctx context.Context, req *api.GenericStartRequest) (*api.GenericStartResponse, error) {
	log.Println("Received start request.")
	if fileInfo, err := os.Stat(ps.workDir); os.IsNotExist(err) || !fileInfo.IsDir() {
		return nil, fmt.Errorf("%s does not exist or is not a directory", ps.workDir)
	}

	// Setup perf collector so that we can start perf on DUT.
	if err := ps.setupServer(ctx, req.GetMessage()); err != nil {
		return nil, fmt.Errorf("unable to setup perf server: %w", err)
	}

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

	go ps.healthMonitor.start(pid)
	go ps.perfCollector.start(outputPerfDir, ps.workDir)
	return &api.GenericStartResponse{}, nil
}

// Run is called by TestRunner after calling Start, but before cros-test
func (ps *perfServer) Run(ctx context.Context, _ *api.GenericRunRequest) (*api.GenericRunResponse, error) {
	return &api.GenericRunResponse{}, nil
}

// Stop is called by TestRunner after cros-test
func (ps *perfServer) Stop(ctx context.Context, _ *api.GenericStopRequest) (*api.GenericStopResponse, error) {
	log.Println("Received stop request.")
	// Stop monitoring.
	ps.healthMonitor.stopMonitoring <- true
	if err := waitForTermination(ps.healthMonitor.terminated, "Health Monitor"); err != nil {
		return nil, err
	}
	// Stop the Perf collector.
	ps.perfCollector.stopCollector <- true
	if err := waitForTermination(ps.perfCollector.terminated, "Perf Collector"); err != nil {
		return nil, err
	}
	// Kill the perf process.
	if err := ps.perfCmd.killAll(ctx); err != nil {
		return nil, err
	}
	// Previous step(killAll) resulted in new perf.data dumped on DUT.
	// Lets process them as well.
	if err := ps.perfCollector.gatherPerf(ctx, outputPerfDir, uuid.New().String(), ps.workDir); err != nil {
		return nil, err
	}
	if err := ps.cleanPerfDir(ctx); err != nil {
		return nil, err
	}
	return &api.GenericStopResponse{}, nil
}
