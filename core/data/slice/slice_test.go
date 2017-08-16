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

package slice_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/slice"
	"github.com/google/gapid/core/log"
)

func TestReplace(t *testing.T) {
	ctx := log.Testing(t)

	for _, test := range []struct {
		data     []int
		at       int
		count    int
		with     interface{}
		expected []int
	}{
		// Single element substitution (with: slice).
		{[]int{1, 2, 3}, 0, 1, []int{9}, []int{9, 2, 3}},
		{[]int{1, 2, 3}, 1, 1, []int{9}, []int{1, 9, 3}},
		{[]int{1, 2, 3}, 2, 1, []int{9}, []int{1, 2, 9}},

		// Single element substitution (with: element).
		{[]int{1, 2, 3}, 0, 1, 9, []int{9, 2, 3}},
		{[]int{1, 2, 3}, 1, 1, 9, []int{1, 9, 3}},
		{[]int{1, 2, 3}, 2, 1, 9, []int{1, 2, 9}},

		// Insertion.
		{[]int{1, 2, 3}, 0, 0, []int{9}, []int{9, 1, 2, 3}},
		{[]int{1, 2, 3}, 1, 0, []int{9}, []int{1, 9, 2, 3}},
		{[]int{1, 2, 3}, 2, 0, []int{9}, []int{1, 2, 9, 3}},

		// Reduction.
		{[]int{1, 2, 3}, 0, 2, []int{9}, []int{9, 3}},
		{[]int{1, 2, 3}, 1, 2, []int{9}, []int{1, 9}},
	} {
		got := make([]int, len(test.data))
		copy(got, test.data)
		slice.Replace(&got, test.at, test.count, test.with)
		assert.For(ctx, "Replace(%v, %v, %v, %v)", test.data, test.at, test.count, test.with).
			That(got).DeepEquals(test.expected)
	}
}

func TestRemove(t *testing.T) {
	ctx := log.Testing(t)

	for _, test := range []struct {
		data     []int
		val      interface{}
		expected []int
	}{
		// No replacement.
		{[]int{1, 2, 3}, 9, []int{1, 2, 3}},
		{[]int{1, 2, 3}, 9, []int{1, 2, 3}},
		{[]int{1, 2, 3}, 9, []int{1, 2, 3}},

		// Single replacements.
		{[]int{1, 2, 3}, 1, []int{2, 3}},
		{[]int{1, 2, 3}, 2, []int{1, 3}},
		{[]int{1, 2, 3}, 3, []int{1, 2}},
	} {
		got := make([]int, len(test.data))
		copy(got, test.data)
		slice.Remove(&got, test.val)
		assert.For(ctx, "Remove(%v, %v)", test.data, test.val).
			That(got).DeepEquals(test.expected)
	}
}

func TestRemoveAt(t *testing.T) {
	ctx := log.Testing(t)

	for _, test := range []struct {
		data     []int
		i, n     int
		expected []int
	}{
		// No removal.
		{[]int{1, 2, 3}, 0, 0, []int{1, 2, 3}},
		{[]int{1, 2, 3}, 1, 0, []int{1, 2, 3}},
		{[]int{1, 2, 3}, 2, 0, []int{1, 2, 3}},

		// Single removal.
		{[]int{1, 2, 3}, 0, 1, []int{2, 3}},
		{[]int{1, 2, 3}, 1, 1, []int{1, 3}},
		{[]int{1, 2, 3}, 2, 1, []int{1, 2}},

		// Double removal.
		{[]int{1, 2, 3}, 0, 2, []int{3}},
		{[]int{1, 2, 3}, 1, 2, []int{1}},

		// All.
		{[]int{1, 2, 3}, 0, 3, []int{}},
	} {
		got := make([]int, len(test.data))
		copy(got, test.data)
		slice.RemoveAt(&got, test.i, test.n)
		assert.For(ctx, "RemoveAt(%v, %v, %v)", test.data, test.i, test.n).
			That(got).DeepEquals(test.expected)
	}
}

func TestInsertBefore(t *testing.T) {
	ctx := log.Testing(t)

	for _, test := range []struct {
		data     []int
		at       int
		val      interface{}
		expected []int
	}{
		// Insertion (with: slice).
		{[]int{1, 2, 3}, 0, []int{9}, []int{9, 1, 2, 3}},
		{[]int{1, 2, 3}, 1, []int{9}, []int{1, 9, 2, 3}},
		{[]int{1, 2, 3}, 2, []int{9}, []int{1, 2, 9, 3}},

		// Insertion (with: element).
		{[]int{1, 2, 3}, 0, 9, []int{9, 1, 2, 3}},
		{[]int{1, 2, 3}, 1, 9, []int{1, 9, 2, 3}},
		{[]int{1, 2, 3}, 2, 9, []int{1, 2, 9, 3}},
	} {
		got := make([]int, len(test.data))
		copy(got, test.data)
		slice.InsertBefore(&got, test.at, test.val)
		assert.For(ctx, "InsertBefore(%v, %v, %v)", test.data, test.at, test.val).
			That(got).DeepEquals(test.expected)
	}
}

func TestAppend(t *testing.T) {
	ctx := log.Testing(t)

	for _, test := range []struct {
		data     []int
		val      interface{}
		expected []int
	}{
		// Insertion (with: slice).
		{[]int{1, 2, 3}, []int{7, 8, 9}, []int{1, 2, 3, 7, 8, 9}},
		{[]int{1, 2, 3}, []int{9}, []int{1, 2, 3, 9}},

		// Insertion (with: element).
		{[]int{1, 2, 3}, 9, []int{1, 2, 3, 9}},
	} {
		got := make([]int, len(test.data))
		copy(got, test.data)
		slice.Append(&got, test.val)
		assert.For(ctx, "Append(%v, %v)", test.data, test.val).
			That(got).DeepEquals(test.expected)
	}
}

func TestReverse(t *testing.T) {
	ctx := log.Testing(t)

	for _, test := range []struct {
		data     []int
		expected []int
	}{
		{[]int{}, []int{}},
		{[]int{1}, []int{1}},
		{[]int{1, 2}, []int{2, 1}},
		{[]int{1, 2, 3}, []int{3, 2, 1}},
		{[]int{1, 2, 3, 4}, []int{4, 3, 2, 1}},
	} {
		data := slice.Clone(test.data)
		slice.Reverse(data)
		assert.For(ctx, "Reverse(%v)", test.data).That(data).DeepEquals(test.expected)
	}
}
