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

	"github.com/google/gapid/core/event"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/test/robot/search"
	"google.golang.org/grpc"
)

type remote struct {
	client ServiceClient
}

// NewRemote returns a Store that talks to a remote grpc build service.
func NewRemote(ctx context.Context, conn *grpc.ClientConn) Store {
	return &remote{client: NewServiceClient(conn)}
}

// SearchArtifacts implements Store.SearchArtifacts
// It forwards the call through grpc to the remote implementation.
func (m *remote) SearchArtifacts(ctx context.Context, query *search.Query, handler ArtifactHandler) error {
	stream, err := m.client.SearchArtifacts(ctx, query)
	if err != nil {
		return err
	}
	return event.Feed(ctx, event.AsHandler(ctx, handler), grpcutil.ToProducer(stream))
}

// SearchPackages implements Store.SearchPackages
// It forwards the call through grpc to the remote implementation.
func (m *remote) SearchPackages(ctx context.Context, query *search.Query, handler PackageHandler) error {
	stream, err := m.client.SearchPackages(ctx, query)
	if err != nil {
		return err
	}
	return event.Feed(ctx, event.AsHandler(ctx, handler), grpcutil.ToProducer(stream))
}

// SearchTracks implements Store.SearchTracks
// It forwards the call through grpc to the remote implementation.
func (m *remote) SearchTracks(ctx context.Context, query *search.Query, handler TrackHandler) error {
	stream, err := m.client.SearchTracks(ctx, query)
	if err != nil {
		return err
	}
	return event.Feed(ctx, event.AsHandler(ctx, handler), grpcutil.ToProducer(stream))
}

// Add implements Store.Add
// It forwards the call through grpc to the remote implementation.
func (m *remote) Add(ctx context.Context, id string, info *Information) (string, bool, error) {
	request := &AddRequest{Id: id, Information: info}
	response, err := m.client.Add(ctx, request)
	if err != nil {
		return "", false, err
	}
	return response.Id, response.Merged, nil
}

// UpdateTrack implements store.UpdateTrack
// It forwards the call through grpc to the remote implementation.
func (m *remote) UpdateTrack(ctx context.Context, entry *Track) (*Track, error) {
	request := &UpdateTrackRequest{Track: entry}
	response, err := m.client.UpdateTrack(ctx, request)
	if err != nil {
		return nil, err
	}
	return response.Track, nil
}

// UpdatePackage implements store.UpdatePackage
// It forwards the call through grpc to the remote implementation.
func (m *remote) UpdatePackage(ctx context.Context, entry *Package) (*Package, error) {
	request := &UpdatePackageRequest{Package: entry}
	response, err := m.client.UpdatePackage(ctx, request)
	if err != nil {
		return nil, err
	}
	return response.Package, nil
}
