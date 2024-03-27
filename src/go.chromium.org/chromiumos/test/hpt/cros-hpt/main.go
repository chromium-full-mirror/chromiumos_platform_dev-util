// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package implements cros-hpt used for High Performance Tracing while running tests.
package main

import (
	"chromiumos/test/util/helpers"
	"chromiumos/test/util/portdiscovery"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/grpc"
)

const (
	defaultRootPath          = "/tmp/cros-hpt"
	defaultAuthTokenFilePath = "/tmp/cros-hpt/authToken.txt"
	defaultPort              = 8010
	logPath                  = "log"
	// TODO(b/327275996): Replace brya with board in /etc/os-release file on dut.
	brya = "brya"
)

var (
	now     = time.Now().Format("20060102-150405")
	workDir = filepath.Join("/tmp/cros-hpt", now)
)

type flags struct {
	// Common input params.
	logPath         string
	version         bool
	port            int
	authFilePath    string
	crosDutEndpoint string
	projectId       string
	topicId         string
	bucket          string
	artifactsBucket string
}

func startServer(ctx context.Context, args []string) int {
	defaultLogDir := filepath.Join(workDir, logPath)
	f := flags{}
	fs := flag.NewFlagSet("Run cros-hpt", flag.ExitOnError)
	fs.StringVar(&f.logPath, logPath, defaultLogDir, fmt.Sprintf("Path to record finder logs. Default value is %s", defaultLogDir))
	fs.IntVar(&f.port, "port", defaultPort, fmt.Sprintf("Specify the port for the server. Default value %d.", defaultPort))
	fs.StringVar(&f.authFilePath, "authFilePath", defaultAuthTokenFilePath, fmt.Sprintf("Path to auth token file. Default is %s", defaultAuthTokenFilePath))
	fs.StringVar(&f.crosDutEndpoint, "crosDutEndpoint", "", fmt.Sprintf("Endpoint for cros dut."))
	fs.StringVar(&f.projectId, "projectId", "", fmt.Sprintf("GCP projectID for perf processing pipeline."))
	fs.StringVar(&f.topicId, "topicId", "", fmt.Sprintf("GCP pubsub topic for notifying perf processing pipeline about newly available data."))
	fs.StringVar(&f.bucket, "bucket", "", fmt.Sprintf("GCS bucket where to upload the zipped perf data."))
	fs.StringVar(&f.artifactsBucket, "artifactsBucket", "", fmt.Sprintf("GCS Bucket where hpt artifacts can be found."))

	fs.Parse(args)
	// Configure log file
	logFile, err := helpers.CreateLogFile(f.logPath)
	if err != nil {
		log.Fatalln("Failed to create log file", err)
		return 2
	}
	defer logFile.Close()
	logger := helpers.NewLogger(logFile)

	// Read and validate auth token required by vm-provision
	authTokenFilePath := defaultAuthTokenFilePath
	if f.authFilePath != "" {
		authTokenFilePath = f.authFilePath
	}
	// Set up gRPC connection
	conn, err := grpc.Dial(f.crosDutEndpoint, grpc.WithInsecure())
	if err != nil {
		logger.Fatalln("Failed to connect: ", err)
	}
	defer conn.Close()
	// Create gRPC dut
	dut := api.NewDutServiceClient(conn)
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", f.port))
	if err != nil {
		logger.Fatalln("Failed to create a net listener: ", err)
		return 2
	}
	logger.Println("Starting Service on port ", f.port)
	perfServer, err := NewServer(ctx, logger, &dut, authTokenFilePath, f.projectId, f.topicId, f.bucket, workDir, f.artifactsBucket)
	if err != nil {
		logger.Fatal(err)
		return 2
	}
	// Write port number to ~/.cftmeta for go/cft-port-discovery
	err = portdiscovery.WriteServiceMetadata("cros-hpt", l.Addr().String(), logger)
	if err != nil {
		logger.Println("Warning: error when writing to metadata file: ", err)
	}
	grpcServer := grpc.NewServer()
	api.RegisterGenericServiceServer(grpcServer, perfServer)
	err = grpcServer.Serve(l)
	if err != nil {
		logger.Fatalln("Failed to initialize server: ", err)
		return 2
	}
	return 0
}

func mainInternal(ctx context.Context) int {
	return startServer(ctx, os.Args[1:])
}

func main() {
	os.Exit(mainInternal(context.Background()))
}
