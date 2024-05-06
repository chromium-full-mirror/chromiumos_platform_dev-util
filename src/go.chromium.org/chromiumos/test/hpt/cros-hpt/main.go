// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package implements cros-hpt used for High Performance Tracing while running tests.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"go.chromium.org/chromiumos/test/util/helpers"
	"go.chromium.org/chromiumos/test/util/portdiscovery"

	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/grpc"
)

const (
	defaultRootPath = "/tmp/cros-hpt"
	defaultPort     = 8010
	logPath         = "log"
	// TODO(b/327275996): Replace brya with board in /etc/os-release file on dut.
	brya = "brya"
)

var (
	now     = time.Now().Format("20060102-150405")
	workDir = filepath.Join(defaultRootPath, now)
)

type flags struct {
	logPath string
	port    int
}

func startServer(ctx context.Context, args []string) int {
	defaultLogDir := filepath.Join(workDir, logPath)
	f := flags{}
	fs := flag.NewFlagSet("Run cros-hpt", flag.ExitOnError)
	fs.StringVar(&f.logPath, logPath, defaultLogDir, fmt.Sprintf("Path to record finder logs. Default value is %s", defaultLogDir))
	fs.IntVar(&f.port, "port", defaultPort, fmt.Sprintf("Specify the port for the server. Default value %d.", defaultPort))

	fs.Parse(args)
	// Configure log file
	logFile, err := helpers.CreateLogFile(f.logPath)
	if err != nil {
		log.Fatalln("Failed to create log file", err)
		return 2
	}
	defer logFile.Close()
	logger := helpers.NewLogger(logFile)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", f.port))
	if err != nil {
		logger.Fatalln("Failed to create a net listener: ", err)
		return 2
	}
	logger.Println("Starting Service on port ", f.port)
	perfServer := NewServer(ctx, logger, workDir)

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
