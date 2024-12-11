// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package server

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/test/post_process/cmd/post-process/common"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// GenericFilterServiceServer ...
type GenericFilterServiceServer struct {
	logPath      string
	name         string
	serverLogger *log.Logger
	commonParams *CommonFilterParams
	executor     func(req *api.InternalTestplan, log *log.Logger, commonParams *CommonFilterParams) (*api.InternalTestplan, error)
}

// NewServer creates an execution server.
func NewServer(logger *log.Logger, logPath, name string, commonParams *CommonFilterParams, executor func(req *api.InternalTestplan, log *log.Logger, commonParams *CommonFilterParams) (*api.InternalTestplan, error)) (*grpc.Server, func()) {
	s := &GenericFilterServiceServer{
		logPath:      logPath,
		name:         name,
		serverLogger: logger,
		commonParams: commonParams,
		executor:     executor,
	}

	server := grpc.NewServer(grpc.MaxRecvMsgSize(1024*1024*32), grpc.MaxSendMsgSize(1024*1024*32))
	var conns []*grpc.ClientConn
	closer := func() {
		for _, conn := range conns {
			conn.Close()
		}
		conns = nil
	}
	api.RegisterGenericFilterServiceServer(server, s)
	// Register reflection service on gRPC server.
	reflection.Register(server)

	logger.Println("filterService listening for requests")
	return server, closer
}

// Execute executes s.executor() with the req. The executor method must be provided on startup to NewServer.
func (s *GenericFilterServiceServer) Execute(ctx context.Context, req *api.InternalTestplan) (*api.InternalTestplan, error) {
	t := time.Now()
	logPath := filepath.Join(s.logPath, req.GetSuiteInfo().GetSuiteRequest().GetTestSuite().GetName(), s.name, t.Format("20060102-150405"))
	s.serverLogger.Printf("Creating Log File at %s", logPath)
	logFile, err := common.CreateLogFile(logPath)
	if err != nil {
		err = fmt.Errorf("failed to create log file: %s", err)
		s.serverLogger.Println(err.Error())
		return req, err
	}
	defer logFile.Close()
	logger := common.NewLogger(logFile)

	logger.Printf("Received Request: %s", req)

	rspn, err := s.executor(req, logger, s.commonParams)
	if err != nil {
		return nil, errors.Annotate(err, "Executor: failed to run").Err()
	}
	logger.Printf("Execute RPC Command was successful")

	return rspn, nil
}
