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

	"github.com/google/gapid/core/event"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/test/robot/search"
	"google.golang.org/grpc"
)

type remote struct {
	client ServiceClient
}

// NewRemote returns a job manager that talks to a remote grpc job manager.
func NewRemote(ctx context.Context, conn *grpc.ClientConn) Manager {
	return &remote{client: NewServiceClient(conn)}
}

// SearchDevices implements Manager.SearchDevicess
// It forwards the call through grpc to the remote implementation.
func (m *remote) SearchDevices(ctx context.Context, query *search.Query, handler DeviceHandler) error {
	stream, err := m.client.SearchDevices(ctx, query)
	if err != nil {
		return err
	}
	return event.Feed(ctx, event.AsHandler(ctx, handler), grpcutil.ToProducer(stream))
}

// SearchWorkers implements Manager.SearchWorkers
// It forwards the call through grpc to the remote implementation.
func (m *remote) SearchWorkers(ctx context.Context, query *search.Query, handler WorkerHandler) error {
	stream, err := m.client.SearchWorkers(ctx, query)
	if err != nil {
		return err
	}
	return event.Feed(ctx, event.AsHandler(ctx, handler), grpcutil.ToProducer(stream))
}

// GetWorker implements Manager.GetWorker
// It forwards the call through grpc to the remote implementation.
func (m *remote) GetWorker(ctx context.Context, host *device.Instance, target *device.Instance, op Operation) (*Worker, error) {
	request := &GetWorkerRequest{Host: host, Target: target, Operation: op}
	response, err := m.client.GetWorker(ctx, request)
	if err != nil {
		return nil, err
	}
	return response.Worker, nil
}
