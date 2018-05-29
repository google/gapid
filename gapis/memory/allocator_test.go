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

import (
	"math"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/math/interval"
)

func TestInvertMemoryRanges(t *testing.T) {
	assert := assert.To(t)

	const heapSize = math.MaxUint64

	for _, testCase := range []struct {
		size   uint64
		after  interval.U64RangeList
		before interval.U64RangeList
	}{
		{
			before: interval.U64RangeList{
				interval.U64Range{First: 2, Count: 3},
				interval.U64Range{First: 7, Count: 2},
			},
			after: interval.U64RangeList{
				interval.U64Range{First: 0, Count: 2},
				interval.U64Range{First: 5, Count: 2},
				interval.U64Range{First: 9, Count: heapSize - 9},
			},
		},
		{
			before: interval.U64RangeList{
				interval.U64Range{First: 1, Count: 1},
				interval.U64Range{First: 2, Count: 1},
				interval.U64Range{First: 5, Count: 0}, // Empty ranges get ignored.
			},
			after: interval.U64RangeList{
				interval.U64Range{First: 0, Count: 1},
				interval.U64Range{First: 3, Count: heapSize - 3},
			},
		},
		{
			before: interval.U64RangeList{
				interval.U64Range{First: 2, Count: 1},
				interval.U64Range{First: 3, Count: 2},
				interval.U64Range{First: 7, Count: 2},
			},
			after: interval.U64RangeList{
				interval.U64Range{First: 0, Count: 2},
				interval.U64Range{First: 5, Count: 2},
				interval.U64Range{First: 9, Count: heapSize - 9},
			},
		},
		{
			before: interval.U64RangeList{
				interval.U64Range{First: 0, Count: 1},
				interval.U64Range{First: heapSize - 1, Count: 1},
			},
			after: interval.U64RangeList{
				interval.U64Range{First: 1, Count: heapSize - 2},
			},
		},
		{
			before: interval.U64RangeList{},
			after:  interval.U64RangeList{interval.U64Range{First: 0, Count: heapSize}},
		},
		{
			before: interval.U64RangeList{interval.U64Range{First: 0, Count: heapSize}},
			after:  interval.U64RangeList{},
		},
	} {
		assert.For("InvertMemoryRanges").ThatSlice(InvertMemoryRanges(testCase.before)).Equals(testCase.after)
	}
}

func TestBasicAllocator(t *testing.T) {
	assert := assert.To(t)

	//  0  1  2  3  4  5  6  7  8  9 10 11 12 13 14 15 16 17 18 19 20
	// ##### .. .. .. .. .. .. ## .. .. .. .. .. .. .. .. .. ## .. ..
	initialFreeList := interval.U64RangeList{
		interval.U64Range{First: 2, Count: 6},
		interval.U64Range{First: 9, Count: 9},
		interval.U64Range{First: 19, Count: 2},
	}

	al := NewBasicAllocator(initialFreeList)

	type allocExpectation struct{ expect func(uint64, bool) }
	alloc := func(count uint64, align uint64) allocExpectation {
		result, err := al.Alloc(count, align)
		return allocExpectation{expect: func(addr uint64, ok bool) {
			if ok {
				assert.For("err").ThatError(err).Succeeded()
				assert.For("res").That(result).Equals(addr)
			} else {
				assert.For("err").ThatError(err).Failed()
			}
		}}
	}
	type freeExpectation struct{ expect func(bool) }
	free := func(base uint64) freeExpectation {
		err := al.Free(base)
		return freeExpectation{expect: func(ok bool) {
			if ok {
				assert.For("err").ThatError(err).Succeeded()
			} else {
				assert.For("err").ThatError(err).Failed()
			}
		}}
	}

	alloc(10, 1).expect(0, false) // No contiguous block of size 10.

	alloc(5, 1).expect(2, true)
	assert.For("FreeList").ThatSlice(al.FreeList()).Equals(interval.U64RangeList{
		interval.U64Range{First: 7, Count: 1},
		interval.U64Range{First: 9, Count: 9},
		interval.U64Range{First: 19, Count: 2},
	})
	assert.For("AllocList").ThatSlice(al.AllocList()).Equals(interval.U64RangeList{
		interval.U64Range{First: 2, Count: 5},
	})

	alloc(1, 1).expect(7, true)

	alloc(9, 1).expect(9, true)

	alloc(3, 1).expect(0, false) // Only have two bytes left here.
	alloc(2, 1).expect(19, true)

	// Entire heap is full now.
	alloc(1, 1).expect(0, false)
	assert.For("AllocList").ThatSlice(al.AllocList()).Equals(interval.U64RangeList{
		interval.U64Range{First: 2, Count: 5},
		interval.U64Range{First: 7, Count: 1},
		interval.U64Range{First: 9, Count: 9},
		interval.U64Range{First: 19, Count: 2},
	})
	assert.For("FreeList").ThatSlice(al.FreeList()).Equals(interval.U64RangeList{})

	free(4).expect(false)
	free(7).expect(true)
	free(2).expect(true)
	free(9).expect(true)
	free(9).expect(false) // Can't free twice.
	free(19).expect(true)

	// Test alignment
	alloc(6, 4).expect(12, true)
	assert.For("FreeList").ThatSlice(al.FreeList()).Equals(interval.U64RangeList{
		interval.U64Range{First: 2, Count: 6},
		interval.U64Range{First: 9, Count: 3},
		interval.U64Range{First: 19, Count: 2},
	})
	free(12).expect(true)
	alloc(8, 8).expect(0, false)
	alloc(4, 8).expect(0, false)
	alloc(2, 8).expect(16, true)
	free(16).expect(true)

	// Make sure we've gone back to the initial state of the allocator.
	assert.For("AllocList").ThatSlice(al.AllocList()).Equals(interval.U64RangeList{})
	assert.For("FreeList").ThatSlice(al.FreeList()).Equals(initialFreeList)
}
