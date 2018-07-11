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
	"fmt"
	"io"
	"sync"

	"github.com/google/gapid/core/app/analytics"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/pack"
	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/messages"
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

const (
	// CurrentCaptureVersion is incremented on breaking changes to the capture format.
	// NB: Also update equally named field in spy_base.cpp
	CurrentCaptureVersion int32 = 3
)

type ErrUnsupportedVersion struct{ Version int32 }

func (e ErrUnsupportedVersion) Error() string {
	return fmt.Sprintf("Unsupported capture format version: %+v", e.Version)
}

type Capture struct {
	Name         string
	Header       *Header
	Commands     []api.Cmd
	APIs         []api.API
	Observed     interval.U64RangeList
	InitialState *InitialState
	Arena        arena.Arena
}

type InitialState struct {
	Memory []api.CmdObservation
	APIs   map[api.API]api.State
}

func init() {
	protoconv.Register(toProto, fromProto)
	protoconv.Register(
		func(ctx context.Context, in *InitialState) (*GlobalState, error) {
			return &GlobalState{}, nil
		},
		func(ctx context.Context, in *GlobalState) (*InitialState, error) {
			return &InitialState{APIs: map[api.API]api.State{}}, nil
		},
	)
}

// New returns a path to a new capture with the given name, header and commands.
// The new capture is stored in the database.
func New(ctx context.Context, name string, header *Header, initialState *InitialState, cmds []api.Cmd) (*path.Capture, error) {
	b := newBuilder()
	for _, state := range initialState.APIs {
		b.addInitialState(ctx, state)
	}
	for _, mem := range initialState.Memory {
		b.addInitialMemory(ctx, mem)
	}
	for _, cmd := range cmds {
		b.addCmd(ctx, cmd)
	}
	hdr := *header
	hdr.Version = CurrentCaptureVersion
	c := b.build(name, &hdr)

	id, err := database.Store(ctx, c)
	if err != nil {
		return nil, err
	}

	capturesLock.Lock()
	captures = append(captures, id)
	capturesLock.Unlock()

	return &path.Capture{ID: path.NewID(id)}, nil
}

// NewState returns a new, default-initialized State object built for the
// capture held by the context.
func NewState(ctx context.Context) (*api.GlobalState, error) {
	c, err := Resolve(ctx)
	if err != nil {
		return nil, err
	}
	return c.NewState(ctx), nil
}

// NewUninitializedState returns a new, uninitialized State object built for the capture.
func (c *Capture) NewUninitializedState(ctx context.Context, rngs interval.U64RangeList) *api.GlobalState {
	freeList := memory.InvertMemoryRanges(c.Observed)
	for _, r := range rngs {
		interval.Remove(&freeList, r.Span())
	}
	interval.Remove(&freeList, interval.U64Span{Start: 0, End: value.FirstValidAddress})
	s := api.NewStateWithAllocator(
		memory.NewBasicAllocator(freeList),
		c.Header.ABI.MemoryLayout,
	)
	return s
}

// NewState returns a new, default-initialized State object built for the
// capture.
func (c *Capture) NewState(ctx context.Context) *api.GlobalState {
	s := c.NewUninitializedState(ctx, interval.U64RangeList{})
	if c.InitialState != nil {
		for _, m := range c.InitialState.Memory {
			pool, _ := s.Memory.Get(memory.PoolID(m.Pool))
			if pool == nil {
				pool = s.Memory.NewAt(memory.PoolID(m.Pool))
			}
			pool.Write(m.Range.Base, memory.Resource(m.ID, m.Range.Size))
		}
		for k, v := range c.InitialState.APIs {
			s.APIs[k.ID()] = v.Clone(s.Arena)
		}
		for _, v := range s.APIs {
			v.SetupInitialState(ctx, s)
		}
	}
	return s
}

