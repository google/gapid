// Copyright (C) 2018 Google Inc.
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

import (
	"github.com/google/gapid/core/math/u64"
)

type ValueList interface {
	MutableList
	GetValue(index int) interface{}
	SetValue(index int, value interface{})
	Insert(index int, count int)
	Delete(index int, count int)
}

type ValueSpan struct {
	Span  U64Span
	Value interface{}
}

type ValueSpanList []ValueSpan

// Length returns the number of elements in the list
// Implements `List.Length`
func (l *ValueSpanList) Length() int {
	return len(*l)
}

// GetSpan returns the span for the element at index in the list
// Implements `List.GetSpan`
func (l *ValueSpanList) GetSpan(index int) U64Span {
	return (*l)[index].Span
}

// SetSpan sets the span for the element at index in the list
// Implements `MutableList.SetSpan`
func (l *ValueSpanList) SetSpan(index int, span U64Span) {
	(*l)[index].Span = span
}

// New creates a new element at the specifed index with the specified span
// Implements `MutableList.New`
func (l *ValueSpanList) New(index int, span U64Span) {
	(*l)[index].Span = span
}

// Copy count list entries
// Implements `MutableList.Copy`
func (l *ValueSpanList) Copy(to, from, count int) {
	copy((*l)[to:to+count], (*l)[from:from+count])
}

// Resize adjusts the length of the array
// Implements `MutableList.Resize`
func (l *ValueSpanList) Resize(length int) {
	if cap(*l) > length {
		*l = (*l)[:length]
	} else {
		old := *l
		capacity := cap(*l) * 2
		if capacity < length {
			capacity = length
		}
		*l = make(ValueSpanList, length, capacity)
		copy(*l, old)
	}
}

func (l ValueSpanList) GetValue(index int) interface{} {
	return l[index].Value
}

func (l *ValueSpanList) SetValue(index int, value interface{}) {
	(*l)[index].Value = value
}

func (l *ValueSpanList) Insert(index int, count int) {
	*l = append(*l, make(ValueSpanList, count)...)
	if index+count < len(*l) {
		copy((*l)[index+count:], (*l)[index:])
	}
}

func (l *ValueSpanList) Delete(index int, count int) {
	if index+count < len(*l) {
		copy((*l)[index:], (*l)[index+count:])
	}
	*l = (*l)[:len(*l)-count]
}

// Update modifies the values in `span` by applying the function `f`.
//   - Parts of `span` that are outside the intervals in `l` are inserted with
//     value `f(nil)`.
//   - If `f` returns `nil`, the corresponding span is removed.
//   - Adjacent intervals with the same value are merged.
func Update(l ValueList, span U64Span, f func(interface{}) interface{}) {
	k := Search(l, func(test U64Span) bool {
		return span.Start < test.End
	})
	elems := []ValueSpan{}

	add := func(val interface{}, start uint64, end uint64) {
		if start >= end || val == nil {
			return
		}
		if val == nil {
			span.Start = end
			return
		}
		if len(elems) > 0 {
			// check if new span should be merged with last existing span
			e := &elems[len(elems)-1]
			if e.Value == val && e.Span.End == start {
				e.Span.End = end
				span.Start = end
				return
			}
		}
		elems = append(elems, ValueSpan{U64Span{start, end}, val})
		span.Start = end
	}

	i := k

	if i < l.Length() {
		// Add the part of `a` before `span` (if it exists).
		// This can only exist for the first `a`, since after that
		// `span.Start` will always be the `End` of the previous `a`,
		// which precedes the current `a`.
		add(l.GetValue(i), l.GetSpan(i).Start, span.Start)
	}

	// For each overlapping span...
	for ; i < l.Length(); i++ {
		iSpan := l.GetSpan(i)
		if iSpan.Start >= span.End {
			break
		}

		// Add the part of `span` before `a`
		add(f(nil), span.Start, iSpan.Start)

		// Add the part of `span` that intersects `a`
		add(f(l.GetValue(i)), span.Start, u64.Min(iSpan.End, span.End))

		if iSpan.End > span.End {
			add(l.GetValue(i), span.End, iSpan.End)
		}
	}

	// Add the part of `span` after the last overlapping span
	add(f(nil), span.Start, span.End)

	if k > 0 && len(elems) > 0 {
		// Merge first `elems` with the previous span, if necessary
		s := l.GetSpan(k - 1)
		e := elems[0]
		if s.End == e.Span.Start && l.GetValue(k-1) == e.Value {
			s.End = e.Span.End
			l.SetSpan(k-1, s)
			elems = elems[1:]
		}
	}

	// Check for intervals that need to be merged
	if i < l.Length() {
		s := l.GetSpan(i)
		if len(elems) > 0 {
			// Merge the last `elems` with span `i`, if necessary
			e := elems[len(elems)-1]
			if s.Start == e.Span.End && l.GetValue(i) == e.Value {
				s.Start = e.Span.Start
				l.SetSpan(i, s)
				elems = elems[:len(elems)-1]
			}
		}
	}
	if len(elems) == 0 && 0 < k && i < l.Length() {
		// Not inserting any elements.
		// Merge span `k-1` with span `i`, if necessary
		si := l.GetSpan(i)
		sk := l.GetSpan(k - 1)
		if sk.End == si.Start && l.GetValue(k-1) == l.GetValue(i) {
			sk.End = si.End
			l.SetSpan(k-1, sk)
			i++
		}
	}

	// List elements `[k,i)` will be deleted, and `elems` will be inserted
	// at index `k`. This may require inserting or deleting elements.
	if len(elems) > i-k {
		// Make room for the new elements
		l.Insert(k, len(elems)-(i-k))
	} else if len(elems) < i-k {
		// Remove excess elements
		l.Delete(k, i-k-len(elems))
	}

	// Assign `elems` to the list indices `k ... k+len(elems)`
	for j, e := range elems {
		l.SetSpan(k+j, e.Span)
		l.SetValue(k+j, e.Value)
	}
}
