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

package atom

import (
	"fmt"
	"strings"

	"github.com/google/gapid/core/math/interval"
)

// GroupList is a list of Groups. Functions in this package expect the list to
// be in ascending atom index order, and maintain that order on mutation.
type GroupList []Group

func (l GroupList) info(depth int) string {
	parts := make([]string, len(l))
	for i, group := range l {
		parts[i] = fmt.Sprintf("%s%.5d: %s",
			strings.Repeat("  ", depth), i, group.info(depth))
	}
	return strings.Join(parts, "\n")
}

// String returns a string representing all groups in the group list.
func (l GroupList) String() string {
	return l.info(0)
}

// IndexOf returns the index of the group that contains the atom index or
// -1 if not found.
func (l *GroupList) IndexOf(atomIndex uint64) int {
	return interval.IndexOf(l, atomIndex)
}

// Add inserts a new atom group into the list with the specified range and name.
// If the new group does not overlap any existing groups in the list then it is
// inserted into the list, keeping ascending atom-identifier order.
// If the new group sits completely within an existing group then this new group
// will be added to the existing group's sub-groups.
// If the new group completely wraps one or more existing groups in the list
// then these existing groups are added as sub-groups to the new group and then
// the new group is added to the list, keeping ascending atom-identifier order.
// If the new group partially overlaps any existing group then the function will
// return an error.
func (l *GroupList) Add(start, end uint64, name string) error {
	r := Range{Start: start, End: end}
	g := Group{Name: name, Range: r}
	s, c := interval.Intersect(l, r.Span())
	if c == 0 {
		// No overlaps, clean insertion
		i := interval.Merge(l, g.Range.Span(), false)
		(*l)[i].Name = g.Name
		(*l)[i].SubGroups = g.SubGroups
	} else {
		// At least one overlap
		first := (*l)[s]
		last := (*l)[s+c-1]
		sIn, eIn := first.Range.Contains(start), last.Range.Contains(end-1)
		switch {
		case c == 1 && sIn && eIn:
			// New group fits entirely within an existing group. Add as subgroup.
			first.SubGroups.Add(start, end, name)
			(*l)[s] = first
		case sIn && start != first.Range.Start:
			return fmt.Errorf("New group '%s' overlaps with existing group '%s'", g, first)
		case eIn && end != last.Range.End:
			return fmt.Errorf("New group '%s' overlaps with existing group '%s'", g, last)
		default:
			// New group completely wraps one or more existing groups. Add the
			// existing group(s) as subgroups to the new group, and add to the list.
			g.SubGroups = append(g.SubGroups, (*l)[s:s+c]...)
			i := interval.Merge(l, g.Range.Span(), false)
			(*l)[i].Name = g.Name
			(*l)[i].SubGroups = g.SubGroups
		}
	}
	return nil
}

// Length returns the number of groups in the list.
func (l GroupList) Length() int {
	return len(l)
}

// GetSpan returns the atom index span for the group at index in the list.
func (l GroupList) GetSpan(index int) interval.U64Span {
	return l[index].Range.Span()
}

// SetSpan sets the atom index span for the group at index in the list.
func (l GroupList) SetSpan(index int, span interval.U64Span) {
	l[index].Range.SetSpan(span)
}

// New sets the atom index span for the group at index in the list.
func (l GroupList) New(index int, span interval.U64Span) {
	l[index].Range.SetSpan(span)
}

// Copy copies count groups within the list.
func (l GroupList) Copy(to, from, count int) {
	copy(l[to:to+count], l[from:from+count])
}

// Resize adjusts the length of the list.
func (l *GroupList) Resize(length int) {
	if cap(*l) > length {
		*l = (*l)[:length]
	} else {
		old := *l
		capacity := cap(*l) * 2
		if capacity < length {
			capacity = length
		}
		*l = make(GroupList, length, capacity)
		copy(*l, old)
	}
}
