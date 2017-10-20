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

package vulkan

import "github.com/google/gapid/core/math/interval"

type sparseBindingList []VkSparseMemoryBind

// Implements the interval.List interface.
func (l sparseBindingList) Length() int {
	return len(l)
}

func (l sparseBindingList) GetSpan(index int) interval.U64Span {
	return l[index].span()
}

func (b VkSparseMemoryBind) span() interval.U64Span {
	return interval.U64Span{Start: uint64(b.ResourceOffset), End: uint64(b.ResourceOffset) + uint64(b.Size)}
}

func addBinding(l sparseBindingList, b VkSparseMemoryBind) sparseBindingList {
	first, count := interval.Intersect(l, b.span())
	if count == 0 {
		// no conflict
		i := interval.Search(l, func(span interval.U64Span) bool {
			return span.Start >= b.span().End
		})
		l = append(l[:i], append(sparseBindingList{b}, l[i:]...)...)
	} else {
		// has conflits, truncate the existing spans to remove conflict, then add
		// the incoming bind again as if there is no conflict. Note that it is
		// guaranteed that there is no conflict among the existing spans
		i := first
		for i < first+count {
			sp := l.GetSpan(i)
			if sp.Start < b.span().Start &&
				sp.End <= b.span().End &&
				sp.End > b.span().Start {
				// truncate the tail of sp
				overlap := VkDeviceSize(sp.End - b.span().Start)
				l[i].Size = l[i].Size - VkDeviceSize(overlap)
				i++

			} else if sp.Start >= b.span().Start &&
				sp.End > b.span().End &&
				sp.Start < b.span().End {
				// truncate the head of sp
				overlap := VkDeviceSize(b.span().End - sp.Start)
				l[i].Size = l[i].Size - VkDeviceSize(overlap)
				l[i].MemoryOffset += overlap
				l[i].ResourceOffset += overlap
				i++

			} else if sp.Start < b.span().Start &&
				sp.End > b.span().End {
				// split sp
				newB := l[i]
				newB.MemoryOffset += VkDeviceSize(b.span().End - sp.Start)
				newB.ResourceOffset += VkDeviceSize(b.span().End - sp.Start)
				newB.Size -= VkDeviceSize(b.span().End - sp.Start)

				l[i].Size -= VkDeviceSize(sp.End - b.span().Start)
				l = addBinding(l, newB)
				// Should not have any other intersects
				break

			} else if sp.Start >= b.span().Start &&
				sp.End <= b.span().End {
				// remove sp, no need to i++
				l = append(l[:i], l[i+1:]...)
				count--
			}
		}
		l = addBinding(l, b)
	}
	return l
}
