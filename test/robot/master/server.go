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

package master

import (
	"context"

	"github.com/google/gapid/test/robot/search"

	"google.golang.org/grpc"

	xctx "golang.org/x/net/context"
)

type server struct {
	master  Master
	restart bool
}

// Serve wraps a Master in a grpc server.
func Serve(ctx context.Context, grpcServer *grpc.Server, m Master) error {
	s := &server{master: m}
	RegisterServiceServer(grpcServer, s)
	return nil
}

// Search implements ServiceServer.Search
// It delegates the call to the provided Master implementation.
func (s *server) Search(query *search.Query, stream Service_SearchServer) error {
	ctx := stream.Context()
	return s.master.Search(ctx, query, func(ctx context.Context, e *Satellite) error { return stream.Send(e) })
}

// Orbit implements ServiceServer.Orbit
// It delegates the call to the provided Master implementation.
func (s *server) Orbit(request *OrbitRequest, stream Service_OrbitServer) error {
	ctx := stream.Context()
	return s.master.Orbit(ctx, *request.Services,
		func(ctx context.Context, command *Command) error { return stream.Send(command) },
	)
}

// Shutdown implements ServiceServer.Shutdown
// It delegates the call to the provided Master implementation.
func (s *server) Shutdown(ctx xctx.Context, request *ShutdownRequest) (*ShutdownResponse, error) {
	return s.master.Shutdown(ctx, request)
}
