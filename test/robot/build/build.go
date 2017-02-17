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
)

// ArtifactHandler is a function used to consume a stream of Artifacts.
type ArtifactHandler func(log.Context, *Artifact) error

// PackageHandler is a function used to consume a stream of Packages.
type PackageHandler func(log.Context, *Package) error

// TrackHandler is a function used to consume a stream of Tracks.
type TrackHandler func(log.Context, *Track) error

// Store is the abstract interface to the storage of build artifacts.
type Store interface {
	// SearchArtifacts delivers matching build artifacts to the supplied handler.
	SearchArtifacts(ctx log.Context, query *search.Query, handler ArtifactHandler) error

	// SearchPackages delivers matching build packages to the supplied handler.
	SearchPackages(ctx log.Context, query *search.Query, handler PackageHandler) error

	// SearchTracks delivers matching build tracks to the supplied handler.
	SearchTracks(ctx log.Context, query *search.Query, handler TrackHandler) error

	// Add adds a new build artifact to the store.
	Add(ctx log.Context, id string, info *Information) (string, bool, error)

	// UpdateTrack updates track information.
	UpdateTrack(ctx log.Context, track *Track) (*Track, error)
}

const (
	UnknownType = Type_UnknownType
	BuildBot    = Type_BuildBot
	User        = Type_User
	Local       = Type_Local
)
