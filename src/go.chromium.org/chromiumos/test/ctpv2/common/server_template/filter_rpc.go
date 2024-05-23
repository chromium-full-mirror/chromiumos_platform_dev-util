// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package server

import (
	"context"
	"log"

	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc"
)

// GenericFilterServiceServer ...
type GenericFilterServiceServer struct {
	logger   *log.Logger
	executor func(req *api.InternalTestplan, log *log.Logger) (*api.InternalTestplan, error)
}

// NewServer creates an execution server.
func NewServer(logger *log.Logger, executor func(req *api.InternalTestplan, log *log.Logger) (*api.InternalTestplan, error)) (*grpc.Server, func()) {
	s := &GenericFilterServiceServer{
		logger:   logger,
		executor: executor,
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

	logger.Println("filterService listening for requests")
	return server, closer
}

// Execute executes s.executor() with the req. The executor method must be provided on startup to NewServer.
func (s *GenericFilterServiceServer) Execute(ctx context.Context, req *api.InternalTestplan) (*api.InternalTestplan, error) {
	s.logger.Printf("Received Request: %s", req)

	rspn, err := s.executor(req, s.logger)
	if err != nil {
		return nil, errors.Annotate(err, "Executor: failed to run").Err()
	}
	s.logger.Printf("Execute RPC Command was successful")

	return rspn, nil
}
