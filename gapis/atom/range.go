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
	"fmt"

	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/api"
)

// Range describes an interval of atoms in a stream.
type Range struct {
	Start api.CmdID // The first atom within the range.
	End   api.CmdID // One past the last atom within the range.
}

// String returns a string representing the range.
func (i Range) String() string {
	return fmt.Sprintf("[%d..%d]", i.Start, i.End-1)
}

// Contains returns true if atomIndex is within the range, otherwise false.
func (i Range) Contains(id api.CmdID) bool {
	return id >= i.Start && id < i.End
}

// Clamp returns the nearest index in the range to id.
func (i Range) Clamp(id api.CmdID) api.CmdID {
	if id < i.Start {
		return i.Start
	}
	if id >= i.End {
		return i.End - 1
	}
	return id
}

// Length returns the number of atoms in the range.
func (i Range) Length() uint64 {
	return uint64(i.End - i.Start)
}

// Range returns the start and end of the range.
func (i Range) Range() (start, end api.CmdID) {
	return i.Start, i.End
}

// First returns the first atom index within the range.
func (i Range) First() api.CmdID {
	return i.Start
}

// Last returns the last atom index within the range.
func (i Range) Last() api.CmdID {
	return i.End - 1
}

// Span returns the start and end of the range as a U64Span.
func (i Range) Span() interval.U64Span {
	return interval.U64Span{Start: uint64(i.Start), End: uint64(i.End)}
}

// SetSpan sets the start and end range using a U64Span.
func (i *Range) SetSpan(span interval.U64Span) {
	i.Start = api.CmdID(span.Start)
	i.End = api.CmdID(span.End)
}

// Split splits this range into two subranges where the first range will have
// a length no larger than the given value.
func (r Range) Split(i uint64) (Range, Range) {
	if i >= r.Length() {
		return r, Range{0, 0}
	}
	x := r.Start + api.CmdID(i)
	return Range{r.Start, x}, Range{x, r.End}
}
