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

package main

import (
	"math/rand"
)

type indices []int

func newConsecutiveIndices(n int) indices {
	result := make([]int, n)
	for i := range result {
		result[i] = i
	}
	return indices(result)
}

// randomize returns a new indices slice containing the
// indices contained in the receiver, in random order.
func (l indices) randomize() indices {
	perm := rand.Perm(len(l))
	result := make([]int, len(l))
	for i, v := range perm {
		result[v] = l[i]
	}
	return indices(result)
}

// reverse returns a new indices slice containing the
// indices contained in the receiver, in reverse order.
func (l indices) reverse() indices {
	result := make([]int, len(l))
	for i := range result {
		result[i] = l[len(l)-i-1]
	}
	return indices(result)
}

// take returns a new indices slice containing up to
// the first n indices in the receiver slice.
func (l indices) take(n int) indices {
	if n >= len(l) {
		n = len(l)
	}
	return indices(l[0:n])
}

// filter returns a new indices slice containing the
// indices contained in the receiver which satisfy the predicate.
func (l indices) filter(f func(idx int) bool) indices {
	result := []int{}
	for _, idx := range l {
		if f(idx) {
			result = append(result, idx)
		}
	}
	return indices(result)
}
