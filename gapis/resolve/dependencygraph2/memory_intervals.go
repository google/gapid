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

package dependencygraph2

import (
	"github.com/google/gapid/core/math/interval"
)

// memoryWrite stores a WriteMemEffect, together with a memory span affected by that write.
// The span may be smaller than the whole write.
type memoryWrite struct {
	// effect WriteMemEffect
	node NodeID
	span interval.U64Span
}

// memoryWriteList represents a collection of memory writes, together with the regions of memory affected by each write.
// memoryWriteList implements the `interval.MutableList` interface, enabling the algorithms in `interval` for efficient queries and updates.
type memoryWriteList []memoryWrite

// Length returns the number of elements in the list
// Implements `interval.List.Length`
func (l *memoryWriteList) Length() int {
	return len(*l)
}

// GetSpan returns the span for the element at index in the list
// Implements `interval.List.GetSpan`
func (l *memoryWriteList) GetSpan(index int) interval.U64Span {
	return (*l)[index].span
}

// SetSpan sets the span for the element at index in the list
// Implements `interval.MutableList.SetSpan`
func (l *memoryWriteList) SetSpan(index int, span interval.U64Span) {
	(*l)[index].span = span
}

// New creates a new element at the specifed index with the specified span
// Implements `interval.MutableList.New`
func (l *memoryWriteList) New(index int, span interval.U64Span) {
	(*l)[index].span = span
}

// Copy count list entries
// Implements `interval.MutableList.Copy`
func (l *memoryWriteList) Copy(to, from, count int) {
	copy((*l)[to:to+count], (*l)[from:from+count])
}

// Resize adjusts the length of the array
// Implements `interval.MutableList.Resize`
func (l *memoryWriteList) Resize(length int) {
	if cap(*l) > length {
		*l = (*l)[:length]
	} else {
		old := *l
		capacity := cap(*l) * 2
		if capacity < length {
			capacity = length
		}
		*l = make(memoryWriteList, length, capacity)
		copy(*l, old)
	}
}

// memoryAccess stores a memory span together with a bool indicating whether that span has been written (true) or only read (false)
type memoryAccess struct {
	mode AccessMode
	span interval.U64Span
}

// memoryAccessList represents a collection of memory access
// memoryAccessList implements the `interval.MutableList` interface, enabling the algorithms in `interval` for efficient queries and updates.
type memoryAccessList []memoryAccess

// Length returns the number of elements in the list
// Implements `interval.List.Length`
func (l *memoryAccessList) Length() int {
	return len(*l)
}

// GetSpan returns the span for the element at index in the list
// Implements `interval.List.GetSpan`
func (l *memoryAccessList) GetSpan(index int) interval.U64Span {
	return (*l)[index].span
}

// SetSpan sets the span for the element at index in the list
// Implements `interval.MutableList.SetSpan`
func (l *memoryAccessList) SetSpan(index int, span interval.U64Span) {
	(*l)[index].span = span
}

// New creates a new element at the specifed index with the specified span
// Implements `interval.MutableList.New`
func (l *memoryAccessList) New(index int, span interval.U64Span) {
	(*l)[index].span = span
}

// Copy count list entries
// Implements `interval.MutableList.Copy`
func (l *memoryAccessList) Copy(to, from, count int) {
	copy((*l)[to:to+count], (*l)[from:from+count])
}

// Resize adjusts the length of the array
// Implements `interval.MutableList.Resize`
func (l *memoryAccessList) Resize(length int) {
	if cap(*l) > length {
		*l = (*l)[:length]
	} else {
		old := *l
		capacity := cap(*l) * 2
		if capacity < length {
			capacity = length
		}
		*l = make(memoryAccessList, length, capacity)
		copy(*l, old)
	}
}

func (l memoryAccessList) GetValue(index int) interface{} {
	return l[index].mode
}

func (l *memoryAccessList) SetValue(index int, value interface{}) {
	(*l)[index].mode = value.(AccessMode)
}

func (l *memoryAccessList) Insert(index int, count int) {
	*l = append(*l, make(memoryAccessList, count)...)
	if index != len(*l) && count > 0 {
		copy((*l)[index+count:], (*l)[index:])
	}
}

func (l *memoryAccessList) Delete(index int, count int) {
	if index+count != len(*l) && count > 0 {
		copy((*l)[index:], (*l)[index+count:])
	}
	*l = (*l)[:len(*l)-count]
}

func (l *memoryAccessList) AddRead(s interval.U64Span) {
	f := func(x interface{}) interface{} {
		if x == nil {
			return ACCESS_READ
		}
		// There is already an access. Always mark the plain read, but be
		// careful about the dependency read: if the same node already accessed
		// before with a write, then the read is not relevant for dependency
		// since it will be reading what the very same node just wrote.
		m := x.(AccessMode)
		m |= ACCESS_PLAIN_READ
		if m&ACCESS_DEP_WRITE == 0 {
			m |= ACCESS_DEP_READ
		}
		return m
	}
	interval.Update(l, s, f)
}

func (l *memoryAccessList) AddWrite(s interval.U64Span) {
	f := func(x interface{}) interface{} {
		if x == nil {
			return ACCESS_WRITE
		}
		m := x.(AccessMode)
		return m | ACCESS_WRITE
	}
	interval.Update(l, s, f)
}
