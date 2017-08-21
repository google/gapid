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
		b.addCmd(ctx, cmd)
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
// and writes it to the supplied io.Writer in the .gfxtrace format.
func (c *Capture) Export(ctx context.Context, w io.Writer) error {
	write, err := pack.NewWriter(w)
	if err != nil {
		return err
	}

	// Write the capture header.
	if err := write.Object(ctx, c.Header); err != nil {
		return err
	}

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
		if err := write.Object(ctx, &Resource{Id: o.ID[:], Data: data.([]uint8)}); err != nil {
			return err
		}
		seen[o.ID] = true
		return nil
	}

	for _, cmd := range c.Commands {
		cmdProto, err := protoconv.ToProto(ctx, cmd)
		if err != nil {
			return err
		}
		cmdID, err := write.BeginGroup(ctx, cmdProto)
		if err != nil {
			return err
		}

		handledCall := false

		for _, e := range cmd.Extras().All() {
			switch e := e.(type) {
			case *api.CmdObservations:
				for _, o := range e.Reads {
					if err := encodeObservation(o); err != nil {
						return err
					}
					msg, err := protoconv.ToProto(ctx, o)
					if err != nil {
						return err
					}
					if err := write.ChildObject(ctx, msg, cmdID); err != nil {
						return err
					}
				}
				msg := api.CmdCallFor(cmd)
				if err := write.ChildObject(ctx, msg, cmdID); err != nil {
					return err
				}
				handledCall = true
				for _, o := range e.Writes {
					if err := encodeObservation(o); err != nil {
						return err
					}
					msg, err := protoconv.ToProto(ctx, o)
					if err != nil {
						return err
					}
					if err := write.ChildObject(ctx, msg, cmdID); err != nil {
						return err
					}
				}
			default:
				msg, err := protoconv.ToProto(ctx, e)
				if err != nil {
					return err
				}
				if err := write.ChildObject(ctx, msg, cmdID); err != nil {
					return err
				}
			}
		}

		if !handledCall {
			msg := api.CmdCallFor(cmd)
			if err := write.ChildObject(ctx, msg, cmdID); err != nil {
				return err
			}
		}

		if err := write.EndGroup(ctx, cmdID); err != nil {
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
	d := newDecoder()
	if err := pack.Read(ctx, bytes.NewReader(r.Data), d); err != nil {
		return nil, err
	}
	if d.header == nil {
		return nil, log.Err(ctx, nil, "Capture was missing header chunk")
	}
	return d.builder.build(r.Name, d.header), nil
}

type cmdInfo struct {
	invoked bool
	id      api.CmdID
}
type decoder struct {
	header  *Header
	builder *builder
	groups  map[uint64]interface{}
	cmds    map[api.Cmd]*cmdInfo
}

func newDecoder() *decoder {
	return &decoder{
		builder: newBuilder(),
		groups:  map[uint64]interface{}{},
		cmds:    map[api.Cmd]*cmdInfo{},
	}
}

func (d *decoder) unmarshal(ctx context.Context, in proto.Message) (interface{}, error) {
	obj, err := protoconv.ToObject(ctx, in)
	if err != nil {
		if e, ok := err.(protoconv.ErrNoConverterRegistered); ok && e.Object == in {
			return in, nil // No registered converter. Treat proto as the object.
		}
		return nil, err
	}
	return obj, nil
}

func (d *decoder) BeginGroup(ctx context.Context, msg proto.Message, id uint64) error {
	obj, err := d.decode(ctx, msg)
	if err != nil {
		return err
	}
	d.groups[id] = obj
	return nil
}

func (d *decoder) BeginChildGroup(ctx context.Context, msg proto.Message, id, parentID uint64) error {
	obj, err := d.decode(ctx, msg)
	if err != nil {
		return err
	}
	d.groups[id] = obj
	return d.add(ctx, obj, d.groups[parentID])
}

func (d *decoder) EndGroup(ctx context.Context, id uint64) error {
	obj := d.groups[id]
	delete(d.groups, id)

	switch obj := obj.(type) {
	case api.Cmd:
		delete(d.cmds, obj)
	}

	return nil
}

func (d *decoder) Object(ctx context.Context, msg proto.Message) error {
	_, err := d.decode(ctx, msg)
	return err
}

func (d *decoder) ChildObject(ctx context.Context, msg proto.Message, parentID uint64) error {
	obj, err := d.decode(ctx, msg)
	if err != nil {
		return err
	}
	return d.add(ctx, obj, d.groups[parentID])
}

func (d *decoder) add(ctx context.Context, child, parent interface{}) error {
	if cmd, ok := parent.(api.Cmd); ok {
		if res, ok := child.(proto.Message); ok {
			if cwr, ok := cmd.(api.CmdWithResult); ok {
				if cwr.SetResult(res) == nil {
					d.cmds[cmd].invoked = true
					return nil
				}
			}
		}

		switch obj := child.(type) {
		case api.Cmd:
			obj.SetCaller(d.cmds[cmd].id)

		case api.CmdObservation:
			d.builder.addObservation(ctx, &obj)
			observations := cmd.Extras().GetOrAppendObservations()
			if !d.cmds[cmd].invoked {
				observations.Reads = append(observations.Reads, obj)
			} else {
				observations.Writes = append(observations.Writes, obj)
			}

		case *api.CmdCall:
			d.cmds[cmd].invoked = true

		case api.CmdExtra:
			cmd.Extras().Add(obj)
		}
	}

	return nil
}

func (d *decoder) decode(ctx context.Context, in proto.Message) (interface{}, error) {
	obj, err := d.unmarshal(ctx, in)
	if err != nil {
		return nil, err
	}

	switch obj := obj.(type) {
	case *Header:
		d.header = obj
		return in, nil

	case *Resource:
		var rID id.ID
		copy(rID[:], obj.Id)
		if err := d.builder.addRes(ctx, rID, obj.Data); err != nil {
			return nil, err
		}
		return in, nil

	case api.Cmd:
		d.cmds[obj] = &cmdInfo{
			id: api.CmdID(len(d.builder.cmds)),
		}
		d.builder.cmds = append(d.builder.cmds, obj)
		d.builder.addAPI(ctx, obj.API())
	}

	return obj, nil
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

func (b *builder) addCmd(ctx context.Context, cmd api.Cmd) {
	b.addAPI(ctx, cmd.API())
	if observations := cmd.Extras().Observations(); observations != nil {
		for i := range observations.Reads {
			b.addObservation(ctx, &observations.Reads[i])
		}
		for i := range observations.Writes {
			b.addObservation(ctx, &observations.Writes[i])
		}
	}
	b.cmds = append(b.cmds, cmd)
}

func (b *builder) addAPI(ctx context.Context, api api.API) {
	if api != nil {
		apiID := api.ID()
		if _, found := b.seenAPIs[apiID]; !found {
			b.seenAPIs[apiID] = struct{}{}
			b.apis = append(b.apis, api)
		}
	}
}

func (b *builder) addObservation(ctx context.Context, o *api.CmdObservation) {
	interval.Merge(&b.observed, o.Range.Span(), true)
	if id, found := b.idmap[o.ID]; found {
		o.ID = id
	}
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
