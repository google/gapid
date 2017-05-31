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

package atom

import (
	"bytes"
	"context"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/memory"
)

// Data encodes and stores the value v to the database d, returning the
// memory range and new resource identifier. Data can be used to as a helper
// to AddRead and AddWrite methods on atoms.
func Data(ctx context.Context, l *device.MemoryLayout, at memory.Pointer, v ...interface{}) (memory.Range, id.ID) {
	buf := &bytes.Buffer{}
	e := memory.NewEncoder(endian.Writer(buf, l.GetEndian()), l)
	memory.Write(e, v)
	id, err := database.Store(ctx, buf.Bytes())
	if err != nil {
		panic(err)
	}
	return memory.Range{Base: at.Address(), Size: uint64(len(buf.Bytes()))}, id
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

// Data can be used as a helper to Add(Read|Write) methods on atoms.
func (r AllocResult) Data() (memory.Range, id.ID) {
	return r.rng, r.id
}

// Range returns the associated memory.Range.
func (r AllocResult) Range() memory.Range {
	return r.rng
}

// Ptr returns the beginning of the range as an application pool pointer.
func (r AllocResult) Ptr() memory.Pointer {
	return memory.BytePtr(r.rng.Base, memory.ApplicationPool)
}

// Offset returns a pointer n bytes to the right of the associated range.
func (r AllocResult) Offset(n uint64) memory.Pointer {
	return memory.BytePtr(r.rng.Base+n, memory.ApplicationPool)
}

// Address returns the beginning of the range.
func (r AllocResult) Address() uint64 {
	return r.rng.Base
}

// AllocData encodes and stores the value v to the database d, allocates a
// memory range big enough to store it using the Allocator associated with
// the given State, and returns a helper that can be used to access the
// database ID, pointer, and range.
func AllocData(ctx context.Context, s *gfxapi.State, v ...interface{}) (AllocResult, error) {
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

// Must ensures that a previous call to AllocData or Alloc
// was successful, and discards the nil error, otherwise it
// panics.
func Must(result AllocResult, err error) AllocResult {
	if err != nil {
		panic(err)
	}
	return result
}

// Alloc allocates a memory range using the Allocator associated with
// the given State, and returns a helper that can be used to access the
// pointer, and range.
func Alloc(ctx context.Context, s *gfxapi.State, count uint64) (AllocResult, error) {
	at, err := s.Allocator.Alloc(count, 8)
	if err != nil {
		return AllocResult{}, err
	}
	return AllocResult{allocator: s.Allocator, rng: memory.Range{Base: at, Size: count}}, nil
}
