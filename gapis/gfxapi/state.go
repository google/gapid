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

package gfxapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/data/pod"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/value"
	"github.com/google/gapid/gapis/stringtable"
)

// State represents the graphics state across all contexts.
type State struct {
	binary.Object `java:"disable"`

	// MemoryLayout holds information about the device memory layout that was
	// used to create the capture.
	MemoryLayout *device.MemoryLayout

	// Memory holds the memory state of the application.
	Memory map[memory.PoolID]*memory.Pool

	// NextPoolID hold the identifier of the next Pool to be created.
	NextPoolID memory.PoolID

	// APIs holds the per-API context states.
	APIs map[API]binary.Object

	// Allocator keeps track of and reserves memory areas not used in the trace.
	Allocator memory.Allocator

	// OnResourceCreated is called when a new resource is created.
	OnResourceCreated func(Resource)

	// OnResourceAccessed is called when a resource is used.
	OnResourceAccessed func(Resource)

	// OnError is called when the command does not conform to the API.
	OnError func(err interface{})

	// NewMessage is called when there is a message to be passed to a report.
	NewMessage func(level log.Severity, msg *stringtable.Msg) uint32

	// AddTag is called when we want to tag report item.
	AddTag func(msgID uint32, msg *stringtable.Msg)
}

// NewStateWithEmptyAllocator returns a new, default-initialized State object,
// that uses an allocator with no allocations.
func NewStateWithEmptyAllocator(memoryLayout *device.MemoryLayout) *State {
	return NewStateWithAllocator(
		memory.NewBasicAllocator(value.ValidMemoryRanges),
		memoryLayout,
	)
}

// NewStateWithAllocator returns a new, default-initialized State object,
// that uses the given memory.Allocator instance.
func NewStateWithAllocator(allocator memory.Allocator, memoryLayout *device.MemoryLayout) *State {
	return &State{
		MemoryLayout: memoryLayout,
		Memory: map[memory.PoolID]*memory.Pool{
			memory.ApplicationPool: {},
		},
		NextPoolID: memory.ApplicationPool + 1,
		APIs:       map[API]binary.Object{},
		Allocator:  allocator,
	}
}

func (st State) String() string {
	mem := make([]string, 0, len(st.Memory))
	for i, p := range st.Memory {
		mem = append(mem, fmt.Sprintf("    %d: %v", i, strings.Replace(p.String(), "\n", "\n      ", -1)))
	}
	apis := make([]string, 0, len(st.APIs))
	for a, s := range st.APIs {
		apis = append(apis, fmt.Sprintf("    %v: %v", a, s))
	}
	return fmt.Sprintf("State{\n  %v\n  Memory:\n%v\n  APIs:\n%v\n}",
		st.MemoryLayout, strings.Join(mem, "\n"), strings.Join(apis, "\n"))
}

// MemoryDecoder returns an endian reader that uses the byte-order of the capture device to decode from the slice s.
func (st State) MemoryDecoder(ctx context.Context, s memory.Slice) pod.Reader {
	return endian.Reader(s.NewReader(ctx), st.MemoryLayout.GetEndian())
}

// MemoryEncoder returns an endian reader that uses the byte-order of the capture device to encode to the pool p,
// for the range rng.
func (st State) MemoryEncoder(p *memory.Pool, rng memory.Range) pod.Writer {
	bw := memory.Writer(p, rng)
	return endian.Writer(bw, st.MemoryLayout.GetEndian())
}
