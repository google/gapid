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
	"github.com/google/gapid/framework/binary"
)

// Group represents a named, contiguous span of atoms with support for sparse
// sub-groups. Groups are ideal for expressing nested hierarchies of atoms.
//
// Groups have the concept of items. An item is either an immediate sub-group,
// or an atom index that is within this group's span but outside of any
// sub-group.
//
// For example a Group spanning the atom index range [0 - 9] with two
// sub-groups spanning [2 - 4] and [7 - 8] would have the following tree of
// items:
//
//  Group
//    │
//    ├─── Item[0] ─── Atom[0]
//    │
//    ├─── Item[1] ─── Atom[1]
//    │
//    ├─── Item[2] ─── Sub-group 0
//    │                   │
//    │                   ├─── Item[0] ─── Atom[2]
//    │                   │
//    │                   ├─── Item[1] ─── Atom[3]
//    │                   │
//    │                   └─── Item[2] ─── Atom[4]
//    │
//    ├─── Item[3] ─── Atom[5]
//    │
//    ├─── Item[4] ─── Atom[6]
//    │
//    ├─── Item[5] ─── Sub-group 1
//    │                   │
//    │                   ├─── Item[0] ─── Atom[7]
//    │                   │
//    │                   └─── Item[1] ─── Atom[8]
//    │
//    └─── Item[6] ─── Atom[9]
//
type Group struct {
	binary.Generate `java:"AtomGroup"`
	Name            string    // Name of this group.
	Range           Range     // The range of atoms this group (and sub-groups) represents.
	SubGroups       GroupList // All sub-groups of this group.
}

func (g Group) info(depth int) string {
	str := fmt.Sprintf("%s%s %s",
		strings.Repeat("  ", depth), g.Range.String(), g.Name)
	if len(g.SubGroups) > 0 {
		str += "\n" + g.SubGroups.info(depth+1)
	}
	return str
}

// String returns a string representing the group's name, range and sub-groups.
func (g Group) String() string {
	return g.info(0)
}

// Count returns the number of immediate items this group contains.
func (g Group) Count() uint64 {
	count := g.Range.Length()
	for _, sg := range g.SubGroups {
		count -= sg.Range.Length()
		count++ // For the group itself
	}
	return count
}

// Index returns the item at the specified index. If the item refers directly
// to an atom index then the atom index is returned in baseAtomIndex and
// subgroup is assigned nil.
// If the item is a sub-group then baseAtomIndex is returned as the lowest atom
// identifier found in the sub-group and subgroup is assigned the sub-group
// pointer.
func (g Group) Index(index uint64) (baseIndex uint64, subgroup *Group) {
	base := g.Range.First()
	for i := range g.SubGroups {
		sg := &g.SubGroups[i]
		if base+index < sg.Range.First() {
			break
		}
		index -= uint64(sg.Range.First() - base)
		if index == 0 {
			return sg.Range.First(), sg
		}
		index--
		base = sg.Range.Last() + 1
	}
	return base + index, nil
}

// IndexOf returns the item index that atomIndex refers directly to, or contains the
// given atom index.
func (g Group) IndexOf(atomIndex uint64) uint64 {
	index := uint64(0)
	base := g.Range.First()
	for _, sg := range g.SubGroups {
		if atomIndex < sg.Range.First() {
			break
		}
		index += uint64(sg.Range.First() - base)
		base = sg.Range.Last() + 1
		if atomIndex <= sg.Range.Last() {
			return index
		}
		index++
	}
	return index + (atomIndex - base)
}

// Insert adjusts the spans of this group and all subgroups for an insertion
// of count elements at atomIndex.
func (g *Group) Insert(atomIndex uint64, count int) {
	s, e := g.Range.Range()
	if s >= atomIndex {
		s += uint64(count)
	}
	if e > atomIndex {
		e += uint64(count)
	}
	g.Range = Range{Start: s, End: e}
	i := interval.Search(&g.SubGroups, func(test interval.U64Span) bool {
		return atomIndex < test.End
	})
	for i < len(g.SubGroups) {
		sg := g.SubGroups[i]
		sg.Insert(atomIndex, count)
		g.SubGroups[i] = sg
		i++
	}
}
