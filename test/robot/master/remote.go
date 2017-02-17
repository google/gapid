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
	"github.com/google/gapid/core/event"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/data/search"
	"google.golang.org/grpc"
)

type remote struct {
	client ServiceClient
}

// NewRemoteMaster returns a Master that talks to a remote grpc Master service.
func NewRemoteMaster(ctx log.Context, conn *grpc.ClientConn) Master {
	return &remote{
		client: NewServiceClient(conn),
	}
}

// Search implements Master.Search
// It forwards the call through grpc to the remote implementation.
func (m *remote) Search(ctx log.Context, query *search.Query, handler SatelliteHandler) error {
	stream, err := m.client.Search(ctx.Unwrap(), query)
	if err != nil {
		return err
	}
	return event.Feed(ctx, event.AsHandler(ctx, handler), grpcutil.ToProducer(stream))
}

// Orbit implements Master.Orbit
// It forwards the call through grpc to the remote implementation.
func (m *remote) Orbit(ctx log.Context, services ServiceList, handler CommandHandler) error {
	request := &OrbitRequest{Services: &services}
	stream, err := m.client.Orbit(ctx.Unwrap(), request)
	if err != nil {
		return err
	}
	return event.Feed(ctx, event.AsHandler(ctx, handler), grpcutil.ToProducer(stream))
}

// Shutdown implements Master.Shutdown
// It forwards the call through grpc to the remote implementation.
func (m *remote) Shutdown(ctx log.Context, request *ShutdownRequest) (*ShutdownResponse, error) {
	return m.client.Shutdown(ctx.Unwrap(), request)
}
