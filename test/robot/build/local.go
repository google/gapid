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

	"github.com/google/gapid/test/robot/record"
	"github.com/google/gapid/test/robot/search"
	"github.com/google/gapid/test/robot/stash"
)

type local struct {
	store *stash.Client

	artifacts artifacts
	packages  packages
	tracks    tracks
}

// NewLocal builds a new build artifact manager that persists its data in the
// supplied library, and the artifacts in the supplied stash.
func NewLocal(ctx context.Context, store *stash.Client, library record.Library) (Store, error) {
	s := &local{store: store}
	if err := s.artifacts.init(ctx, store, library); err != nil {
		return nil, err
	}
	if err := s.packages.init(ctx, library); err != nil {
		return nil, err
	}
	if err := s.tracks.init(ctx, library); err != nil {
		return nil, err
	}
	return s, nil
}

// SearchArtifacts implements Store.SearchArtifacts
// It searches the set of persisted artifacts, and supports monitoring of artifacts as they arrive.
func (s *local) SearchArtifacts(ctx context.Context, query *search.Query, handler ArtifactHandler) error {
	return s.artifacts.search(ctx, query, handler)
}

// SearchPackages implements Store.SearchPackages
// It searches the set of persisted packages, and supports monitoring of packages as they arrive.
func (s *local) SearchPackages(ctx context.Context, query *search.Query, handler PackageHandler) error {
	return s.packages.search(ctx, query, handler)
}

// SearchTracks implements Store.SearchTracks
// It searches the set of persisted tracks, and supports monitoring of tracks as they arrive.
func (s *local) SearchTracks(ctx context.Context, query *search.Query, handler TrackHandler) error {
	return s.tracks.search(ctx, query, handler)
}

// Add implements Store.Add
// It adds the package to the persisten store, and attempts to add it into the track it should be part of.
func (s *local) Add(ctx context.Context, id string, info *Information) (string, bool, error) {
	a, err := s.artifacts.get(ctx, id, info.Builder.Configuration.ABIs[0])
	if err != nil {
		return "", false, err
	}
	pkg, merged, err := s.packages.addArtifact(ctx, a, info)
	if err != nil {
		return "", false, err
	}
	if merged {
		return pkg.Id, true, nil
	}
	parent, err := s.tracks.addPackage(ctx, pkg)
	if err != nil {
		return "", false, err
	}
	if parent != "" {
		if _, err := s.packages.update(ctx, &Package{Id: pkg.Id, Parent: parent}); err != nil {
			return "", false, err
		}
	}
	return pkg.Id, false, nil
}

// UpdateTrack implements store.UpdateTrack
// if the track identified by the track id exists, it modifies the track head pointer, name
// and description, otherwise it creates a new track.
func (s *local) UpdateTrack(ctx context.Context, entry *Track) (*Track, error) {
	track, _, err := s.tracks.createOrUpdate(ctx, entry)
	return track, err
}

// UpdatePackage implements store.UpdatePackage
// if the package identified by the package id exists, it modifies the package parent pointer,
// and description.
func (s *local) UpdatePackage(ctx context.Context, entry *Package) (*Package, error) {
	return s.packages.update(ctx, entry)
}
