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

import (
	"fmt"

	"github.com/google/gapid/core/math/interval"
)

type shrinkOutOfMemBindingBound struct {
	binding memBinding
	offset  uint64
	size    uint64
}

func (e shrinkOutOfMemBindingBound) Error() string {
	return fmt.Sprintf("Out of the bound of memBinding's span: %v, with shrink(offset: %v, size: %v)",
		e.binding.span(), e.offset, e.size)
}

type memBinding interface {
	span() interval.U64Span
	size() uint64
	shrink(offset, size uint64) error
	duplicate() memBinding
}

type memBindingList []memBinding

// Implements the interval.List interface.
func (l memBindingList) Length() int {
	return len(l)
}

func (l memBindingList) GetSpan(index int) interval.U64Span {
	return l[index].span()
}

// Add new memBinding to the memBindingList
func addBinding(l memBindingList, b memBinding) (memBindingList, error) {
	var err error
	first, count := interval.Intersect(l, b.span())
	if count == 0 {
		// no conflict
		i := interval.Search(l, func(span interval.U64Span) bool {
			return span.Start >= b.span().End
		})
		l = append(l[:i], append(memBindingList{b}, l[i:]...)...)
		return l, nil
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
				overlap := sp.End - b.span().Start
				if err = l[i].shrink(0, l[i].size()-overlap); err != nil {
					return nil, err
				}
				i++

			} else if sp.Start >= b.span().Start &&
				sp.End > b.span().End &&
				sp.Start < b.span().End {
				// truncate the head of sp
				overlap := b.span().End - sp.Start
				if err = l[i].shrink(overlap, l[i].size()-overlap); err != nil {
					return nil, err
				}
				i++

			} else if sp.Start < b.span().Start &&
				sp.End > b.span().End {
				// split sp
				newB := l[i].duplicate()
				if err = newB.shrink(b.span().End-sp.Start, newB.size()-b.span().End+sp.Start); err != nil {
					return nil, err
				}
				if err = l[i].shrink(0, sp.End-b.span().Start); err != nil {
					return nil, err
				}
				l, err = addBinding(l, newB)
				if err != nil {
					return nil, err
				}
				// Should not have any other intersects
				break

			} else if sp.Start >= b.span().Start &&
				sp.End <= b.span().End {
				// remove sp, no need to i++
				l = append(l[:i], l[i+1:]...)
				count--
			}
		}
		l, err = addBinding(l, b)
		if err != nil {
			return nil, err
		}
	}
	return l, nil
}
