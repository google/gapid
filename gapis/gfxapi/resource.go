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

package gfxapi

import (
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/service/path"
)

// ResourceMap is a map from Resource to its id in a database.
type ResourceMap map[Resource]id.ID

// Resource represents an asset in a capture.
type Resource interface {
	// ResourceName returns the UI name for the resource.
	ResourceName() string

	// ResourceType returns the type of this resource.
	ResourceType() ResourceType

	// ResourceData returns the resource data given the current state.
	ResourceData(ctx log.Context, s *State, resources ResourceMap) (interface{}, error)

	// SetResourceData sets resource data in a new capture.
	SetResourceData(ctx log.Context, at *path.Command, data interface{}, resources ResourceMap, edits ReplaceCallback) error
}

// ResourceMeta represents resource with a state information obtained during building.
type ResourceMeta struct {
	Resource Resource    // Resolved resource.
	IDMap    ResourceMap // Map for resolved resources to ids.
}

// ResourceAtom describes atoms which should be replaced with resource data.
// TODO: Remove it and use atom.Atom itself (which is impossible for now because of a dependency cycle).
type ResourceAtom interface {
	// Replace clones an atom and sets new data.
	Replace(ctx log.Context, data interface{}) ResourceAtom
}

// ReplaceCallback is called from SetResourceData to propagate changes to current atom stream.
type ReplaceCallback func(where uint64, with ResourceAtom)
