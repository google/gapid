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

import "github.com/google/gapid/core/math/interval"

type RangeList []Range

// Length returns the number of ranges in the RangeList.
func (l *RangeList) Length() int {
	return len(*l)
}

// GetSpan returns the span of the range with the specified index in the
// RangeList.
func (l *RangeList) GetSpan(index int) interval.U64Span {
	return (*l)[index].Span()
}

// SetSpan adjusts the range of the span with the specified index in the
// RangeList.
func (l *RangeList) SetSpan(index int, span interval.U64Span) {
	(*l)[index] = Range{
		Base: span.Start,
		Size: span.End - span.Start,
	}
}

// New replaces the range of the span with the specified index in the
// RangeList.
func (l *RangeList) New(index int, span interval.U64Span) {
	l.SetSpan(index, span)
}

// Copy performs a copy of ranges within the RangeList.
func (l *RangeList) Copy(to, from, count int) {
	copy((*l)[to:to+count], (*l)[from:from+count])
}

// Resize resizes the RangeList to the specified length.
func (l *RangeList) Resize(length int) {
	if cap(*l) > length {
		*l = (*l)[:length]
	} else {
		old := *l
		capacity := cap(*l) * 2
		if capacity < length {
			capacity = length
		}
		*l = make(RangeList, length, capacity)
		copy(*l, old)
	}
}
