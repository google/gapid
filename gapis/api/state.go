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
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/value"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/stringtable"
)

// GlobalState represents the graphics state across all contexts.
type GlobalState struct {
	// MemoryLayout holds information about the device memory layout that was
	// used to create the capture.
	MemoryLayout *device.MemoryLayout

	// Memory holds the memory state of the application.
	Memory memory.Pools

	// APIs holds the per-API context states.
	APIs map[ID]State

	// Allocator keeps track of and reserves memory areas not used in the trace.
	Allocator memory.Allocator

	// OnResourceCreated is called when a new resource is created.
	OnResourceCreated func(Resource)

	// OnResourceAccessed is called when a resource is used.
	OnResourceAccessed func(Resource)

	// OnResourceDestroyed is called when a resource is destroyed.
	OnResourceDestroyed func(Resource)

	// OnError is called when the command does not conform to the API.
	OnError func(err interface{})

	// NewMessage is called when there is a message to be passed to a report.
	NewMessage func(level log.Severity, msg *stringtable.Msg) uint32

	// AddTag is called when we want to tag report item.
	AddTag func(msgID uint32, msg *stringtable.Msg)
}

// State represents the graphics state for a single API.
type State interface {
	// All states belong to an API
	APIObject

	// Clone returns a deep copy of the state object.
	Clone() State

	// Root returns the path to the root of the state to display. It can vary
	// based on filtering mode. Returning nil, nil indicates there is no state
	// to show at this point in the capture.
	Root(ctx context.Context, p *path.State, r *path.ResolveConfig) (path.Node, error)

	// SetupInitialState sanitizes deserialized state to make it valid.
	// It can fill in any derived data which we choose not to serialize,
	// or it can apply backward-compatibility fixes for older traces.
	SetupInitialState(ctx context.Context, state *GlobalState)

	// TrimInitialState removes some parts of the state that are
	// not used by the capture commands.
	TrimInitialState(ctx context.Context, p *path.Capture) error
}

// NewStateWithEmptyAllocator returns a new, default-initialized State object,
// that uses an allocator with no allocations.
func NewStateWithEmptyAllocator(memoryLayout *device.MemoryLayout) *GlobalState {
	return NewStateWithAllocator(
		memory.NewBasicAllocator(value.ValidMemoryRanges),
		memoryLayout,
	)
}

// NewStateWithAllocator returns a new, default-initialized State object,
// that uses the given memory.Allocator instance.
func NewStateWithAllocator(allocator memory.Allocator, memoryLayout *device.MemoryLayout) *GlobalState {
	return &GlobalState{
		MemoryLayout: memoryLayout,
		Memory:       memory.NewPools(),
		APIs:         map[ID]State{},
		Allocator:    allocator,
	}
}

// ReserveMemory reserves the specifed memory ranges from the state's allocator,
// preventing them from being allocated.
// ReserveMemory is a fluent helper function for calling
// s.Allocator.ReserveMemory(rngs).
func (s *GlobalState) ReserveMemory(rngs interval.U64RangeList) *GlobalState {
	s.Allocator.ReserveRanges(rngs)
	return s
}

func (s GlobalState) String() string {
	apis := make([]string, 0, len(s.APIs))
	for a, s := range s.APIs {
		apis = append(apis, fmt.Sprintf("    %v: %v", a, s))
	}
	return fmt.Sprintf("GlobalState{\n  %v\n  Memory:\n%v\n  APIs:\n%v\n}",
		s.MemoryLayout, s.Memory, strings.Join(apis, "\n"))
}

// MemoryReader returns a binary reader using the state's memory endianness to
// read data from d.
func (s GlobalState) MemoryReader(ctx context.Context, d memory.Data) binary.Reader {
	return endian.Reader(d.NewReader(ctx), s.MemoryLayout.GetEndian())
}

