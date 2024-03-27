// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package main implements the post-process for finding tests based on tags.
package server

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"go.chromium.org/chromiumos/test/post_process/cmd/post-process/common"
	"chromiumos/test/util/portdiscovery"

	"go.chromium.org/chromiumos/config/go/test/api"
)

const (
	defaultRootPath = "/tmp/filters/%s"
)

// Version is the version info of this command. It is filled in during emerge.
var Version = "<unknown>"
var defaultPort = 0

type args struct {
	// Common input params.
	logPath   string
	inputPath string
	output    string
	version   bool

	// Server mode params
	port        int
	dutendpoint string
}

// startServer is the entry point for running post-process (TestFinderService) in server mode.
func startServer(d []string, executor func(req *api.InternalTestplan, log *log.Logger) (*api.InternalTestplan, error), name string) error {
	a := args{}
	t := time.Now()
	defaultLogPath := filepath.Join(fmt.Sprintf(defaultRootPath, name), t.Format("20060102-150405"))
	fs := flag.NewFlagSet("Run cros-provision-request-generator", flag.ExitOnError)
	fs.StringVar(&a.logPath, "log", defaultLogPath, fmt.Sprintf("Path to record finder logs. Default value is %s", defaultLogPath))
	fs.IntVar(&a.port, "port", defaultPort, fmt.Sprintf("Specify the port for the server. Default value %d.", defaultPort))

	fs.Parse(d)

	logFile, err := common.CreateLogFile(a.logPath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %s", err)
	}
	defer logFile.Close()

	logger := common.NewLogger(logFile)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("failed to create a net listener: %s", err)
	}
	// Write port number to ~/.cftmeta for go/cft-port-discovery
	err = portdiscovery.WriteServiceMetadata(name, l.Addr().String(), logger)
	if err != nil {
		return fmt.Errorf("failed to write metadata port: %s", err)
	}
	server, closer := NewServer(logger, executor)
	defer closer()

	err = server.Serve(l)
	if err != nil {
		return fmt.Errorf("failed to initialize server: %s", err)
	}
	logger.Println("Starting generic filter service on port ", a.port)

	return nil
}

// Server starts the generic filter server.
func Server(executor func(req *api.InternalTestplan, log *log.Logger) (*api.InternalTestplan, error), name string) error {
	return startServer(os.Args[2:], executor, name)
}
