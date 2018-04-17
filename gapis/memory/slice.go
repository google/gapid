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
)

// Slice is the interface implemented by types that represent a slice on
// a memory pool.
type Slice interface {
	// Root returns the original pointer this slice derives from.
	Root() uint64

	// Base returns the address of first element.
	Base() uint64

	// Size returns the size of the slice in bytes.
	Size() uint64

	// Count returns the number of elements in the slice.
	Count() uint64

	// Pool returns the the pool identifier.
	Pool() PoolID

	// ElementType returns the reflect.Type of the elements in the slice.
	ElementType() reflect.Type

	// ISlice returns a sub-slice from this slice using start and end indices.
	ISlice(start, end uint64) Slice
}

// NewSlice returns a new Slice.
func NewSlice(root, base, size, count uint64, pool PoolID, elTy reflect.Type) Slice {
	return &sli{root, base, size, count, pool, elTy}
}

// sli is a slice of a basic type.
type sli struct {
	root  uint64
	base  uint64
	size  uint64
	count uint64
	pool  PoolID
	elTy  reflect.Type
}

func (s sli) Root() uint64              { return s.root }
func (s sli) Base() uint64              { return s.base }
func (s sli) Size() uint64              { return s.size }
func (s sli) Count() uint64             { return s.count }
func (s sli) Pool() PoolID              { return s.pool }
func (s sli) ElementSize() uint64       { return s.Size() / s.Count() }
func (s sli) ElementType() reflect.Type { return s.elTy }
func (s sli) ISlice(start, end uint64) Slice {
	if start > end {
		panic(fmt.Errorf("%v.ISlice start (%d) is greater than the end (%d)", s, start, end))
	}
	if end > s.Count() {
		panic(fmt.Errorf("%v.ISlice(%d, %d) - out of bounds", s, start, end))
	}
	count := end - start
	elSize := s.ElementSize()
	return sli{
		root:  s.root,
		base:  s.base + start*elSize,
		size:  count * elSize,
		count: count,
		pool:  s.pool,
		elTy:  s.elTy,
	}
}
