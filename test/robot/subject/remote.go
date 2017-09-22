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

package subject

import (
	"context"

	"github.com/google/gapid/core/event"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/test/robot/search"
	"google.golang.org/grpc"
)

type remote struct {
	client ServiceClient
}

// NewRemote returns a Subjects that talks to a remote grpc Subject service.
func NewRemote(ctx context.Context, conn *grpc.ClientConn) Subjects {
	return &remote{
		client: NewServiceClient(conn),
	}
}

// Search implements Subjects.Search
// It forwards the call through grpc to the remote implementation.
func (m *remote) Search(ctx context.Context, query *search.Query, handler Handler) error {
	stream, err := m.client.Search(ctx, query)
	if err != nil {
		return err
	}
	return event.Feed(ctx, event.AsHandler(ctx, handler), grpcutil.ToProducer(stream))
}

// Add implements Subjects.Add
// It forwards the call through grpc to the remote implementation.
func (m *remote) Add(ctx context.Context, id string, hints *Hints) (*Subject, bool, error) {
	request := &AddRequest{Id: id, Hints: hints}
	response, err := m.client.Add(ctx, request)
	if err != nil {
		return nil, false, err
	}
	return response.Subject, response.Created, nil
}

func (m *remote) Update(ctx context.Context, subj *Subject) (*Subject, error) {
	request := &UpdateRequest{Subject: subj}
	response, err := m.client.Update(ctx, request)
	if err != nil {
		return nil, err
	}
	return response.Subject, nil
}
