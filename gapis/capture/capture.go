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

package capture

import (
	"context"
	"io"
	"sync"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/pack"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/atom_pb"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/value"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/pkg/errors"
)

// The list of captures currently imported.
// TODO: This needs to be moved to persistent storage.
var (
	capturesLock sync.RWMutex
	captures     = []id.ID{}
)

// NewState returns a new, default-initialized State object built for the
// capture held by the context.
func NewState(ctx context.Context) *gfxapi.State {
	c, err := Resolve(ctx)
	if err != nil {
		panic(err)
	}
	return c.NewState()
}

// NewState returns a new, default-initialized State object built for the
// capture.
func (c *Capture) NewState() *gfxapi.State {
	freeList := memory.InvertMemoryRanges(fromMemoryRanges(c.Observed))
	interval.Remove(&freeList, interval.U64Span{Start: 0, End: value.FirstValidAddress})
	return gfxapi.NewStateWithAllocator(
		memory.NewBasicAllocator(freeList),
		c.Header.Abi.MemoryLayout,
	)
}

// Atoms resolves and returns the atom list for the capture.
func (c *Capture) Atoms(ctx context.Context) (*atom.List, error) {
	obj, err := database.Resolve(ctx, c.Commands.ID())
	if err != nil {
		return nil, err
	}
	return obj.(*atom.List), nil
}

// Service returns the service.Capture description for this capture.
func (c *Capture) Service(ctx context.Context, p *path.Capture) *service.Capture {
	apis := make([]*path.API, len(c.Apis))
	for i, a := range c.Apis {
		apis[i] = &path.API{Id: path.NewID(a.ID())}
	}
	observations := make([]*service.MemoryRange, len(c.Observed))
	for i, o := range c.Observed {
		observations[i] = &service.MemoryRange{Base: o.Base, Size: o.Size}
	}
	return &service.Capture{
		Name:         c.Name,
		Device:       c.Header.Device,
		Commands:     p.Commands(),
		Apis:         apis,
		Observations: observations,
	}
}

// AtomsImportHandler is the interface optionally implements by APIs that want
// to process the atom stream on import.
type AtomsImportHandler interface {
	TransformAtomStream(context.Context, []atom.Atom) ([]atom.Atom, error)
}

// Captures returns all the captures stored by the database by identifier.
func Captures() []*path.Capture {
	capturesLock.RLock()
	defer capturesLock.RUnlock()
	out := make([]*path.Capture, len(captures))
	for i, c := range captures {
		out[i] = &path.Capture{Id: path.NewID(c)}
	}
	return out
}

// ResolveFromID resolves a single capture with the ID id.
func ResolveFromID(ctx context.Context, id id.ID) (*Capture, error) {
	obj, err := database.Resolve(ctx, id)
	if err != nil {
		return nil, err
	}
	return obj.(*Capture), nil
}

// ResolveFromPath resolves a single capture with the path p.
func ResolveFromPath(ctx context.Context, p *path.Capture) (*Capture, error) {
	return ResolveFromID(ctx, p.Id.ID())
}

// Import reads capture data from an io.Reader, imports into the given
// database and returns the new capture identifier.
func Import(ctx context.Context, name string, in io.ReadSeeker) (*path.Capture, error) {
	reader, err := pack.NewReader(in)
	if err != nil {
		return nil, err
	}

	list := atom.NewList()
	convert := atom.FromConverter(func(a atom.Atom) {
		list.Atoms = append(list.Atoms, a)
	})
	var header *Header
	for {
		msg, err := reader.Unmarshal()
		if errors.Cause(err) == io.EOF {
			break
		}
		if err != nil {
			return nil, log.Err(ctx, err, "Failed to unmarshal")
		}
		if h, ok := msg.(*Header); ok {
			header = h
			continue
		}
		convert(ctx, msg)
	}

	if header == nil {
		return nil, log.Err(ctx, nil, "Capture was missing header chunk")
	}

	// must invoke the converter with nil to flush the last atom
	if err := convert(ctx, nil); err != nil {
		return nil, err
	}

	if len(list.Atoms) == 0 {
		return nil, nil
	}
	return ImportAtomList(ctx, name, list, header)
}

// Export encodes the given capture and associated resources
// and writes it to the supplied io.Writer in the pack file format,
// producing output suitable for use with Import or opening in the trace editor.
func Export(ctx context.Context, p *path.Capture, w io.Writer) error {
	c, err := ResolveFromPath(ctx, p)
	if err != nil {
		return err
	}
	return c.Export(ctx, w)
}

