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
	"context"

	"github.com/google/gapid/test/robot/search"

	"google.golang.org/grpc"

	xctx "golang.org/x/net/context"
)

type server struct {
	store Store
}

// Serve wraps a store in a grpc server.
func Serve(ctx context.Context, grpcServer *grpc.Server, store Store) error {
	RegisterServiceServer(grpcServer, &server{store: store})
	return nil
}

// SearchArtifacts implements ServiceServer.SearchArtifacts
// It delegates the call to the provided Store implementation.
func (s *server) SearchArtifacts(query *search.Query, stream Service_SearchArtifactsServer) error {
	ctx := stream.Context()
	return s.store.SearchArtifacts(ctx, query, func(ctx context.Context, e *Artifact) error { return stream.Send(e) })
}

// SearchPackages implements ServiceServer.SearchPackages
// It delegates the call to the provided Store implementation.
func (s *server) SearchPackages(query *search.Query, stream Service_SearchPackagesServer) error {
	ctx := stream.Context()
	return s.store.SearchPackages(ctx, query, func(ctx context.Context, e *Package) error { return stream.Send(e) })
}

// SearchTracks implements ServiceServer.SearchTrackst
// It delegates the call to the provided Store implementation.
func (s *server) SearchTracks(query *search.Query, stream Service_SearchTracksServer) error {
	ctx := stream.Context()
	return s.store.SearchTracks(ctx, query, func(ctx context.Context, e *Track) error { return stream.Send(e) })
}

// Add implements ServiceServer.Add
// It delegates the call to the provided Store implementation.
func (s *server) Add(outer xctx.Context, request *AddRequest) (*AddResponse, error) {
	ctx := outer
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
func (s *server) UpdateTrack(outer xctx.Context, request *UpdateTrackRequest) (*UpdateTrackResponse, error) {
	ctx := outer
	track, err := s.store.UpdateTrack(ctx, request.Track)
	if err != nil {
		return nil, err
	}
	return &UpdateTrackResponse{Track: track}, nil
}

// UpdatePackage implements ServiceServer.UpdatePackage
// It delegates the call to the provided Store implementation.
func (s *server) UpdatePackage(outer xctx.Context, request *UpdatePackageRequest) (*UpdatePackageResponse, error) {
	ctx := outer
	pkg, err := s.store.UpdatePackage(ctx, request.Package)
	if err != nil {
		return nil, err
	}
	return &UpdatePackageResponse{Package: pkg}, nil
}
