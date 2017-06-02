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

package trace

import (
	"context"

	"github.com/google/gapid/test/robot/job/worker"
	"github.com/google/gapid/test/robot/search"
	"google.golang.org/grpc"

	xctx "golang.org/x/net/context"
)

type server struct {
	manager Manager
}

// Serve wraps a manager in a grpc server.
func Serve(ctx context.Context, grpcServer *grpc.Server, manager Manager) error {
	RegisterServiceServer(grpcServer, &server{manager: manager})
	return nil
}

// Search implements ServiceServer.Search
// It delegates the call to the provided Manager implementation.
func (s *server) Search(query *search.Query, stream Service_SearchServer) error {
	ctx := stream.Context()
	return s.manager.Search(ctx, query, func(ctx context.Context, e *Action) error { return stream.Send(e) })
}

// Register implements ServiceServer.Register
// It delegates the call to the provided Manager implementation.
func (s *server) Register(request *worker.RegisterRequest, stream Service_RegisterServer) error {
	ctx := stream.Context()
	return s.manager.Register(ctx, request.Host, request.Target, func(ctx context.Context, t *Task) error { return stream.Send(t) })
}

// Do implements ServiceServer.Do
// It delegates the call to the provided Manager implementation.
func (s *server) Do(ctx xctx.Context, request *DoRequest) (*worker.DoResponse, error) {
	id, err := s.manager.Do(ctx, request.Device, request.Input)
	return &worker.DoResponse{Id: id}, err
}

// Update implements ServiceServer.Update
// It delegates the call to the provided Manager implementation.
func (s *server) Update(ctx xctx.Context, request *UpdateRequest) (*worker.UpdateResponse, error) {
	if err := s.manager.Update(ctx, request.Action, request.Status, request.Output); err != nil {
		return nil, err
	}
	return &worker.UpdateResponse{}, nil
}