// ImportAtomList builds a new capture containing a, stores it into d and
// returns the new capture path.
func ImportAtomList(ctx context.Context, name string, a *atom.List, h *Header) (*path.Capture, error) {
	a, observed, err := process(ctx, a)
	if err != nil {
		return nil, err
	}

	streamID, err := database.Store(ctx, a)
	if err != nil {
		return nil, err
	}

	// Gather all the APIs used by the capture
	apis := map[gfxapi.ID]gfxapi.API{}
	apiIDs := []*ID{}
	for _, a := range a.Atoms {
		if api := a.API(); api != nil {
			apiID := api.ID()
			if _, found := apis[apiID]; !found {
				apis[apiID] = api
				apiIDs = append(apiIDs, NewID(id.ID(apiID)))
			}
		}
	}

	for _, api := range apis {
		if aih, ok := api.(AtomsImportHandler); ok {
			a.Atoms, err = aih.TransformAtomStream(ctx, a.Atoms)
			if err != nil {
				return nil, err
			}
		}
	}

	capture := &Capture{
		Name:     name,
		Apis:     apiIDs,
		Commands: NewID(streamID),
		Header:   h,
		Observed: observed,
	}

	captureID, err := database.Store(ctx, capture)
	if err != nil {
		return nil, err
	}

	capturesLock.Lock()
	captures = append(captures, captureID)
	capturesLock.Unlock()

	return &path.Capture{Id: path.NewID(captureID)}, nil
}

type atomWriter func(ctx context.Context, a atom.Atom) error

// Export encodes the given capture and associated resources
// and writes it to the supplied io.Writer in the .gfxtrace format,
// producing output suitable for use with Import or opening in the trace editor.
func (c *Capture) Export(ctx context.Context, w io.Writer) error {
	atoms, err := c.Atoms(ctx)
	if err != nil {
		return err
	}

	write, err := pack.NewWriter(w)
	if err != nil {
		return err
	}

	writeMsg := func(ctx context.Context, a atom_pb.Atom) error { return write.Marshal(a) }

	if err := writeMsg(ctx, c.Header); err != nil {
		return err
	}

	writeAtom := func(ctx context.Context, a atom.Atom) error {
		c, ok := a.(atom.Convertible)
		if !ok {
			return atom.ErrNotConvertible
		}
		return c.Convert(ctx, writeMsg)
	}

	// IDs seen, so we can avoid encoding the same resource data multiple times.
	seen := map[id.ID]bool{}

	encodeObservation := func(o atom.Observation) error {
		if seen[o.ID] {
			return nil
		}
		data, err := database.Resolve(ctx, o.ID)
		if err != nil {
			return err
		}
		err = writeAtom(ctx, &atom.Resource{ID: o.ID, Data: data.([]uint8)})
		seen[o.ID] = true
		return err
	}

	for _, a := range atoms.Atoms {
		if observations := a.Extras().Observations(); observations != nil {
			for _, r := range observations.Reads {
				if err := encodeObservation(r); err != nil {
					return err
				}
			}
			for _, w := range observations.Writes {
				if err := encodeObservation(w); err != nil {
					return err
				}
			}
		}
		if err := writeAtom(ctx, a); err != nil {
			return err
		}
	}

	return nil
}

// process returns a new atom list with all the resources extracted and placed
// into the database. process also returns the merged interval list of all
// observed memory ranges.
func process(ctx context.Context, a *atom.List) (*atom.List, []*MemoryRange, error) {
	out := atom.NewList(make([]atom.Atom, 0, len(a.Atoms))...)
	rngs := interval.U64RangeList{}
	idmap := map[id.ID]id.ID{}
	for _, a := range a.Atoms {
		observations := a.Extras().Observations()

		if observations != nil {
			for _, rd := range observations.Reads {
				interval.Merge(&rngs, rd.Range.Span(), true)
			}
			for _, wr := range observations.Writes {
				interval.Merge(&rngs, wr.Range.Span(), true)
			}
		}

		switch a := a.(type) {
		case *atom.Resource:
			id, err := database.Store(ctx, a.Data)
			if err != nil {
				return nil, nil, err
			}
			if _, dup := idmap[a.ID]; dup {
				return nil, nil, log.Errf(ctx, nil, "Duplicate resource with ID: %v", a.ID)
			}
			idmap[a.ID] = id

		default:
			// Replace resource IDs from identifiers generated at capture time to
			// direct database identifiers. This avoids a database link indirection.
			if observations != nil {
				for i, r := range observations.Reads {
					if id, found := idmap[r.ID]; found {
						observations.Reads[i].ID = id
					}
				}
				for i, w := range observations.Writes {
					if id, found := idmap[w.ID]; found {
						observations.Writes[i].ID = id
					}
				}
			}
			out.Atoms = append(out.Atoms, a)
		}
	}

	return out, toMemoryRanges(rngs), nil
}

func toMemoryRanges(l interval.U64RangeList) []*MemoryRange {
	out := make([]*MemoryRange, len(l))
	for i, r := range l {
		out[i] = &MemoryRange{Base: r.First, Size: r.Count}
	}
	return out
}

func fromMemoryRanges(l []*MemoryRange) interval.U64RangeList {
	out := make(interval.U64RangeList, len(l))
	for i, r := range l {
		out[i] = interval.U64Range{First: r.Base, Count: r.Size}
	}
	return out
}
