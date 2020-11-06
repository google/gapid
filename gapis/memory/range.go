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

	"github.com/google/gapid/core/math/interval"
)

func min(a, b uint64) uint64 {
	if a < b {
		return a
	} else {
		return b
	}
}

func max(a, b uint64) uint64 {
	if a > b {
		return a
	} else {
		return b
	}
}

// Range represents a region of memory.
type Range struct {
	Base uint64 // The address of the first byte in the memory range.
	Size uint64 // The size in bytes of the memory range.
}

// Expand returns a new Range that is grown to include the address addr.
func (i Range) Expand(addr uint64) Range {
	if i.Base > addr {
		i.Base = addr
	}
	if i.Last() < addr {
		i.Size += addr - i.Last()
	}
	return i
}

// Contains returns true if the address addr is within the Range.
func (i Range) Contains(addr uint64) bool {
	return i.First() <= addr && addr <= i.Last()
}

// Includes returns true if the Range r is included within the Range i.
func (i Range) Includes(r Range) bool {
	return i.First() <= r.First() && r.Last() <= i.Last()
}

// Overlaps returns true if other overlaps this memory range.
func (i Range) Overlaps(other Range) bool {
	a, b := i.Span(), other.Span()
	s, e := max(a.Start, b.Start), min(a.End, b.End)
	return s < e
}

// Intersect returns the range that is common between this Range and other.
// If the two memory ranges do not intersect, then this function panics.
func (i Range) Intersect(other Range) Range {
	a, b := i.Span(), other.Span()
	s, e := max(a.Start, b.Start), min(a.End, b.End)
	if e < s {
		panic(fmt.Errorf("Ranges %v and %v do not intersect", i, other))
	}
	return Range{Base: s, Size: e - s}
}

// Window returns the intersection of i and win, with the origin (0) address
// at win.Base.
// If the two memory ranges do not intersect, then this function panics.
func (i Range) Window(win Range) Range {
	r := i.Intersect(win)
	r.Base -= win.Base
	return r
}

// First returns the address of the first byte in the Range.
func (i Range) First() uint64 {
	return i.Base
}

// Last returns the address of the last byte in the Range.
func (i Range) Last() uint64 {
	return i.Base + i.Size - 1
}

// End returns the address of one byte beyond the end of the Range.
func (i Range) End() uint64 {
	return i.Base + i.Size
}

// Span returns the Range as a U64Span.
func (i Range) Span() interval.U64Span {
	return interval.U64Span{
		Start: i.Base,
		End:   i.Base + i.Size,
	}
}

func (i Range) String() string {
	return fmt.Sprintf("[0x%.16x-0x%.16x]", i.First(), i.Last())
}

// TrimLeft cuts count bytes from the left of the Range.
func (i Range) TrimLeft(count uint64) Range {
	if count > i.Size {
		panic(fmt.Errorf("Trying to trim %v by %d", i, count))
	}
	i.Base += count
	i.Size -= count
	return i
}
