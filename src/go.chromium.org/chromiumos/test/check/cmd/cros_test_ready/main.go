// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package main implements the cros-test-finder for finding tests based on tags.
package main

import (
	"context"
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/subcommands"
	"google.golang.org/protobuf/encoding/protojson"

	"go.chromium.org/chromiumos/config/go/test/api"
)

const (
	defaultConfigPath = "/usr/local/etc/cros_test_ready_config.jsonpb"
)

// createLogFile creates a file and its parent directory for logging purpose.
func createLogFile(fullPath string) (*os.File, error) {
	if fullPath == "" {
		t := time.Now()
		fullPath = fmt.Sprintf("/var/tmp/cros_test_ready/log_%s.txt", t.Format("20060102150405"))
	}
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %v: %w", fullPath, err)
	}

	logFullPathName := filepath.Join(fullPath, "log.txt")

	// Log the full output of the command to disk.
	logFile, err := os.Create(logFullPathName)
	if err != nil {
		return nil, fmt.Errorf("failed to create file %v: %w", fullPath, err)
	}
	return logFile, nil
}

// newLogger creates a logger. Using go default logger for now.
func newLogger(logFile *os.File) *log.Logger {
	var writer io.Writer = os.Stderr
	if logFile != nil {
		writer = io.MultiWriter(logFile, os.Stderr)
	}
	return log.New(writer, "", log.LstdFlags|log.LUTC)
}

// readInput reads a CrosTestReadyConfig jsonproto file and returns a pointer to RunTestsRequest.
func readInput(fileName string) (*api.CrosTestReadyConfig, error) {
	f, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("fail to read file %v: %v", fileName, err)
	}
	req := api.CrosTestReadyConfig{}
	if err := protojson.Unmarshal(f, &req); err != nil {
		return nil, fmt.Errorf("fail to unmarshal file %v: %v", fileName, err)
	}
	return &req, nil
}

// Version is the version info of this command. It is filled in during emerge.
var Version = "<unknown>"

// checkCmd implements subcommands.Command to check
// if a DUT is test ready to be used for testing.
type checkCmd struct {
	config      string
	logFile     string
	exitOnError bool
	logger      *log.Logger
}

// newCheckCmd returns a new checkCmd that will check
// if a DUT is test ready to be used for testing.
func newCheckCmd() *checkCmd {
	return &checkCmd{}
}

func (*checkCmd) Name() string     { return "check" }
func (*checkCmd) Synopsis() string { return "check" }
func (*checkCmd) Usage() string {
	return `Usage: generate [flag]...

Description:
    Check whether a DUT is test ready.
	It will exit with status 0 if all checks are passed.
	Otherwise, it will exit with status 1.

Flag:
`
}

func (c *checkCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.config,
		"config", defaultConfigPath,
		"Specify the jsonpb configuration file for what are needed to be checked on the DUT.")
	f.StringVar(&c.logFile,
		"log", "",
		"Specify the log file path (default: /var/tmp/cros_test_ready/log_<YYYYmmDDHHMMSS>.txt)")
	f.BoolVar(&c.exitOnError,
		"exit_on_error",
		true,
		"Specify whether cros_test_ready should exit when it encounters the first error.")

}

func (c *checkCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	fullLog, err := createLogFile(c.logFile)
	if err != nil {
		c.logger = newLogger(nil)
		c.logger.Printf("Failed to create log file %s; use stderr only: %v", c.logFile, err)
	} else {
		c.logger = newLogger(fullLog)
		defer fullLog.Close()
	}

	checkConfig, err := readInput(c.config)
	if err != nil {
		c.logger.Fatalf("Failed to read configuration %s: %v", c.config, err)
	}
	hasError := false

	for _, fileChecksum := range checkConfig.GetChecksums() {
		checksum, err := md5Str(fileChecksum.GetKey())
		if err != nil {
			c.logger.Printf("Failed to get checksum for file %s: %v", fileChecksum.GetKey(), err)
			hasError = true
		}
		if checksum != fileChecksum.GetValue() {
			c.logger.Printf("Failed to get expected checksum for file %s: got %q; wanted %q",
				fileChecksum.GetKey(), checksum, fileChecksum.GetValue())
			hasError = true
		}
		if c.exitOnError && hasError {
			return subcommands.ExitFailure
		}
	}
	return subcommands.ExitSuccess
}

func md5Str(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %v", filename, err)
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to calculate checksum for file %s: %v", filename, err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func mainInternal(ctx context.Context) int {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")
	subcommands.Register(newCheckCmd(), "")

	version := flag.Bool("version", false, "print version and exit")

	flag.Parse()

	if *version {
		fmt.Printf("cros_test_ready version %s\n", Version)
		return 0
	}

	return int(subcommands.Execute(ctx))
}

func main() {
	os.Exit(mainInternal(context.Background()))
}
