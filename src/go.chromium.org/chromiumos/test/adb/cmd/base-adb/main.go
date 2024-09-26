// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package main implements the base-adb used to setup CrOS devices.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"go.chromium.org/chromiumos/test/adb/cmd/base-adb/server"

	"go.chromium.org/luci/common/errors"
)

const (
	helpDescription = `base-adb tool

The tool is allow to communicate with adb on the host.

Commands:
  server    Starting server and allow work with server by RPC calls. Mostly used for tests.
            usage: base-adb server -device {device_name} [-log-path /tmp/base-adb/] [-port 80]

  -version  Print version of lib.
  -help     Print this help.`
	defaultLogDirectory = "/tmp/base-adb/"
	defaultPort         = 80
)

// createLogFile creates a file and its parent directory for logging purpose.
func createLogFile(dir string) (*os.File, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %v: %v", dir, err)
	}
	logFilePath := filepath.Join(dir, "log.txt")
	// Log the full output of the command to disk.
	logFile, err := os.Create(logFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file %v: %v", logFilePath, err)
	}
	return logFile, nil
}

// newLogger creates a logger. Using go default logger for now.
func newLogger(logFile *os.File) *log.Logger {
	mw := io.MultiWriter(logFile, os.Stderr)
	return log.New(mw, "", log.LstdFlags|log.LUTC)
}

type args struct {
	// Common input params.
	logPath    string
	deviceName string
	port       int
}

func (a args) validate() error {
	if a.deviceName == "" {
		return errors.Reason("device name is not specified").Err()
	}
	return nil
}

func startServer(d []string) int {
	a := args{}
	fs := flag.NewFlagSet("Start server", flag.ExitOnError)
	fs.StringVar(&a.logPath, "log-path", defaultLogDirectory, fmt.Sprintf("Path to record execution logs. Default value is %s", defaultLogDirectory))
	fs.StringVar(&a.deviceName, "device", "", "Specify device name for target device.")
	fs.IntVar(&a.port, "port", defaultPort, fmt.Sprintf("Specify the port for the server. Default value %d.", defaultPort))
	fs.Parse(d)

	logFile, err := createLogFile(a.logPath)
	if err != nil {
		log.Fatalln("Failed to create log file", err)
		return 2
	}
	defer logFile.Close()
	logger := newLogger(logFile)
	if err := a.validate(); err != nil {
		log.Fatalf("Validate input fail: %s", err)
		return 2
	}
	p, err := server.New(logger, a.deviceName)
	if err != nil {
		logger.Fatalln("Failed to create server: ", err)
		return 2
	}
	if err := p.StartServer(a.port); err != nil {
		logger.Fatalln("Failed to start server: ", err)
		return 1
	}
	return 0
}

// Specify run mode for CLI.
type runMode string

const (
	runCli     runMode = "cli"
	runServer  runMode = "server"
	runVersion runMode = "version"
	runHelp    runMode = "help"
)

func getRunMode() (runMode, error) {
	if len(os.Args) > 1 {
		for _, a := range os.Args {
			if a == "-version" {
				return runVersion, nil
			}
		}
		switch strings.ToLower(os.Args[1]) {
		case "cli":
			return runCli, nil
		case "server":
			return runServer, nil
		}
	}
	// If we did not find special run mode then just print help for user.
	return runHelp, nil
}

func mainInternal(ctx context.Context) int {
	rm, err := getRunMode()
	if err != nil {
		log.Fatalln(err)
		return 2
	}
	switch rm {
	case runCli:
		log.Printf("Running CLI is not supported yet!")
		return 2
	case runServer:
		log.Printf("Running server mode!")
		return startServer(os.Args[2:])
	case runVersion:
		log.Printf("base-adb version: %s", server.Version)
		return 0
	case runHelp:
		fallthrough
	default:
		log.Printf(helpDescription)
		return 0
	}
}

func main() {
	os.Exit(mainInternal(context.Background()))
}
