// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"chromiumos/test/publish/cmd/publishserver/storage"
	"context"
	"log"

	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/grpc"
)

type perfServer struct {
	logger   *log.Logger
	dut      *api.DutServiceClient
	gsClient *storage.GSClient
}

func NewServer(ctx context.Context, logger *log.Logger, dut *api.DutServiceClient, gcpCredentials string) (*grpc.Server, func(), error) {
	gsClient, err := storage.NewGSClient(ctx, gcpCredentials)
	if err != nil {
		return nil, nil, err
	}

	server := &perfServer{
		logger:   logger,
		dut:      dut,
		gsClient: gsClient,
	}

	destructor := func() {
		server.gsClient.Close()
	}

	grpcServer := grpc.NewServer()

	//TODO: When the service proto is ready, call api.RegisterGenericServer().
	return grpcServer, destructor, nil
}

// Start is called by TestRunner before cros-test
func (ps *perfServer) Start(ctx context.Context) error {
	return nil
}

// Run is called by TestRunner after calling Start, but before cros-test
func (ps *perfServer) Run(ctx context.Context) error {
	return nil
}

// Close is called by TestRunner after cros-test
func (ps *perfServer) Close(ctx context.Context) error {
	return nil
}