// BuildInitialCommands returns a set of commands which will setup the initial state.
func (c *Capture) BuildInitialCommands(ctx context.Context) ([]api.Cmd, interval.U64RangeList) {
	ranges := interval.U64RangeList{}
	cmds := []api.Cmd{}
	// TODO: This can be easily cached for the given capture.
	if c.InitialState != nil {
		s := c.NewState(ctx)
		for _, v := range s.APIs {
			s, r := v.RebuildState(ctx, s)
			ranges = append(ranges, r...)
			cmds = append(cmds, s...)
		}
		return cmds, ranges
	}
	return nil, interval.U64RangeList{}
}

// Service returns the service.Capture description for this capture.
func (c *Capture) Service(ctx context.Context, p *path.Capture) *service.Capture {
	apis := make([]*path.API, len(c.APIs))
	for i, a := range c.APIs {
		apis[i] = &path.API{ID: path.NewID(id.ID(a.ID()))}
	}
	observations := make([]*service.MemoryRange, len(c.Observed))
	for i, o := range c.Observed {
		observations[i] = &service.MemoryRange{Base: o.First, Size: o.Count}
	}
	return &service.Capture{
		Name:         c.Name,
		Device:       c.Header.Device,
		ABI:          c.Header.ABI,
		NumCommands:  uint64(len(c.Commands)),
		APIs:         apis,
		Observations: observations,
	}
}

