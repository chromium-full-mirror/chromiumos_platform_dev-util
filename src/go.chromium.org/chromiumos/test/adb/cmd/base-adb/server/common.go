// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package server implement adb-service API.
package server

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/test/util/portdiscovery"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc"
)

const (
	// Version is version of the server.
	Version = "0.1"
)

// Server to run adb service.
type Server interface {
	StartServer(port int) error
}

// adbService holds info for server.
type adbService struct {
	logger     *log.Logger
	deviceName string
}

// New creates new server to perform.
func New(logger *log.Logger, deviceName string) (Server, error) {
	return &adbService{
		logger:     logger,
		deviceName: deviceName,
	}, nil
}

// createLogFile creates a file and its parent directory for logging purpose.
func createLogFile() (*os.File, error) {
	t := time.Now()
	fullPath := filepath.Join("/tmp/base-adb/", t.Format("20060102-150405"))
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %v: %v", fullPath, err)
	}

	logFullPathName := filepath.Join(fullPath, "log.txt")

	// Log the full output of the command to disk.
	logFile, err := os.Create(logFullPathName)
	if err != nil {
		return nil, fmt.Errorf("failed to create file %v: %v", fullPath, err)
	}
	return logFile, nil
}

// newLogger creates a logger. Using go default logger for now.
func newLogger(logFile *os.File) *log.Logger {
	mw := io.MultiWriter(logFile, os.Stderr)
	newLog := log.New(mw, "<base-adb>: ", log.LstdFlags|log.LUTC)
	newLog.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmsgprefix)
	return newLog
}

// startServer starts service on requested port.
func (s *adbService) StartServer(port int) error {
	logFile, err := createLogFile()
	if err != nil {
		log.Fatalln("Failed to create log file: ", err)
	}
	defer logFile.Close()
	logger := newLogger(logFile)
	logger.Println("Starting base-adb version ", Version)
	logger.Println("Starting base-adb on port ", port)
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return errors.Annotate(err, "start server: create listener at %d", port).Err()
	}
	// Log the port _after_ the conn is established server.
	logger.Println("Started server on address ", l.Addr().String())
	logger.Println("Continue")

	// Write port number to ~/.cftmeta for go/cft-port-discovery
	if err := portdiscovery.WriteServiceMetadata("base-adb", l.Addr().String(), logger); err != nil {
		logger.Println("Warning: error when writing to metadata file: ", err)
	}
	srv := grpc.NewServer()
	api.RegisterADBServiceServer(srv, s)
	s.logger.Println("service listen to request at ", l.Addr().String())
	if err := srv.Serve(l); err != nil {
		return err
	}
	logger.Println("Server stopped!")
	return err
}
