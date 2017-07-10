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

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/pack"
	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/database"
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
	Commands []api.Cmd
	APIs     []api.API
	Observed interval.U64RangeList
}

func init() {
	protoconv.Register(toProto, fromProto)
}

// New returns a path to a new capture with the given name, header and commands.
// The new capture is stored in the database.
func New(ctx context.Context, name string, header *Header, cmds []api.Cmd) (*path.Capture, error) {
	b := newBuilder()
	for _, cmd := range cmds {
		b.addCmd(cmd)
	}
	c := b.build(name, header)

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
func NewState(ctx context.Context) (*api.State, error) {
	c, err := Resolve(ctx)
	if err != nil {
		return nil, err
	}
	return c.NewState(), nil
}

// NewState returns a new, default-initialized State object built for the
// capture.
func (c *Capture) NewState() *api.State {
	freeList := memory.InvertMemoryRanges(c.Observed)
	interval.Remove(&freeList, interval.U64Span{Start: 0, End: value.FirstValidAddress})
	return api.NewStateWithAllocator(
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
		NumCommands:  uint64(len(c.Commands)),
		Apis:         apis,
		Observations: observations,
	}
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
		return nil, log.Err(ctx, err, "Error resolving capture")
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

	writeMsg := func(ctx context.Context, m proto.Message) error { return write.Marshal(m) }

	if err := writeMsg(ctx, c.Header); err != nil {
		return err
	}

	writeAtom := api.CmdToProto(func(m proto.Message) { writeMsg(ctx, m) })

	// IDs seen, so we can avoid encoding the same resource data multiple times.
	seen := map[id.ID]bool{}

	encodeObservation := func(o api.CmdObservation) error {
		if seen[o.ID] {
			return nil
		}
		data, err := database.Resolve(ctx, o.ID)
		if err != nil {
			return err
		}
		err = writeMsg(ctx, &Resource{Id: o.ID[:], Data: data.([]uint8)})
		seen[o.ID] = true
		return err
	}

	for _, a := range c.Commands {
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

	b := newBuilder()
	convert := api.ProtoToCmd(b.addCmd)

	var header *Header
	for {
		msg, err := reader.Unmarshal()
		if err != nil {
			cause := errors.Cause(err)
			if cause == io.EOF || cause == io.ErrUnexpectedEOF {
				break
			}
			return nil, log.Err(ctx, err, "Failed to unmarshal")
		}
		switch msg := msg.(type) {
		case *Header:
			header = msg

		case *Resource:
			var rID id.ID
			copy(rID[:], msg.Id)
			if err := b.addRes(ctx, rID, msg.Data); err != nil {
				return nil, err
			}

		default:
			if err := convert(ctx, msg); err != nil {
				return nil, err
			}
		}
	}

	// must invoke the converter with nil to flush the last command
	if err := convert(ctx, nil); err != nil {
		return nil, err
	}

	if header == nil {
		return nil, log.Err(ctx, nil, "Capture was missing header chunk")
	}

	return b.build(r.Name, header), nil
}

type builder struct {
	idmap    map[id.ID]id.ID
	apis     []api.API
	seenAPIs map[api.ID]struct{}
	observed interval.U64RangeList
	cmds     []api.Cmd
}

func newBuilder() *builder {
	return &builder{
		idmap:    map[id.ID]id.ID{},
		apis:     []api.API{},
		seenAPIs: map[api.ID]struct{}{},
		observed: interval.U64RangeList{},
		cmds:     []api.Cmd{},
	}
}

func (b *builder) addCmd(cmd api.Cmd) {
	if api := cmd.API(); api != nil {
		apiID := api.ID()
		if _, found := b.seenAPIs[apiID]; !found {
			b.seenAPIs[apiID] = struct{}{}
			b.apis = append(b.apis, api)
		}
	}
	if observations := cmd.Extras().Observations(); observations != nil {
		for i, r := range observations.Reads {
			interval.Merge(&b.observed, r.Range.Span(), true)
			if id, found := b.idmap[r.ID]; found {
				observations.Reads[i].ID = id
			}
		}
		for i, w := range observations.Writes {
			interval.Merge(&b.observed, w.Range.Span(), true)
			if id, found := b.idmap[w.ID]; found {
				observations.Writes[i].ID = id
			}
		}
	}
	b.cmds = append(b.cmds, cmd)
}

func (b *builder) addRes(ctx context.Context, id id.ID, data []byte) error {
	dID, err := database.Store(ctx, data)
	if err != nil {
		return err
	}
	if _, dup := b.idmap[id]; dup {
		return log.Errf(ctx, nil, "Duplicate resource with ID: %v", id)
	}
	b.idmap[id] = dID
	return nil
}

func (b *builder) build(name string, header *Header) *Capture {
	return &Capture{
		Name:     name,
		Header:   header,
		Commands: b.cmds,
		Observed: b.observed,
		APIs:     b.apis,
	}
}
