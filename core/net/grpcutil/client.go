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

	"google.golang.org/grpc"
)

// ClientTask is invoked with an open grpc connection by Client.
type ClientTask func(context.Context, *grpc.ClientConn) error

// Dial connects to a grpc server, and returns the connection.
// It also installs the standard options we normally use.
func Dial(ctx context.Context, target string, options ...grpc.DialOption) (*grpc.ClientConn, error) {
	options = append([]grpc.DialOption{
		grpc.WithCompressor(grpc.NewGZIPCompressor()),
		grpc.WithDecompressor(grpc.NewGZIPDecompressor()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt32)),
	}, options...)
	return grpc.Dial(target, options...)
}

// Client dials a grpc server, and runs the task with the connection, and closes the connection
// again before returning.
// It also installs the standard options we normally use.
func Client(ctx context.Context, target string, task ClientTask, options ...grpc.DialOption) error {
	conn, err := Dial(ctx, target, options...)
	if err != nil {
		return err
	}
	defer conn.Close()
	return task(ctx, conn)
}
