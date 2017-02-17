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

package resolve

import (
	"fmt"
	"reflect"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/fault/cause"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Hierarchies resolves the list of hierarchies.
func Hierarchies(ctx log.Context, h *path.Hierarchies) ([]*service.Hierarchy, error) {
	obj, err := database.Build(ctx, &HierarchiesResolvable{h.Capture})
	if err != nil {
		return nil, err
	}
	return obj.([]*service.Hierarchy), nil
}

// Resolve implements the database.Resolver interface.
func (r *HierarchiesResolvable) Resolve(ctx log.Context) (interface{}, error) {
	ctx = capture.Put(ctx, r.Capture)

	c, err := capture.Resolve(ctx)
	if err != nil {
		return nil, err
	}

	list, err := c.Atoms(ctx)
	if err != nil {
		return nil, err
	}

	atoms := list.Atoms

	ohb := newOverviewHierarchyBuilder(atoms)
	contexts := map[gfxapi.ContextID]*contextHierarchyBuilder{}

	var currentAtomIndex int
	var currentAtom atom.Atom
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = fault.Const(fmt.Sprint(r))
			}
			panic(cause.Wrap(ctx, err).With("atomID", currentAtomIndex).With("atom", reflect.TypeOf(currentAtom)))
		}
	}()

	// Build the overview hierarchy from the context spans and each of the
	// per-context hierarchies from user markers.
	s := c.NewState()
	for i, a := range atoms {
		currentAtomIndex, currentAtom = i, a
		// Use the context before mutation and ignore core API.
		// This will place "switching" atoms at the end of group.
		if api := a.API(); api != nil && api.Index() != 0 /* core */ {
			ohb.add(api.Context(s), uint64(i))
		}

		a.Mutate(ctx, s, nil)

		if api := a.API(); api != nil {
			if context := api.Context(s); context != nil {
				id := context.ID()
				chb, ok := contexts[id]
				if !ok {
					chb = newContextHierarchyBuilder(context, atoms, uint64(i))
					contexts[id] = chb
				}
				chb.addUserMarkers(ctx, a, uint64(i), s)
			}
		}
	}

	// Add to each per-context hierarchy groups for draw calls and end-of-frames.
	s = c.NewState()
	for i, a := range atoms {
		a.Mutate(ctx, s, nil)
		if api := a.API(); api != nil {
			if context := api.Context(s); context != nil {
				contexts[context.ID()].addFrameAndDraws(ctx, a, uint64(i), s)
			}
		}
	}

	out := make([]*service.Hierarchy, 0, len(contexts)+ /*overview*/ 1)

	// Add the overview hierarchy
	ohb.finalize(uint64(len(atoms)))
	overview := service.NewHierarchy("Overview", id.ID{}, ohb.root)
	out = append(out, overview)

	// Add each of the context hierarchies
	for _, chb := range contexts {
		chb.finalize(atoms)
		hierarchy := service.NewHierarchy(chb.name, chb.context, chb.root)
		out = append(out, hierarchy)
	}

	return out, nil
}

// overviewHierarchyBuilder constructs an 'overview' hierarchy.
// This hierarchy lists each of the contexts in use as a 1-level deep tree.
type overviewHierarchyBuilder struct {
	start   uint64
	context gfxapi.Context
	root    atom.Group
}

func newOverviewHierarchyBuilder(atoms []atom.Atom) *overviewHierarchyBuilder {
	return &overviewHierarchyBuilder{
		root: atom.Group{
			Range: atom.Range{End: uint64(len(atoms))},
		},
	}
}

func (h *overviewHierarchyBuilder) add(context gfxapi.Context, i uint64) {
	if context != h.context {
		h.finalize(i)
	}
	h.context = context
}

func (h *overviewHierarchyBuilder) finalize(end uint64) {
	if end > h.start {
		if h.context != nil {
			h.root.SubGroups.Add(h.start, end, h.context.Name())
		} else {
			h.root.SubGroups.Add(h.start, end, "No context")
		}
	}
	h.start = end
}

type userMarker struct {
	start uint64
	name  string
}

// contextHierarchyBuilder constructs an hierarchy for a single context.
// This hierarchy uses user-markers, frames and draw calls to build a deep
// hierarchy.
type contextHierarchyBuilder struct {
	userMarkers     []userMarker
	userMarkerCount int
	frameStart      uint64
	frameCount      int
	drawStart       uint64
	drawCount       int
	name            string
	context         id.ID
	root            atom.Group
}

const notStarted = 0xffffffffffffffff

func newContextHierarchyBuilder(context gfxapi.Context, atoms []atom.Atom, contextStart uint64) *contextHierarchyBuilder {
	r := &contextHierarchyBuilder{
		frameStart: notStarted,
		drawStart:  notStarted,
		name:       context.Name(),
		context:    id.ID(context.ID()),
		root: atom.Group{
			Range: atom.Range{End: uint64(len(atoms))},
		},
	}
	r.root.SubGroups.Add(0, contextStart, "Context Setup")
	return r
}

func (h *contextHierarchyBuilder) addUserMarkers(ctx log.Context, a atom.Atom, i uint64, s *gfxapi.State) {
	if a.AtomFlags().IsPushUserMarker() {
		marker := userMarker{start: i}
		if labeled, ok := a.(atom.Labeled); ok {
			marker.name = labeled.Label(ctx, s)
		} else {
			marker.name = fmt.Sprintf("User marker %d", h.userMarkerCount)
		}
		h.userMarkerCount++
		h.userMarkers = append(h.userMarkers, marker)
	}
	if a.AtomFlags().IsPopUserMarker() {
		if c := len(h.userMarkers); c > 0 {
			marker := h.userMarkers[c-1]
			h.userMarkers = h.userMarkers[:c-1]
			h.root.SubGroups.Add(marker.start, i+1, marker.name)
		}
	}
}

func (h *contextHierarchyBuilder) addFrameAndDraws(ctx log.Context, a atom.Atom, i uint64, s *gfxapi.State) {
	if h.frameStart == notStarted {
		h.frameStart = i
	}
	if h.drawStart == notStarted {
		h.drawStart = i
	}
	endIndex := uint64(i) + 1 // Increment by one, since atom.Range's end is non-inclusive.
	if a.AtomFlags().IsEndOfFrame() {
		h.root.SubGroups.Add(h.frameStart, endIndex, fmt.Sprintf("Frame %d", h.frameCount+1))
		h.frameStart = notStarted
		h.frameCount++
		h.drawStart = notStarted
		h.drawCount = 0 // Reset the draw index, it is relative to the new frame index.
	}
	if a.AtomFlags().IsDrawCall() {
		h.root.SubGroups.Add(h.drawStart, endIndex, fmt.Sprintf("Draw %d", h.drawCount))
		h.drawStart = endIndex
		h.drawCount++
	}
}

func (h *contextHierarchyBuilder) finalize(atoms []atom.Atom) {
	if h.frameStart != notStarted && h.frameCount > 0 {
		h.root.SubGroups.Add(h.frameStart, uint64(len(atoms)), "Incomplete Frame")
	}
}
