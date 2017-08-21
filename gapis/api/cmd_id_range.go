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

package api

import (
	"fmt"

	"github.com/google/gapid/core/math/interval"
)

// CmdIDRange describes an interval of commands.
type CmdIDRange struct {
	Start CmdID // The first command within the range.
	End   CmdID // One past the last command within the range.
}

// String returns a string representing the range.
func (i CmdIDRange) String() string {
	return fmt.Sprintf("[%d..%d]", i.Start, i.End-1)
}

// Contains returns true if id is within the range, otherwise false.
func (i CmdIDRange) Contains(id CmdID) bool {
	return id >= i.Start && id < i.End
}

// Clamp returns the nearest index in the range to id.
func (i CmdIDRange) Clamp(id CmdID) CmdID {
	if id < i.Start {
		return i.Start
	}
	if id >= i.End {
		return i.End - 1
	}
	return id
}

// Length returns the number of commands in the range.
func (i CmdIDRange) Length() uint64 {
	return uint64(i.End - i.Start)
}

// CmdIDRange returns the start and end of the range.
func (i CmdIDRange) CmdIDRange() (start, end CmdID) {
	return i.Start, i.End
}

// First returns the first command index within the range.
func (i CmdIDRange) First() CmdID {
	return i.Start
}

// Last returns the last command index within the range.
func (i CmdIDRange) Last() CmdID {
	return i.End - 1
}

// Span returns the start and end of the range as a U64Span.
func (i CmdIDRange) Span() interval.U64Span {
	return interval.U64Span{Start: uint64(i.Start), End: uint64(i.End)}
}

// SetSpan sets the start and end range using a U64Span.
func (i *CmdIDRange) SetSpan(span interval.U64Span) {
	i.Start = CmdID(span.Start)
	i.End = CmdID(span.End)
}

// Split splits this range into two subranges where the first range will have
// a length no larger than the given value.
func (r CmdIDRange) Split(i uint64) (*CmdIDRange, *CmdIDRange) {
	if i >= r.Length() {
		return &r, &CmdIDRange{0, 0}
	}
	x := r.Start + CmdID(i)
	return &CmdIDRange{r.Start, x}, &CmdIDRange{x, r.End}
}
