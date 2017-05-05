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

import "sort"

// intersection holds the full set of calculated information when testing for
// list intersections
type intersection struct {
	overlap        int     // the count of intervals that overlap the span
	lowIndex       int     // the index of the low interval
	low            U64Span // the span at the low index
	intersectsLow  bool    // whether the low span intersected, or was just before
	highIndex      int     // the index of the high interval
	high           U64Span // the span at the high index
	intersectsHigh bool    // whether the high span intersected, or was just after
}

// findSpanFor searches a list for the span that encompasses a value, returning
// the index if found, or -1 otherwise
func findSpanFor(l List, value uint64) int {
	index := sort.Search(l.Length(), func(at int) bool {
		return value < l.GetSpan(at).Start
	})
	index--
	if index >= 0 {
		if value < l.GetSpan(index).End {
			return index
		}
	}
	return -1
}

// search the list for the first interval that matches the predicate.
// If no interval matches, it will return the list length.
func search(l List, t Predicate) int {
	i := 0
	j := l.Length()
	for i < j {
		h := i + (j-i)/2
		if !t(l.GetSpan(h)) {
			i = h + 1
		} else {
			j = h
		}
	}
	return i
}

// intersect a span with a list, calculating the intersection span and interval range
func (s *intersection) intersect(l List, span U64Span, expand bool) {
	var beforeLen, afterIndex int
	if expand {
		beforeLen = search(l, func(test U64Span) bool {
			return span.Start <= test.End
		})
		afterIndex = search(l, func(test U64Span) bool {
			return span.End < test.Start
		})
	} else {
		beforeLen = search(l, func(test U64Span) bool {
			return span.Start < test.End
		})
		afterIndex = search(l, func(test U64Span) bool {
			return span.End <= test.Start
		})
	}
	if afterIndex < beforeLen {
		afterIndex, beforeLen = beforeLen, afterIndex
	}
	s.lowIndex = beforeLen
	s.highIndex = afterIndex - 1
	s.overlap = afterIndex - beforeLen
	s.intersectsLow = false
	s.intersectsHigh = false
	if s.overlap > 0 {
		s.low = l.GetSpan(s.lowIndex)
		s.intersectsLow = s.low.Start < span.Start
		s.high = l.GetSpan(s.highIndex)
		s.intersectsHigh = span.End < s.high.End
	}
}

// merges a new span into a list, returning the index of the span
func merge(l MutableList, span U64Span, joinAdj bool) int {
	s := intersection{}
	s.intersect(l, span, joinAdj)
	adjust(l, s.lowIndex, 1-s.overlap)
	if s.intersectsLow {
		span.Start = s.low.Start
	}
	if s.intersectsHigh {
		span.End = s.high.End
	}
	l.SetSpan(s.lowIndex, span)
	return s.lowIndex
}

// cut slices a hole matching the specified span from a list.
// If add is true, it puts a new span in that space
// It is used to implement both Remove and Replace
func cut(l MutableList, span U64Span, add bool) (int, U64Span) {
	s := intersection{}
	s.intersect(l, span, false)
	if s.overlap == 0 {
		if add {
			adjust(l, s.lowIndex, 1)
		}
		return s.lowIndex, span
	}

	insertLen := 0
	insertPoint := s.lowIndex
	if s.intersectsLow {
		s.low.End = span.Start
		insertLen++
		insertPoint++
	}
	if add {
		insertLen++
	}
	if s.intersectsHigh {
		s.high.Start = span.End
		insertLen++
	}
	delta := insertLen - s.overlap
	adjust(l, insertPoint, delta)
	if s.intersectsLow {
		l.SetSpan(s.lowIndex, s.low)
	}
	if s.intersectsHigh {
		l.SetSpan(s.lowIndex+insertLen-1, s.high)
	}
	return insertPoint, span
}

// adjust implements list size adjustment logic, given a delta in size, and an
// index to adjust at
func adjust(l MutableList, at, delta int) {
	if delta == 0 {
		return
	}
	oldLen := l.Length()
	newLen := oldLen + delta
	if delta > 0 {
		l.Resize(newLen)
	}
	copyStart := at - delta
	copyTo := at
	if copyStart < 0 {
		copyTo -= copyStart
		copyStart = 0
	}
	l.Copy(copyTo, copyStart, newLen-copyTo)
	if delta < 0 {
		l.Resize(newLen)
	}
}
