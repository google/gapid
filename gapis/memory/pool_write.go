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

type poolWrite struct {
	dst Range
	src Data
}

// Span returns the Range as a U64Span.
func (w poolWrite) Span() interval.U64Span {
	return w.dst.Span()
}

type poolWriteList []poolWrite

// Length returns the number of ranges in the poolWriteList.
func (l *poolWriteList) Length() int {
	return len(*l)
}

// GetSpan returns the span of the range with the specified index in the
// poolWriteList.
func (l *poolWriteList) GetSpan(index int) interval.U64Span {
	return (*l)[index].Span()
}

// SetSpan adjusts the range of the span with the specified index in the
// poolWriteList.
func (l *poolWriteList) SetSpan(index int, span interval.U64Span) {
	w := &(*l)[index]
	w.src = w.src.Slice(Range{
		Base: span.Start - w.dst.Base,
		Size: span.End - span.Start,
	})
	w.dst = Range{
		Base: span.Start,
		Size: span.End - span.Start,
	}
}

func (l *poolWriteList) New(index int, span interval.U64Span) {
	(*l)[index].dst = Range{
		Base: span.Start,
		Size: span.End - span.Start,
	}
}

// Copy performs a copy of ranges within the poolWriteList.
func (l *poolWriteList) Copy(to, from, count int) {
	copy((*l)[to:to+count], (*l)[from:from+count])
}

// Resize resizes the poolWriteList to the specified length.
func (l *poolWriteList) Resize(length int) {
	if cap(*l) > length {
		*l = (*l)[:length]
	} else {
		old := *l
		capacity := cap(*l) * 2
		if capacity < length {
			capacity = length
		}
		*l = make(poolWriteList, length, capacity)
		copy(*l, old)
	}
}
