// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"chromiumos/lro"
	"context"
	"io/ioutil"
	"log"
	"strings"

	"go.chromium.org/chromiumos/config/go/longrunning"
	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/anypb"
)

// VMLeaserServiceServer is implementation of vm_leaser.proto
type VMLeaserServiceServer struct {
	logger         *log.Logger
	vmleaserClient api.VMLeaserServiceClient
	manager        *lro.Manager
	token          string
}

// NewServer creates an execution server
func NewServer(vmleaserClient api.VMLeaserServiceClient, logger *log.Logger, token string) (*grpc.Server, func()) {
	s := &VMLeaserServiceServer{
		logger:         logger,
		vmleaserClient: vmleaserClient,
		manager:        lro.New(),
		token:          token,
	}
	server := grpc.NewServer()
	var conns []*grpc.ClientConn
	closer := func() {
		for _, conn := range conns {
			conn.Close()
		}
		conns = nil
	}
	api.RegisterGenericProvisionServiceServer(server, s)
	longrunning.RegisterOperationsServer(server, s.manager)
	reflection.Register(server)

	return server, closer
}

// Install calls the VMLeaser service
func (s *VMLeaserServiceServer) Install(ctc context.Context, req *api.InstallRequest) (*longrunning.Operation, error) {

	op := s.manager.NewOperation()
	authorization := "Bearer " + s.token
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("Authorization", authorization))

	leaseVMreq := &api.LeaseVMRequest{}
	if err := req.Metadata.UnmarshalTo(leaseVMreq); err != nil {
		s.logger.Printf("Invalid request, %s", err)
		return nil, err
	}

	// Make the gRPC request
	s.logger.Printf("Making grpc call")
	resp, err := s.vmleaserClient.LeaseVM(ctx, leaseVMreq)
	if err != nil {
		s.logger.Printf("Failed to make grpc request, %s", err)
		return nil, err
	}
	// Create an empty anypb.Any
	anyResp := &anypb.Any{}

	// Marshal the LeaseVMResponse to the anypb.Any
	if err := anyResp.MarshalFrom(resp); err != nil {
		s.logger.Printf("Failed to marshal response, %s", err)
		return nil, err
	}

	// Create the InstallResponse with the Metadata field set to anypb.Any
	provisionResp := &api.InstallResponse{
		Metadata: anyResp,
	}
	s.logger.Printf("gRPC request succeeded")
	s.manager.SetResult(op.Name, provisionResp)
	return op, nil
}

// readAuthToken reads the authorization token required for calling vm leaser service
func readAuthToken(authTokenFilePath string) (string, error) {
	content, err := ioutil.ReadFile(authTokenFilePath)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(content))

	return token, nil
}
