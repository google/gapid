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

	"github.com/google/gapid/test/robot/search"

	"google.golang.org/grpc"

	xctx "golang.org/x/net/context"
)

type server struct {
	subjects Subjects
}

// Serve wraps a subject service in a grpc server.
func Serve(ctx context.Context, grpcServer *grpc.Server, subjects Subjects) error {
	RegisterServiceServer(grpcServer, &server{subjects: subjects})
	return nil
}

// Add implements ServiceServer.Add
// It delegates the call to the provided Subjects implementation.
func (s *server) Add(ctx xctx.Context, request *AddRequest) (*AddResponse, error) {
	subj, created, err := s.subjects.Add(ctx, request.Id, request.Hints)
	if err != nil {
		return nil, err
	}
	return &AddResponse{Subject: subj, Created: created}, nil
}

// Search implements ServiceServer.Search
// It delegates the call to the provided Subjects implementation.
func (s *server) Search(query *search.Query, stream Service_SearchServer) error {
	ctx := stream.Context()
	return s.subjects.Search(ctx, query, func(ctx context.Context, e *Subject) error { return stream.Send(e) })
}

// Update implements ServiceServer.Update
// It delegates the call to the provided Subjects implementation.
func (s *server) Update(ctx xctx.Context, request *UpdateRequest) (*UpdateResponse, error) {
	subj, err := s.subjects.Update(ctx, request.Subject)
	if err != nil {
		return nil, err
	}
	return &UpdateResponse{Subject: subj}, nil
}
