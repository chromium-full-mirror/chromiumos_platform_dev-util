// Copyright 2020 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package lroold_test

import (
	"context"
	"net"
	"testing"
	"time"

	"go.chromium.org/chromiumos/lroold"

	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"go.chromium.org/chromiumos/config/go/api/test/tls/dependencies/longrunning"
	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type exampleServer struct {
	tls.UnimplementedCommonServer
	*lroold.Manager
}

func (s *exampleServer) Serve(l net.Listener) error {
	s.Manager = lroold.New()
	defer s.Manager.Close()
	server := grpc.NewServer()
	tls.RegisterCommonServer(server, s)
	longrunning.RegisterOperationsServer(server, s.Manager)
	return server.Serve(l)
}

func (s *exampleServer) RunTests(ctx context.Context, req *api.CrosTestRequest) (*longrunning.Operation, error) {
	op := s.Manager.NewOperation()
	go s.RunTests(ctx, req)
	return op, nil
}

func RunServer() {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	s := exampleServer{}
	if err := s.Serve(l); err != nil {
		panic(err)
	}
}

func TestServe(t *testing.T) {
	go RunServer()
	// TODO(shapiroc): Add testing
}

func TestWaitOperationTimeout(t *testing.T) {
	m := lroold.New()
	defer m.Close()

	t.Run("seconds timeout blocks until elapsed", func(t *testing.T) {
		op := m.NewOperation()
		start := time.Now()
		got, err := m.WaitOperation(context.Background(), &longrunning.WaitOperationRequest{
			Name:    op.Name,
			Timeout: durationpb.New(time.Second),
		})
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("WaitOperation failed: %v", err)
		}
		if got.GetDone() {
			t.Fatalf("expected operation to still be pending, got Done=true")
		}
		if elapsed < 950*time.Millisecond {
			t.Fatalf("WaitOperation returned after %v, want >= 950ms", elapsed)
		}
	})

	t.Run("sub-second nanos timeout is respected", func(t *testing.T) {
		op := m.NewOperation()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		start := time.Now()
		got, err := m.WaitOperation(ctx, &longrunning.WaitOperationRequest{
			Name:    op.Name,
			Timeout: durationpb.New(50 * time.Millisecond),
		})
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("WaitOperation failed: %v", err)
		}
		if got.GetDone() {
			t.Fatalf("expected operation to still be pending, got Done=true")
		}
		if ctx.Err() != nil {
			t.Fatalf("WaitOperation ignored sub-second Timeout and hit parent context deadline: %v", ctx.Err())
		}
		if elapsed < 45*time.Millisecond {
			t.Fatalf("WaitOperation returned after %v, want >= 45ms", elapsed)
		}
	})

	t.Run("returns immediately on completion before timeout", func(t *testing.T) {
		op := m.NewOperation()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		timer := time.AfterFunc(20*time.Millisecond, func() {
			_ = m.SetResult(op.Name, &emptypb.Empty{})
		})
		defer timer.Stop()

		got, err := m.WaitOperation(ctx, &longrunning.WaitOperationRequest{
			Name:    op.Name,
			Timeout: durationpb.New(30 * time.Second),
		})
		if err != nil {
			t.Fatalf("WaitOperation failed: %v", err)
		}
		if !got.GetDone() {
			t.Fatalf("expected Done=true after SetResult")
		}
	})
}
