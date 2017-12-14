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

package report

import (
	"context"

	"github.com/google/gapid/core/event"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/test/robot/job"
	"github.com/google/gapid/test/robot/job/worker"
	"github.com/google/gapid/test/robot/search"
	"google.golang.org/grpc"
)

type remote struct {
	client ServiceClient
}

// NewRemote returns a Worker that talks to a remote grpc report service.
func NewRemote(ctx context.Context, conn *grpc.ClientConn) Manager {
	return &remote{
		client: NewServiceClient(conn),
	}
}

// Search implements Manager.Search
// It forwards the call through grpc to the remote implementation.
func (m *remote) Search(ctx context.Context, query *search.Query, handler ActionHandler) error {
	stream, err := m.client.Search(ctx, query)
	if err != nil {
		return err
	}
	return event.Feed(ctx, event.AsHandler(ctx, handler), grpcutil.ToProducer(stream))
}

// Register implements Manager.Register
// It forwards the call through grpc to the remote implementation.
func (m *remote) Register(ctx context.Context, host *device.Instance,
	target *device.Instance, handler TaskHandler) error {
	request := &worker.RegisterRequest{Host: host, Target: target}
	stream, err := m.client.Register(ctx, request)
	if err != nil {
		return err
	}
	return event.Feed(ctx, event.AsHandler(ctx, handler), grpcutil.ToProducer(stream))
}

// Do implements Manager.Do
// It forwards the call through grpc to the remote implementation.
func (m *remote) Do(ctx context.Context, device string, input *Input) (string, error) {
	response, err := m.client.Do(ctx, &DoRequest{Device: device, Input: input})
	return response.Id, err
}

// Update implements Manager.Update
// It forwards the call through grpc to the remote implementation.
func (m *remote) Update(ctx context.Context, action string, status job.Status, output *Output) error {
	request := &UpdateRequest{Action: action, Status: status, Output: output}
	_, err := m.client.Update(ctx, request)
	return err
}
