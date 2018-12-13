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

package builder

import (
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/value"
)

type mappedMemoryRange struct {
	memory.Range
	Target value.Pointer
}

// MappedMemoryRangeList represents a bunch of mapped memory ranges,
// and the location that they are stored in the heap.
type MappedMemoryRangeList []mappedMemoryRange

// Length returns the number of ranges in the MappedMemoryRangeList.
func (l *MappedMemoryRangeList) Length() int {
	return len(*l)
}

// GetSpan returns the span of the range with the specified index in the
// MappedMemoryRangeList.
func (l *MappedMemoryRangeList) GetSpan(index int) interval.U64Span {
	return (*l)[index].Span()
}

// SetSpan adjusts the range of the span with the specified index in the
// MappedMemoryRangeList.
func (l *MappedMemoryRangeList) SetSpan(index int, span interval.U64Span) {
	(*l)[index].Range = memory.Range{Base: span.Start, Size: span.End - span.Start}
}

// New replaces specified index in the MappedMemoryRangeList.
func (l *MappedMemoryRangeList) New(index int, span interval.U64Span) {
	(*l)[index] = mappedMemoryRange{
		Range: memory.Range{Base: span.Start, Size: span.End - span.Start},
	}
}

// Copy performs a copy of ranges within the MappedMemoryRangeList.
func (l *MappedMemoryRangeList) Copy(to, from, count int) {
	copy((*l)[to:to+count], (*l)[from:from+count])
}

// Resize resizes the MappedMemoryRangeList to the specified length.
func (l *MappedMemoryRangeList) Resize(length int) {
	if cap(*l) > length {
		*l = (*l)[:length]
	} else {
		old := *l
		capacity := cap(*l) * 2
		if capacity < length {
			capacity = length
		}
		*l = make(MappedMemoryRangeList, length, capacity)
		copy(*l, old)
	}
}
