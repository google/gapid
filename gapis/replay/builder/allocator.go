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

package builder

type allocator struct {
	alignment uint64
	size      uint64
	head      uint64
}

func max(a, b uint64) uint64 {
	if a > b {
		return a
	} else {
		return b
	}
}

func align(val, by uint64) uint64 {
	return ((val + by - 1) / by) * by
}

func (a *allocator) alloc(size uint64) uint64 {
	ptr := align(a.head, a.alignment)
	a.head = ptr + size
	a.size = max(a.size, a.head)
	return ptr
}

func (a *allocator) reset() {
	a.head = 0
}
