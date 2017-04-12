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
	"bytes"
	"context"
	"io"
	"sync"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/pack"
	"github.com/google/gapid/core/data/protoconv"
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

type Capture struct {
	Name     string
	Header   *Header
	Atoms    []atom.Atom
	APIs     []gfxapi.API
	Observed interval.U64RangeList
}

func init() {
	protoconv.Register(toProto, fromProto)
}

// New returns a path to a new capture with the given name, header and atoms.
// The new capture is stored in the database.
func New(ctx context.Context, name string, header *Header, atoms []atom.Atom) (*path.Capture, error) {
	c, err := build(ctx, name, header, atoms)
	if err != nil {
		return nil, err
	}

	id, err := database.Store(ctx, c)
	if err != nil {
		return nil, err
	}

	capturesLock.Lock()
	captures = append(captures, id)
	capturesLock.Unlock()

	return &path.Capture{Id: path.NewID(id)}, nil
}

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
	freeList := memory.InvertMemoryRanges(c.Observed)
	interval.Remove(&freeList, interval.U64Span{Start: 0, End: value.FirstValidAddress})
	return gfxapi.NewStateWithAllocator(
		memory.NewBasicAllocator(freeList),
		c.Header.Abi.MemoryLayout,
	)
}

// Service returns the service.Capture description for this capture.
func (c *Capture) Service(ctx context.Context, p *path.Capture) *service.Capture {
	apis := make([]*path.API, len(c.APIs))
	for i, a := range c.APIs {
		apis[i] = &path.API{Id: path.NewID(id.ID(a.ID()))}
	}
	observations := make([]*service.MemoryRange, len(c.Observed))
	for i, o := range c.Observed {
		observations[i] = &service.MemoryRange{Base: o.First, Size: o.Count}
	}
	return &service.Capture{
		Name:         c.Name,
		Device:       c.Header.Device,
		Abi:          c.Header.Abi,
		NumCommands:  uint64(len(c.Atoms)),
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

// Import imports the capture by name and data, and stores it in the database.
func Import(ctx context.Context, name string, data []byte) (*path.Capture, error) {
	id, err := database.Store(ctx, &Record{
		Name: name,
		Data: data,
	})
	if err != nil {
		return nil, err
	}

	capturesLock.Lock()
	captures = append(captures, id)
	capturesLock.Unlock()

	return &path.Capture{Id: path.NewID(id)}, nil
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

// Export encodes the given capture and associated resources
// and writes it to the supplied io.Writer in the .gfxtrace format,
// producing output suitable for use with Import or opening in the trace editor.
func (c *Capture) Export(ctx context.Context, w io.Writer) error {
	write, err := pack.NewWriter(w)
	if err != nil {
		return err
	}

	writeMsg := func(ctx context.Context, a atom_pb.Atom) error { return write.Marshal(a) }

	if err := writeMsg(ctx, c.Header); err != nil {
		return err
	}

	writeAtom := atom.AtomToProto(func(a atom_pb.Atom) { writeMsg(ctx, a) })

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

	for _, a := range c.Atoms {
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

func toProto(ctx context.Context, c *Capture) (*Record, error) {
	buf := bytes.Buffer{}
	if err := c.Export(ctx, &buf); err != nil {
		return nil, err
	}
	return &Record{
		Name: c.Name,
		Data: buf.Bytes(),
	}, nil
}

func fromProto(ctx context.Context, r *Record) (*Capture, error) {
	reader, err := pack.NewReader(bytes.NewReader(r.Data))
	if err != nil {
		return nil, err
	}

	atoms := []atom.Atom{}
	convert := atom.ProtoToAtom(func(a atom.Atom) { atoms = append(atoms, a) })
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

	return build(ctx, r.Name, header, atoms)
}

// build creates a capture from the name, header and atoms.
// The atoms are inspected for APIs used and observed memory ranges.
// All resources are extracted placed into the database.
func build(ctx context.Context, name string, header *Header, atoms []atom.Atom) (*Capture, error) {
	out := &Capture{
		Name:     name,
		Header:   header,
		Observed: interval.U64RangeList{},
		APIs:     []gfxapi.API{},
	}

	idmap := map[id.ID]id.ID{}
	apiSet := map[gfxapi.ID]gfxapi.API{}

	for _, a := range atoms {
		if api := a.API(); api != nil {
			apiID := api.ID()
			if _, found := apiSet[apiID]; !found {
				apiSet[apiID] = api
				out.APIs = append(out.APIs, api)
			}
		}

		observations := a.Extras().Observations()

		if observations != nil {
			for _, rd := range observations.Reads {
				interval.Merge(&out.Observed, rd.Range.Span(), true)
			}
			for _, wr := range observations.Writes {
				interval.Merge(&out.Observed, wr.Range.Span(), true)
			}
		}

		switch a := a.(type) {
		case *atom.Resource:
			id, err := database.Store(ctx, a.Data)
			if err != nil {
				return nil, err
			}
			if _, dup := idmap[a.ID]; dup {
				return nil, log.Errf(ctx, nil, "Duplicate resource with ID: %v", a.ID)
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

	for _, api := range out.APIs {
		if aih, ok := api.(AtomsImportHandler); ok {
			var err error
			out.Atoms, err = aih.TransformAtomStream(ctx, out.Atoms)
			if err != nil {
				return nil, err
			}
		}
	}

	return out, nil
}
