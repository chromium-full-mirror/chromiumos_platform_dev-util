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
	"google.golang.org/grpc/credentials"
)

const (
	defaultRootPath          = "/tmp/cros-hpt"
	defaultAuthTokenFilePath = "/tmp/cros-hpt/authToken.txt"
)

// Version is the version info of this command. It is filled in during emerge.
var Version = "<unknown>"
var defaultPort = 8010

type args struct {
	// Common input params.
	logPath         string
	version         bool
	port            int
	authFilePath    string
	crosDutEndpoint string
}

func startServer(ctx context.Context, d []string) int {
	a := args{}
	t := time.Now()
	defaultLogPath := filepath.Join(defaultRootPath, t.Format("20060102-150405"))
	fs := flag.NewFlagSet("Run cros-hpt", flag.ExitOnError)
	fs.StringVar(&a.logPath, "log", defaultLogPath, fmt.Sprintf("Path to record finder logs. Default value is %s", defaultLogPath))
	fs.IntVar(&a.port, "port", defaultPort, fmt.Sprintf("Specify the port for the server. Default value %d.", defaultPort))
	fs.StringVar(&a.authFilePath, "authFilePath", defaultAuthTokenFilePath, fmt.Sprintf("Path to auth token file. Default is %s", defaultAuthTokenFilePath))
	fs.StringVar(&a.crosDutEndpoint, "crosDutEndpoint", "", fmt.Sprintf("Endpoint for cros dut."))
	fs.Parse(d)

	// Configure log file
	logFile, err := helpers.CreateLogFile(a.logPath)
	if err != nil {
		log.Fatalln("Failed to create log file", err)
		return 2
	}
	defer logFile.Close()
	logger := helpers.NewLogger(logFile)

	// Read and validate auth token required by vm-provision
	authTokenFilePath := defaultAuthTokenFilePath
	if a.authFilePath != "" {
		authTokenFilePath = a.authFilePath
	}

	// Set up gRPC connection
	conn, err := grpc.Dial(a.crosDutEndpoint, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
	if err != nil {
		logger.Fatalln("Failed to connect: ", err)
	}
	defer conn.Close()

	// Create gRPC dut
	dut := api.NewDutServiceClient(conn)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		logger.Fatalln("Failed to create a net listener: ", err)
		return 2
	}

	logger.Println("Starting Service on port ", a.port)

	server, closer, err := NewServer(ctx, logger, &dut, authTokenFilePath)
	defer closer()

	// Write port number to ~/.cftmeta for go/cft-port-discovery
	err = portdiscovery.WriteServiceMetadata("cros-hpt", l.Addr().String(), logger)
	if err != nil {
		logger.Println("Warning: error when writing to metadata file: ", err)
	}
	err = server.Serve(l)
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
