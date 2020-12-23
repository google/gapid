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

package api

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/protoutil"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/gapis/service/path"
)

// ResourceMap is a map from Resource handles to Resource IDs in the database.
// Note this map is not time-globally valid. It is only valid at a specific
// point in a trace, since handles may be re-used.
type ResourceMap map[string]id.ID

// Resource represents an asset in a capture.
type Resource interface {
	// ResourceHandle returns the UI identity for the resource.
	// For GL this is the GLuint object name, for Vulkan the pointer.
	ResourceHandle() string

	// ResourceLabel returns the UI name for the resource.
	ResourceLabel() string

	// Order returns an integer used to sort the resources for presentation.
	Order() uint64

	// ResourceType returns the type of this resource.
	ResourceType(ctx context.Context) ResourceType

	// ResourceData returns the resource data given the current state.
	ResourceData(ctx context.Context, s *GlobalState, cmd *path.Command, r *path.ResolveConfig) (*ResourceData, error)

	// SetResourceData sets resource data in a new capture.
	SetResourceData(
		ctx context.Context,
		at *path.Command,
		data *ResourceData,
		resources ResourceMap,
		edits ReplaceCallback,
		mutate MutateInitialState,
		r *path.ResolveConfig) error
}

// ReplaceCallback is called from SetResourceData to propagate changes to current command stream.
type ReplaceCallback func(where uint64, with interface{})

// MutateInitialState is called from SetResourceData to get a mutable instance of the initial state.
type MutateInitialState func(API API) State

// Interface compliance check
var _ = image.Convertable((*ResourceData)(nil))
var _ = image.Thumbnailer((*ResourceData)(nil))

// ConvertTo returns this Texture2D with each mip-level converted to the requested format.
func (r *ResourceData) ConvertTo(ctx context.Context, f *image.Format) (interface{}, error) {
	data := protoutil.OneOf(r.Data)
	if c, ok := data.(image.Convertable); ok {
		data, err := c.ConvertTo(ctx, f)
		if err != nil {
			return nil, err
		}
		if data == nil {
			return nil, nil
		}
		return NewResourceData(data), nil
	}
	return nil, nil
}

// Thumbnail returns the image that most closely matches the desired size.
func (r *ResourceData) Thumbnail(ctx context.Context, w, h, d uint32) (*image.Info, error) {
	data := protoutil.OneOf(r.Data)
	if t, ok := data.(image.Thumbnailer); ok {
		return t.Thumbnail(ctx, w, h, d)
	}
	return nil, nil
}

// NewResourceData returns a new *ResourceData with the specified data.
func NewResourceData(data interface{}) *ResourceData {
	switch data := data.(type) {
	case *Texture:
		return &ResourceData{Data: &ResourceData_Texture{data}}
	case *Shader:
		return &ResourceData{Data: &ResourceData_Shader{data}}
	case *Program:
		return &ResourceData{Data: &ResourceData_Program{data}}
	case *Pipeline:
		return &ResourceData{Data: &ResourceData_Pipeline{data}}
	default:
		panic(fmt.Errorf("%T is not a ResourceData type", data))
	}
}

// NewMultiResourceData returns a new *MultiResourceData with the specified resources.
func NewMultiResourceData(resources []*ResourceData) *MultiResourceData {
	return &MultiResourceData{Resources: resources}
}