// Captures returns all the captures stored by the database by identifier.
func Captures() []*path.Capture {
	capturesLock.RLock()
	defer capturesLock.RUnlock()
	out := make([]*path.Capture, len(captures))
	for i, c := range captures {
		out[i] = &path.Capture{ID: path.NewID(c)}
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
	return ResolveFromID(ctx, p.ID.ID())
}

// Import imports the capture by name and data, and stores it in the database.
func Import(ctx context.Context, name string, data []byte) (*path.Capture, error) {
	dataID, err := database.Store(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("Unable to store capture data: %v", err)
	}
	id, err := database.Store(ctx, &Record{
		Name: name,
		Data: dataID[:],
	})
	if err != nil {
		return nil, err
	}

	capturesLock.Lock()
	captures = append(captures, id)
	capturesLock.Unlock()

	return &path.Capture{ID: path.NewID(id)}, nil
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
	writer, err := pack.NewWriter(w)
	if err != nil {
		return err
	}
	e := newEncoder(c, writer)

	// The encoder implements the ID Remapper interface,
	// which protoconv functions need to handle resources.
	ctx = id.PutRemapper(ctx, e)

	return e.encode(ctx)
}

func toProto(ctx context.Context, c *Capture) (*Record, error) {
	buf := bytes.Buffer{}
	if err := c.Export(ctx, &buf); err != nil {
		return nil, err
	}
	id, err := database.Store(ctx, buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("Unable to store capture data: %v", err)
	}
	return &Record{
		Name: c.Name,
		Data: id[:],
	}, nil
}

func fromProto(ctx context.Context, r *Record) (out *Capture, err error) {
	var dataID id.ID
	copy(dataID[:], r.Data)
	data, err := database.Resolve(ctx, dataID)
	if err != nil {
		return nil, fmt.Errorf("Unable to load capture data: %v", err)
	}

	stopTiming := analytics.SendTiming("capture", "deserialize")
	defer func() {
		size := len(r.Data)
		count := 0
		if out != nil {
			count = len(out.Commands)
		}
		stopTiming(analytics.Size(size), analytics.Count(count))
	}()

	d := newDecoder()

	// Bind the arena used to for all allocations for this capture.
	ctx = arena.Put(ctx, d.builder.arena)

	// The decoder implements the ID Remapper interface,
	// which protoconv functions need to handle resources.
	ctx = id.PutRemapper(ctx, d)

	if err := pack.Read(ctx, bytes.NewReader(data.([]byte)), d, false); err != nil {
		switch err := errors.Cause(err).(type) {
		case pack.ErrUnsupportedVersion:
			log.E(ctx, "%v", err)
			switch {
			case err.Version.Major > pack.MaxMajorVersion:
				return nil, &service.ErrUnsupportedVersion{
					Reason:        messages.ErrFileTooNew(),
					SuggestUpdate: true,
				}
			case err.Version.Major < pack.MinMajorVersion:
				return nil, &service.ErrUnsupportedVersion{
					Reason: messages.ErrFileTooOld(),
				}
			default:
				return nil, &service.ErrUnsupportedVersion{
					Reason: messages.ErrFileCannotBeRead(),
				}
			}
		case ErrUnsupportedVersion:
			switch {
			case err.Version > CurrentCaptureVersion:
				return nil, &service.ErrUnsupportedVersion{
					Reason:        messages.ErrFileTooNew(),
					SuggestUpdate: true,
				}
			case err.Version < CurrentCaptureVersion:
				return nil, &service.ErrUnsupportedVersion{
					Reason: messages.ErrFileTooOld(),
				}
			default:
				return nil, &service.ErrUnsupportedVersion{
					Reason: messages.ErrFileCannotBeRead(),
				}
			}
		}
		return nil, err
	}
	d.flush(ctx)
	if d.header == nil {
		return nil, log.Err(ctx, nil, "Capture was missing header chunk")
	}
	return d.builder.build(r.Name, d.header), nil
}

type builder struct {
	apis         []api.API
	seenAPIs     map[api.ID]struct{}
	observed     interval.U64RangeList
	cmds         []api.Cmd
	resIDs       []id.ID
	initialState *InitialState
	arena        arena.Arena
}

func newBuilder() *builder {
	return &builder{
		apis:         []api.API{},
		seenAPIs:     map[api.ID]struct{}{},
		observed:     interval.U64RangeList{},
		cmds:         []api.Cmd{},
		resIDs:       []id.ID{id.ID{}},
		arena:        arena.New(),
		initialState: &InitialState{APIs: map[api.API]api.State{}},
	}
}

func (b *builder) addCmd(ctx context.Context, cmd api.Cmd) api.CmdID {
	b.addAPI(ctx, cmd.API())
	if observations := cmd.Extras().Observations(); observations != nil {
		for i := range observations.Reads {
			b.addObservation(ctx, &observations.Reads[i])
		}
		for i := range observations.Writes {
			b.addObservation(ctx, &observations.Writes[i])
		}
	}
	id := api.CmdID(len(b.cmds))
	b.cmds = append(b.cmds, cmd)
	return id
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
}

func (b *builder) addRes(ctx context.Context, expectedIndex int64, data []byte) error {
	dID, err := database.Store(ctx, data)
	if err != nil {
		return err
	}
	arrayIndex := int64(len(b.resIDs))
	b.resIDs = append(b.resIDs, dID)
	// If the Resource had the optional Index field, use it for verification.
	if expectedIndex != 0 && arrayIndex != expectedIndex {
		panic(fmt.Errorf("Resource has array index %v but we expected %v", arrayIndex, expectedIndex))
	}
	return nil
}

func (b *builder) addInitialState(ctx context.Context, state api.State) error {
	if _, ok := b.initialState.APIs[state.API()]; ok {
		return fmt.Errorf("We have more than one set of initial state for API %v", state.API())
	}
	b.initialState.APIs[state.API()] = state
	b.addAPI(ctx, state.API())
	return nil
}

func (b *builder) addInitialMemory(ctx context.Context, mem api.CmdObservation) error {
	b.initialState.Memory = append(b.initialState.Memory, mem)
	b.addObservation(ctx, &mem)
	return nil
}

func (b *builder) build(name string, header *Header) *Capture {
	for _, api := range b.apis {
		analytics.SendEvent("capture", "uses-api", api.Name())
	}
	// TODO: Mark the arena as read-only.
	return &Capture{
		Name:         name,
		Header:       header,
		Commands:     b.cmds,
		Observed:     b.observed,
		APIs:         b.apis,
		InitialState: b.initialState,
		Arena:        b.arena,
	}
}
