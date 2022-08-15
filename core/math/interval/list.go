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

// List is the interface to an object that can be used as an interval list by
// the algorithms in this package.
type List interface {
	// Length returns the number of elements in the list
	Length() int
	// GetSpan returns the span for the element at index in the list
	GetSpan(index int) U64Span
}

// MutableList is a mutable form of a List.
type MutableList interface {
	List
	// SetSpan sets the span for the element at index in the list
	SetSpan(index int, span U64Span)
	// New creates a new element at the specifed index with the specified span
	New(index int, span U64Span)
	// Copy count list entries
	Copy(to, from, count int)
	// Resize adjusts the length of the array
	Resize(length int)
}

// Predicate is used as the condition for a Search
type Predicate func(test U64Span) bool

// Contains returns true if the value is found inside on of the intervals.
func Contains(l List, value uint64) bool {
	return findSpanFor(l, value) >= 0
}

// IndexOf returns the index of the span the value is a part of, or -1 if not found
func IndexOf(l List, value uint64) int {
	return findSpanFor(l, value)
}

// Merge adds a span to the list, merging it with existing spans if it overlaps
// them, and returns the index of that span.
// If the joinAdj parameter is true, then any intervals that are immediately
// adjacent to span will be merged with span.
// For example, consider the merging of intervals [0, 2] and [3, 5]:
//
// When joinAdj == false:
//
//	╭       ╮       ╭       ╮   ╭       ╮╭       ╮
//	│0  1  2│ merge │3  4  5│ = │0  1  2││3  4  5│
//	╰       ╯       ╰       ╯   ╰       ╯╰       ╯
//
// When join == true:
//
//	╭       ╮       ╭       ╮   ╭                ╮
//	│0  1  2│ merge │3  4  5│ = │0  1  2  3  4  5│
//	╰       ╯       ╰       ╯   ╰                ╯
func Merge(l MutableList, span U64Span, joinAdj bool) int {
	return merge(l, span, joinAdj)
}

// Replace cuts the span out of any existing intervals, and then adds a new interval,
// and returns its index.
func Replace(l MutableList, span U64Span) int {
	index, newSpan := cut(l, span, true)
	l.New(index, newSpan)
	return index
}

// Remove strips the specified span from the list, cutting it out of any
// overlapping intervals
func Remove(l MutableList, span U64Span) {
	cut(l, span, false)
}

// Intersect finds the intervals from the list that overlap with the specified span.
func Intersect(l List, span U64Span) (first, count int) {
	s := intersection{}
	s.intersect(l, span, false)
	return s.lowIndex, s.overlap
}

// Search finds the first interval in the list that the supplied predicate returns
// true for. If no interval matches the predicate, it returns the length of the list.
func Search(l List, t Predicate) int {
	return search(l, t)
}
