// Copyright (C) 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package grpcutil

import (
	"context"
	"math"
	"net"

	"github.com/google/gapid/core/log"
	"google.golang.org/grpc"
)

// PrepareTask is called to add the services to a grpc server before it starts running.
type PrepareTask func(context.Context, net.Listener, *grpc.Server) error

// Serve prepares and runs a grpc server on the specified address.
// It also installs the standard options we normally use.
func Serve(ctx context.Context, address string, prepare PrepareTask, options ...grpc.ServerOption) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.F(ctx, true, "Could not start grpc server. Error: %v", err)
	}
	return ServeWithListener(ctx, listener, prepare, options...)
}

// ServeWithListener prepares and runs a grpc server using the specified net.Listener.
// It also installs the standard options we normally use.
func ServeWithListener(ctx context.Context, listener net.Listener, prepare PrepareTask, options ...grpc.ServerOption) error {
	options = append([]grpc.ServerOption{
		grpc.RPCCompressor(grpc.NewGZIPCompressor()),
		grpc.RPCDecompressor(grpc.NewGZIPDecompressor()),
		grpc.MaxRecvMsgSize(math.MaxInt32),
	}, options...)
	defer listener.Close()
	grpcServer := grpc.NewServer(options...)
	if err := prepare(ctx, listener, grpcServer); err != nil {
		return err
	}
	log.I(ctx, "Starting grpc server")
	if err := grpcServer.Serve(listener); err != nil {
		return log.Errf(ctx, err, "Abort running grpc server: %v", listener.Addr())
	}
	log.I(ctx, "Shutting down grpc server")
	return nil
}
