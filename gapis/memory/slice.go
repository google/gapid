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

package memory

import (
	"fmt"
	"reflect"

	"github.com/google/gapid/core/os/device"
)

// Slice is the interface implemented by types that represent a slice on
// a memory pool.
type Slice interface {
	// Root returns the original pointer this slice derives from.
	Root() uint64

	// Base returns the address of first element.
	Base() uint64

	// Count returns the number of elements in the slice.
	Count() uint64

	// Pool returns the the pool identifier.
	Pool() PoolID

	// ElementType returns the reflect.Type of the elements in the slice.
	ElementType() reflect.Type

	// ElementSize returns the size in bytes of a single element in the slice.
	ElementSize(*device.MemoryLayout) uint64

	// Range returns the memory range this slice represents in the underlying pool.
	Range(*device.MemoryLayout) Range

	// ISlice returns a sub-slice from this slice using start and end indices.
	ISlice(start, end uint64, m *device.MemoryLayout) Slice

	// IIndex returns a pointer to the i'th element in the slice.
	IIndex(i uint64, m *device.MemoryLayout) Pointer
}

// NewSlice returns a new Slice.
func NewSlice(root, base, count uint64, pool PoolID, elTy reflect.Type) Slice {
	return &sli{root, base, count, pool, elTy}
}

// sli is a slice of a basic type.
type sli struct {
	root  uint64
	base  uint64
	count uint64
	pool  PoolID
	elTy  reflect.Type
}

func (s sli) Root() uint64                              { return s.root }
func (s sli) Base() uint64                              { return s.base }
func (s sli) Count() uint64                             { return s.count }
func (s sli) Pool() PoolID                              { return s.pool }
func (s sli) ElementType() reflect.Type                 { return s.elTy }
func (s sli) ElementSize(m *device.MemoryLayout) uint64 { return SizeOf(s.elTy, m) }
func (s sli) Range(m *device.MemoryLayout) Range {
	return Range{s.base, s.ElementSize(m) * s.count}
}
func (s sli) ISlice(start, end uint64, m *device.MemoryLayout) Slice {
	if start > end {
		panic(fmt.Errorf("%v.ISlice start (%d) is greater than the end (%d)", s, start, end))
	}
	if end > s.count {
		panic(fmt.Errorf("%v.ISlice(%d, %d) - out of bounds", s, start, end))
	}
	return sli{root: s.root, base: s.base + start*s.ElementSize(m), count: end - start, pool: s.pool}
}
func (s sli) IIndex(i uint64, m *device.MemoryLayout) Pointer {
	if i >= s.count {
		panic(fmt.Errorf("%v.IIndex(%d) is out of bounds [0 - %d]", s, i, s.count-1))
	}
	return ptr{addr: s.base + i*s.ElementSize(m), pool: s.pool, elTy: s.elTy}
}
