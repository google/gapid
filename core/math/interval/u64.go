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

package interval

// U64Span is the base interval type understood by the algorithms in this package.
// It is a half open interval that includes the lower bound, but not the upper.
type U64Span struct {
	Start uint64 // the value at which the interval begins
	End   uint64 // the next value not included in the interval.
}

// U64Range is an interval specified by a beginning and size.
type U64Range struct {
	First uint64 // the first value in the interval
	Count uint64 // the count of values in the interval
}

// U64SpanList implements List for an array of U64Span intervals
type U64SpanList []U64Span

// U64RangeList implements List for an array of U64Range intervals
type U64RangeList []U64Range

// Range converts a U64Span to a U64Range
func (s U64Span) Range() U64Range { return U64Range{First: s.Start, Count: s.End - s.Start} }

// Span converts a U64Range to a U64Span
func (r U64Range) Span() U64Span { return U64Span{Start: r.First, End: r.First + r.Count} }

func (l U64SpanList) Length() int                     { return len(l) }
func (l U64SpanList) GetSpan(index int) U64Span       { return l[index] }
func (l U64SpanList) SetSpan(index int, span U64Span) { l[index] = span }
func (l U64SpanList) New(index int, span U64Span)     { l[index] = span }
func (l U64SpanList) Copy(to, from, count int)        { copy(l[to:to+count], l[from:from+count]) }
func (l *U64SpanList) Resize(length int) {
	if cap(*l) > length {
		*l = (*l)[:length]
	} else {
		old := *l
		capacity := cap(*l) * 2
		if capacity < length {
			capacity = length
		}
		*l = make(U64SpanList, length, capacity)
		copy(*l, old)
	}
}

func (l U64RangeList) Length() int                     { return len(l) }
func (l U64RangeList) GetSpan(index int) U64Span       { return l[index].Span() }
func (l U64RangeList) SetSpan(index int, span U64Span) { l[index] = span.Range() }
func (l U64RangeList) New(index int, span U64Span)     { l[index] = span.Range() }
func (l U64RangeList) Copy(to, from, count int)        { copy(l[to:to+count], l[from:from+count]) }
func (l *U64RangeList) Resize(length int) {
	if cap(*l) > length {
		*l = (*l)[:length]
	} else {
		old := *l
		capacity := cap(*l) * 2
		if capacity < length {
			capacity = length
		}
		*l = make(U64RangeList, length, capacity)
		copy(*l, old)
	}
}

func (l U64RangeList) Clone() U64RangeList {
	res := make(U64RangeList, len(l))
	copy(res, l)
	return res
}
