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

package build

import (
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/data/search"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type server struct {
	store Store
}

// Serve wraps a store in a grpc server.
func Serve(ctx log.Context, grpcServer *grpc.Server, store Store) error {
	RegisterServiceServer(grpcServer, &server{store: store})
	return nil
}

// SearchArtifacts implements ServiceServer.SearchArtifacts
// It delegates the call to the provided Store implementation.
func (s *server) SearchArtifacts(query *search.Query, stream Service_SearchArtifactsServer) error {
	ctx := log.Wrap(stream.Context())
	return s.store.SearchArtifacts(ctx, query, func(ctx log.Context, e *Artifact) error { return stream.Send(e) })
}

// SearchPackages implements ServiceServer.SearchPackages
// It delegates the call to the provided Store implementation.
func (s *server) SearchPackages(query *search.Query, stream Service_SearchPackagesServer) error {
	ctx := log.Wrap(stream.Context())
	return s.store.SearchPackages(ctx, query, func(ctx log.Context, e *Package) error { return stream.Send(e) })
}

// SearchTracks implements ServiceServer.SearchTrackst
// It delegates the call to the provided Store implementation.
func (s *server) SearchTracks(query *search.Query, stream Service_SearchTracksServer) error {
	ctx := log.Wrap(stream.Context())
	return s.store.SearchTracks(ctx, query, func(ctx log.Context, e *Track) error { return stream.Send(e) })
}

// Add implements ServiceServer.Add
// It delegates the call to the provided Store implementation.
func (s *server) Add(outer context.Context, request *AddRequest) (*AddResponse, error) {
	ctx := log.Wrap(outer)
	id, merged, err := s.store.Add(ctx, request.Id, request.Information)
	if err != nil {
		return nil, err
	}
	return &AddResponse{
		Id:     id,
		Merged: merged,
	}, nil
}

// UpdateTrack implements ServiceServer.UpdateTrack
// It delegates the call to the provided Store implementation.
func (s *server) UpdateTrack(outer context.Context, request *UpdateTrackRequest) (*UpdateTrackResponse, error) {
	ctx := log.Wrap(outer)
	track, err := s.store.UpdateTrack(ctx, request.Track)
	if err != nil {
		return nil, err
	}
	return &UpdateTrackResponse{Track: track}, nil
}
