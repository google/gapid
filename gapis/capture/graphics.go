// Copyright (C) 2019 Google Inc.
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
	"bufio"
	"context"
	"fmt"
	"io"

	"github.com/google/gapid/core/app/analytics"
	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/pack"
	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/replay/value"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/pkg/errors"
)

const (
	// CurrentCaptureVersion is incremented on breaking changes to the capture format.
	// NB: Also update equally named field in gapii/cc/spy_base.cpp
	CurrentCaptureVersion int32 = 4
)

type ErrUnsupportedVersion struct{ Version int32 }

func (e ErrUnsupportedVersion) Error() string {
	return fmt.Sprintf("Unsupported capture format version: %+v", e.Version)
}

type GraphicsCapture struct {
	name         string
	Header       *Header
	Commands     []api.Cmd
	APIs         []api.API
	Observed     interval.U64RangeList
	InitialState *InitialState
	Messages     []*TraceMessage
}

// Name returns the capture's name.
func (g *GraphicsCapture) Name() string {
	return g.name
}

// Path returns the path of this capture in the database.
func (g *GraphicsCapture) Path(ctx context.Context) (*path.Capture, error) {
	return New(ctx, g)
}

type InitialState struct {
	Memory []api.CmdObservation
	APIs   map[api.API]api.State
}

func init() {
	protoconv.Register(
		func(ctx context.Context, in *InitialState) (*GlobalState, error) {
			return &GlobalState{}, nil
		},
		func(ctx context.Context, in *GlobalState) (*InitialState, error) {
			return &InitialState{APIs: map[api.API]api.State{}}, nil
		},
	)
}

// NewGraphicsCapture returns a new graphics capture with the given values.
func NewGraphicsCapture(ctx context.Context, name string, header *Header, initialState *InitialState, cmds []api.Cmd) (*GraphicsCapture, error) {
	b := newBuilder()
	if initialState != nil {
		for _, state := range initialState.APIs {
			b.addInitialState(ctx, state)
		}
		for _, mem := range initialState.Memory {
			b.addInitialMemory(ctx, mem)
		}
	}
	for _, cmd := range cmds {
		b.addCmd(ctx, cmd)
	}
	hdr := *header
	hdr.Version = CurrentCaptureVersion
	return b.build(name, &hdr), nil
}

// NewState returns a new, default-initialized State object built for the
// capture held by the context.
func NewState(ctx context.Context) (*api.GlobalState, error) {
	c, err := ResolveGraphics(ctx)
	if err != nil {
		return nil, err
	}
	return c.NewState(ctx), nil
}

// NewUninitializedState returns a new, uninitialized GlobalState built for the
// capture c. The returned state does not contain the capture's mid-execution
// state.
func (c *GraphicsCapture) NewUninitializedState(ctx context.Context) *api.GlobalState {
	freeList := memory.InvertMemoryRanges(c.Observed)
	interval.Remove(&freeList, interval.U64Span{Start: 0, End: value.FirstValidAddress})
	s := api.NewStateWithAllocator(
		memory.NewBasicAllocator(freeList),
		c.Header.ABI.MemoryLayout,
	)
	return s
}

func (c *GraphicsCapture) NewUninitializedStateSharingAllocator(ctx context.Context, oldGlobalState *api.GlobalState) *api.GlobalState {
	s := api.NewStateWithAllocator(
		oldGlobalState.Allocator,
		c.Header.ABI.MemoryLayout,
	)
	return s
}

// NewState returns a new, initialized GlobalState object built for the capture
// c. If the capture contains a mid-execution state, then this will be copied
// into the returned state.
func (c *GraphicsCapture) NewState(ctx context.Context) *api.GlobalState {
	out := c.NewUninitializedState(ctx)
	if c.InitialState != nil {
		ctx = status.Start(ctx, "CloneState")
		defer status.Finish(ctx)

		// Rebuild all the writes into the memory pools.
		for _, m := range c.InitialState.Memory {
			pool, _ := out.Memory.Get(memory.PoolID(m.Pool))
			if pool == nil {
				pool = out.Memory.NewAt(memory.PoolID(m.Pool))
			}
			pool.Write(m.Range.Base, memory.Resource(m.ID, m.Range.Size))
		}
		// Clone serialized state, and initialize it for use.
		for k, v := range c.InitialState.APIs {
			s := v.Clone()
			s.SetupInitialState(ctx, out)
			out.APIs[k.ID()] = s
		}
	}
	return out
}

// CloneInitialState clones this capture's initial state and returns it.
func (c *GraphicsCapture) CloneInitialState() *InitialState {
	if c.InitialState == nil {
		return nil
	}

	is := &InitialState{
		Memory: c.InitialState.Memory,
		APIs:   make(map[api.API]api.State, len(c.InitialState.APIs)),
	}
	for api, s := range c.InitialState.APIs {
		is.APIs[api] = s.Clone()
	}

	return is
}

// Service returns the service.Capture description for this capture.
func (c *GraphicsCapture) Service(ctx context.Context, p *path.Capture) *service.Capture {
	apis := make([]*path.API, len(c.APIs))
	for i, a := range c.APIs {
		apis[i] = &path.API{ID: path.NewID(id.ID(a.ID()))}
	}
	var observations []*service.MemoryRange
	if !p.ExcludeMemoryRanges {
		observations = make([]*service.MemoryRange, len(c.Observed))
		for i, o := range c.Observed {
			observations[i] = &service.MemoryRange{Base: o.First, Size: o.Count}
		}
	}
	return &service.Capture{
		Type:         service.TraceType_Graphics,
		Name:         c.name,
		Device:       c.Header.Device,
		ABI:          c.Header.ABI,
		NumCommands:  uint64(len(c.Commands)),
		APIs:         apis,
		Observations: observations,
	}
}

// Export encodes the given capture and associated resources
// and writes it to the supplied io.Writer in the .gfxtrace format.
func (c *GraphicsCapture) Export(ctx context.Context, w io.Writer) error {
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

func isGFXTraceFormat(in *bufio.Reader) bool {
	return pack.CheckMagic(in)
}

func deserializeGFXTrace(ctx context.Context, r *Record, in io.Reader) (out *GraphicsCapture, err error) {
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

	// The decoder implements the ID Remapper interface,
	// which protoconv functions need to handle resources.
	ctx = id.PutRemapper(ctx, d)

	if err := pack.Read(ctx, in, d, false); err != nil {
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

	for _, api := range d.builder.apis {
		if api.Name() == "gles" {
			return nil, &service.ErrUnsupportedVersion{
				Reason: messages.ErrFileOpenGl(),
			}
		}
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
	messages     []*TraceMessage
}

func newBuilder() *builder {
	return &builder{
		apis:         []api.API{},
		seenAPIs:     map[api.ID]struct{}{},
		observed:     interval.U64RangeList{},
		cmds:         []api.Cmd{},
		resIDs:       []id.ID{id.ID{}},
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

func (b *builder) addMessage(ctx context.Context, t *TraceMessage) {
	b.messages = append(b.messages, &TraceMessage{Timestamp: t.Timestamp, Message: t.Message})
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

func (b *builder) build(name string, header *Header) *GraphicsCapture {
	for _, api := range b.apis {
		analytics.SendEvent("capture", "uses-api", api.Name())
	}
	return &GraphicsCapture{
		name:         name,
		Header:       header,
		Commands:     b.cmds,
		Observed:     b.observed,
		APIs:         b.apis,
		InitialState: b.initialState,
		Messages:     b.messages,
	}
}
