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

package job

import (
	"context"

	"github.com/google/gapid/test/robot/search"

	"google.golang.org/grpc"

	xctx "golang.org/x/net/context"
)

type server struct {
	manager Manager
}

// Serve wraps a manager in a grpc server.
func Serve(ctx xctx.Context, grpcServer *grpc.Server, manager Manager) error {
	RegisterServiceServer(grpcServer, &server{manager: manager})
	return nil
}

// SearchDevices implements ServiceServer.SearchDevicess
// It delegates the call to the provided Manager implementation.
func (s *server) SearchDevices(query *search.Query, stream Service_SearchDevicesServer) error {
	ctx := stream.Context()
	return s.manager.SearchDevices(ctx, query, func(ctx context.Context, e *Device) error { return stream.Send(e) })
}

// SearchWorkers implements ServiceServer.SearchWorkers
// It delegates the call to the provided Manager implementation.
func (s *server) SearchWorkers(query *search.Query, stream Service_SearchWorkersServer) error {
	ctx := stream.Context()
	return s.manager.SearchWorkers(ctx, query, func(ctx context.Context, e *Worker) error { return stream.Send(e) })
}

// GetWorker implements ServiceServer.GetWorker
// It delegates the call to the provided Manager implementation.
func (s *server) GetWorker(ctx xctx.Context, request *GetWorkerRequest) (*GetWorkerResponse, error) {
	d, err := s.manager.GetWorker(ctx, request.Host, request.Target, request.Operation)
	if err != nil {
		return nil, err
	}
	return &GetWorkerResponse{Worker: d}, nil
}