// MemoryWriter returns a binary writer using the state's memory endianness to
// write data to the pool p, for the range rng.
func (s GlobalState) MemoryWriter(p memory.PoolID, rng memory.Range) binary.Writer {
	bw := memory.Writer(s.Memory.MustGet(p), rng)
	return endian.Writer(bw, s.MemoryLayout.GetEndian())
}

// MemoryDecoder returns a memory decoder using the state's memory layout to
// decode data from d.
func (s GlobalState) MemoryDecoder(ctx context.Context, d memory.Data) *memory.Decoder {
	return memory.NewDecoder(s.MemoryReader(ctx, d), s.MemoryLayout)
}

// MemoryEncoder returns a memory encoder using the state's memory layout
// to encode to the pool p, for the range rng.
func (s GlobalState) MemoryEncoder(p memory.PoolID, rng memory.Range) *memory.Encoder {
	return memory.NewEncoder(s.MemoryWriter(p, rng), s.MemoryLayout)
}

// Alloc allocates a memory range using the Allocator associated with
// the given State, and returns a AllocResult that can be used to access the
// pointer, and range.
func (s *GlobalState) Alloc(ctx context.Context, size uint64) (AllocResult, error) {
	at, err := s.Allocator.Alloc(size, 8)
	if err != nil {
		return AllocResult{}, err
	}
	return AllocResult{allocator: s.Allocator, rng: memory.Range{Base: at, Size: size}}, nil
}

// AllocData encodes and stores the value v to the database d, allocates a
// memory range big enough to store it using the Allocator associated with
// the given State, and returns a AllocResult that can be used to access the
// database ID, pointer, and range.
func (s *GlobalState) AllocData(ctx context.Context, v ...interface{}) (AllocResult, error) {
	buf := &bytes.Buffer{}
	e := memory.NewEncoder(endian.Writer(buf, s.MemoryLayout.GetEndian()), s.MemoryLayout)
	memory.Write(e, v)
	id, err := database.Store(ctx, buf.Bytes())
	if err != nil {
		return AllocResult{}, err
	}

	bufLength := uint64(len(buf.Bytes()))

	at, err := s.Allocator.Alloc(bufLength, 8)
	if err != nil {
		return AllocResult{}, err
	}
	return AllocResult{id: id, allocator: s.Allocator, rng: memory.Range{Base: at, Size: bufLength}}, nil
}

// AllocOrPanic is like Alloc, but panics if there's an error.
func (s *GlobalState) AllocOrPanic(ctx context.Context, size uint64) AllocResult {
	res, err := s.Alloc(ctx, size)
	if err != nil {
		panic(err)
	}
	return res
}

// AllocDataOrPanic is like AllocData, but panics if there's an error.
func (s *GlobalState) AllocDataOrPanic(ctx context.Context, v ...interface{}) AllocResult {
	res, err := s.AllocData(ctx, v...)
	if err != nil {
		panic(err)
	}
	return res
}

// AllocResult represents the result of allocating a range using
// a memory.Allocator, and potentially the database ID for data
// that's meant to be stored in the range.
type AllocResult struct {
	id        id.ID            // ID of the data stored in the range.
	allocator memory.Allocator // Allocator that allocated the range, for freeing.
	rng       memory.Range     // Allocated range.
}

// Free frees the memory range through the originating allocator.
// This is not currently used.
func (r AllocResult) Free() {
	r.allocator.Free(r.rng.Base)
}

// Data can be used as a helper to Add(Read|Write) methods on commands.
func (r AllocResult) Data() (memory.Range, id.ID) {
	return r.rng, r.id
}

// Range returns the associated memory.Range.
func (r AllocResult) Range() memory.Range {
	return r.rng
}

// Ptr returns a pointer to the beginning of the range.
func (r AllocResult) Ptr() memory.Pointer {
	return memory.BytePtr(r.rng.Base)
}

// Offset returns a pointer n bytes to the right of the associated range.
func (r AllocResult) Offset(n uint64) memory.Pointer {
	return memory.BytePtr(r.rng.Base + n)
}

// Address returns the beginning of the range.
func (r AllocResult) Address() uint64 {
	return r.rng.Base
}
