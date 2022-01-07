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
	"sort"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapil/constset"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/service/path"
)

// API is the common interface to a graphics programming api.
type API interface {
	// Name returns the official name of the api.
	Name() string

	// Index returns the API index.
	Index() uint8

	// ID returns the unique API identifier.
	ID() ID

	// ConstantSets returns the constant set pack for the API.
	ConstantSets() *constset.Pack

	// GetFramebufferAttachmentInfo returns the width, height, and format of the
	// specified framebuffer attachment.
	// It also returns an API specific index that maps the given attachment into
	// an API specific representation.
	GetFramebufferAttachmentInfos(
		ctx context.Context,
		state *GlobalState) (info []FramebufferAttachmentInfo, err error)

	// CreateCmd constructs and returns a new command with the specified name.
	CreateCmd(name string) Cmd

	// RebuildState returns a set of commands which, if executed on a new clean
	// state, will reproduce the API's state in s.
	// The segments of memory that were used to create these commands are
	// returned in the rangeList.
	RebuildState(ctx context.Context, s *GlobalState) ([]Cmd, interval.U64RangeList)

	// GetFramegraph returns the framegraph of the capture.
	GetFramegraph(ctx context.Context, p *path.Capture) (*Framegraph, error)

	// ProfileStaticAnalysis computes the static analysis profiling data of a capture.
	ProfileStaticAnalysis(ctx context.Context, p *path.Capture) (*StaticAnalysisProfileData, error)
}

// FramebufferAttachmentInfo describes a framebuffer at a given point in the trace
type FramebufferAttachmentInfo struct {
	// Width in texels of the framebuffer
	Width uint32
	// Height in texels of the framebuffer
	Height uint32
	// Framebuffer index
	Index uint32
	// Format of the image
	Format *image.Format
	// CanResize is true if this can be efficiently resized during replay.
	CanResize bool
	// Attachment type (Color, Depth, Input, Resolve)
	Type FramebufferAttachmentType
	// Error message when calling state.getFramebufferAttachmentInfo
	Err error
}

// StaticAnalysisProfileData is the result of the profiling static analysis.
type StaticAnalysisProfileData struct {
	CounterSpecs []StaticAnalysisCounter
	CounterData  []StaticAnalysisCounterSamples
}

// StaticAnalysisCounter represents the metadata of a counter produced via the static analysis.
type StaticAnalysisCounter struct {
	ID          uint32
	Name        string
	Description string
	Unit        string // this should match the unit from the Perfetto data.
}

// StaticAnalysisCounterSample is a single sample of a counter.
type StaticAnalysisCounterSample struct {
	Counter uint32
	Value   float64
}

// StaticAnalysisCounterSamples contains all the counter samples at a command index (draw call).
type StaticAnalysisCounterSamples struct {
	Index   SubCmdIdx
	Samples []StaticAnalysisCounterSample
}

// ID is an API identifier
type ID id.ID

// IsValid returns true if the id is not the default zero value.
func (i ID) IsValid() bool  { return id.ID(i).IsValid() }
func (i ID) String() string { return id.ID(i).String() }

// CoreId return id in gapid/core/data/id.ID type instead of api.ID type.
func (i ID) CoreId() id.ID { return id.ID(i) }

// APIObject is the interface implemented by types that belong to an API.
type APIObject interface {
	// API returns the API identifier that this type belongs to.
	API() API
}

var apis = map[ID]API{}
var indices = map[uint8]API{}

// Register adds an api to the understood set.
// It is illegal to register the same name twice.
func Register(api API) {
	id := api.ID()
	if existing, present := apis[id]; present {
		panic(fmt.Errorf("API %s registered more than once. First: %T, Second: %T", id, existing, api))
	}
	apis[id] = api

	index := api.Index()
	if existing, present := indices[index]; present {
		panic(fmt.Errorf("API %s used an occupied index %d. First: %T, Second: %T", id, index, existing, api))
	}
	indices[index] = api
}

// Find looks up a graphics API by identifier.
// If the id has not been registered, it returns nil.
func Find(id ID) API {
	return apis[id]
}

// CloneContext is used to keep track of references when cloning API objects.
type CloneContext map[interface{}]interface{}

// All returns all the registered APIs.
func All() []API {
	out := make([]API, 0, len(apis))
	for _, api := range apis {
		out = append(out, api)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Index() < out[j].Index() })
	return out
}

type Slice interface {
	Reset(uint64, uint64, uint64, *GlobalState, memory.PoolID)
}
