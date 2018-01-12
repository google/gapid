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
	"math"
	"sort"

	"github.com/google/gapid/core/math/interval"
)

// Allocator is a memory allocator.
type Allocator interface {
	// Alloc allocates count bytes at an offset that is a multiple of align.
	Alloc(count, align uint64) (uint64, error)

	// Free marks the block starting at the given offset as free.
	Free(base uint64) error

	// AllocList returns the ranges currently allocated from this allocator.
	AllocList() interval.U64RangeList

	// FreeList returns the free ranges this allocator can allocate from.
	FreeList() interval.U64RangeList

	// ReserveRanges reserves the given ranges in the free-list, meaning
	// they cannot be allocated from
	ReserveRanges(interval.U64RangeList)
}

// BasicAllocator is a simple memory range allocator
// based on a list of free ranges.
type basicAllocator struct {
	freeList    interval.U64RangeList
	allocations map[uint64]uint64
}

// Alloc implements Allocator.
func (c *basicAllocator) Alloc(count, align uint64) (uint64, error) {
	for _, chunk := range c.freeList {
		pad := align - chunk.First%align
		if pad == align {
			pad = 0
		}
		base := chunk.First + pad
		if base+count <= chunk.First+chunk.Count {
			interval.Remove(&c.freeList, interval.U64Span{Start: base, End: base + count})
			c.allocations[base] = count
			return base, nil
		}
	}
	return 0, fmt.Errorf("Not enough contiguous free space to allocate %d bytes", count)
}

// Free implements Allocator.
func (c *basicAllocator) Free(base uint64) error {
	size, ok := c.allocations[base]
	if !ok {
		return fmt.Errorf("Attempted to free with an unknown offset %v", base)
	}
	delete(c.allocations, base)
	interval.Merge(&c.freeList, interval.U64Span{Start: base, End: base + size}, true)
	return nil
}

type rangeListByFirst interval.U64RangeList

func (l rangeListByFirst) Len() int           { return len(l) }
func (l rangeListByFirst) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l rangeListByFirst) Less(i, j int) bool { return l[i].First < l[j].First }

// AllocList implements Allocator.
func (c *basicAllocator) AllocList() interval.U64RangeList {
	res := make(interval.U64RangeList, 0, len(c.allocations))
	for base, count := range c.allocations {
		res = append(res, interval.U64Range{First: base, Count: count})
	}
	sort.Sort(rangeListByFirst(res))
	return res
}

// FreeList implements Allocator.
func (c *basicAllocator) FreeList() interval.U64RangeList {
	return c.freeList.Clone()
}

// ReserveRanges removes the given ranges from the freelist.
func (c *basicAllocator) ReserveRanges(ranges interval.U64RangeList) {
	for _, r := range ranges {
		interval.Remove(&c.freeList, r.Span())
	}
}

// NewBasicAllocator creates a new allocator which allocates
// memory from the given list of free ranges. Memory is allocated
// by finding the leftmost free block large enough to fit the
// required size with the specified alignment. The returned allocator
// is not thread-safe.
func NewBasicAllocator(freeList interval.U64RangeList) Allocator {
	return &basicAllocator{
		freeList:    freeList.Clone(),
		allocations: make(map[uint64]uint64),
	}
}

// InvertMemoryRanges converts a used memory range list
// into a free list and the other way around.
func InvertMemoryRanges(inputList interval.U64RangeList) interval.U64RangeList {
	invertedList := interval.U64RangeList{}

	add := func(span interval.U64Span) {
		if span.End-span.Start > 0 {
			invertedList = append(invertedList, span.Range())
		}
	}

	last := interval.U64Span{}
	for i := 0; i < inputList.Length(); i++ {
		span := inputList.GetSpan(i)
		if span.Range().Count > 0 {
			add(interval.U64Span{Start: last.End, End: span.Start})
			last = span
		}
	}
	add(interval.U64Span{Start: last.End, End: math.MaxUint64})

	return invertedList
}
