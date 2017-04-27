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
	Name      string    // Name of this group.
	Range     Range     // The range of atoms this group (and sub-groups) represents.
	SubGroups GroupList // All sub-groups of this group.
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
// to an atom index then the atom index is returned in atomIndex and subgroup
// is assigned nil.
// If the item is a sub-group then atomIndex is returned as the lowest atom
// identifier found in the sub-group and subgroup is assigned the sub-group
// pointer.
func (g Group) Index(index uint64) (atomIndex uint64, subgroup *Group) {
	base := g.Range.Start // base atom index
	for i := range g.SubGroups {
		sg := &g.SubGroups[i]
		if base+index < sg.Range.Start {
			break
		}
		index -= uint64(sg.Range.Start - base)
		if index == 0 {
			return sg.Range.Start, sg
		}
		index--
		base = sg.Range.End
	}
	return base + index, nil
}

// IterateForwards calls cb with each contained atom index or group starting
// with the item at index. If cb returns an error then traversal is stopped and
// the error is returned.
func (g *Group) IterateForwards(index uint64, cb func(childIdx, atomIndex uint64, subgroup *Group) error) error {
	childIndex := uint64(0)
	visit := func(atomIndex uint64, subgroup *Group) error {
		idx := childIndex
		childIndex++
		if idx < index {
			return nil
		}
		return cb(idx, atomIndex, subgroup)
	}

	base := g.Range.Start // base atom index
	for i := range g.SubGroups {
		sg := &g.SubGroups[i]
		for i, e := base, sg.Range.Start; i < e; i++ {
			if err := visit(i, nil); err != nil {
				return err
			}
		}
		if err := visit(sg.Range.Start, sg); err != nil {
			return err
		}
		base = sg.Range.End
	}
	for i, e := base, g.Range.End; i < e; i++ {
		if err := visit(i, nil); err != nil {
			return err
		}
	}
	return nil
}

// IterateBackwards calls cb with each contained atom index or group starting
// with the item at index. If cb returns an error then traversal is stopped and
// the error is returned.
func (g *Group) IterateBackwards(index uint64, cb func(childIdx, atomIndex uint64, subgroup *Group) error) error {
	childIndex := g.Count() - 1
	visit := func(atomIndex uint64, subgroup *Group) error {
		idx := childIndex
		childIndex--
		if idx > index {
			return nil
		}
		return cb(idx, atomIndex, subgroup)
	}

	base := g.Range.End // base atom index
	for i := range g.SubGroups {
		sg := &g.SubGroups[len(g.SubGroups)-i-1]
		for i, e := base, sg.Range.End; i > e; i-- {
			if err := visit(i-1, nil); err != nil {
				return err
			}
		}
		if err := visit(sg.Range.Start, sg); err != nil {
			return err
		}
		base = sg.Range.Start
	}
	for i, e := base, g.Range.Start; i > e; i-- {
		if err := visit(i-1, nil); err != nil {
			return err
		}
	}
	return nil
}

// IndexOf returns the item index that atomIndex refers directly to, or contains the
// given atom index.
func (g Group) IndexOf(atomIndex uint64) uint64 {
	index := uint64(0)
	base := g.Range.Start
	for _, sg := range g.SubGroups {
		if atomIndex < sg.Range.Start {
			break
		}
		index += uint64(sg.Range.Start - base)
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

// TraverseCallback is the function that's called for each traversed item in a
// group.
type TraverseCallback func(indices []uint64, atomIdx uint64, group *Group) error

// Traverse traverses the atom group starting with the specified index,
// calling cb for each encountered node.
func (g *Group) Traverse(backwards bool, start []uint64, cb TraverseCallback) error {
	t := groupTraverser{backwards: backwards, cb: cb}

	indices := start
	groups := make([]*Group, 1, len(indices)+1)
	groups[0] = g
	for i := range indices {
		_, g := groups[i].Index(indices[i])
		if g == nil {
			break
		}
		groups = append(groups, g)
	}

	// Examples of groups / indices:
	//
	// groups[0]       | groups[0]       | groups[0]       |  groups[0]
	//      indices[0] |      indices[0] |      indices[0] |
	// groups[1]       | groups[1]       |                 |
	//      indices[1] |      indices[1] |                 |
	// groups[2]       | -               |                 |
	//      indices[2] |      indices[2] |                 |
	// groups[3]       | -               |                 |

	for i := len(groups) - 1; i >= 0; i-- {
		g := groups[i]
		t.indices = indices[:i]
		var err error
		switch {
		case i >= len(indices):
			// Group doesn't have an index specifiying a child to search from.
			// Search the entire group.
			if backwards {
				err = g.IterateBackwards(g.Count()-1, t.visit)
			} else {
				err = g.IterateForwards(0, t.visit)
			}
		case i == len(groups)-1:
			// Group is the deepest.
			// Search from index that passes through this group.
			if backwards {
				if err := g.IterateBackwards(indices[i], t.visit); err != nil {
					return err
				}
				err = cb(t.indices, g.Range.Start, g)
			} else {
				err = g.IterateForwards(indices[i], t.visit)
			}
		default:
			// Group is not the deepest.
			// Search after / before the index that passes through this group.
			if backwards {
				err = g.IterateBackwards(indices[i]-1, t.visit)
			} else {
				err = g.IterateForwards(indices[i]+1, t.visit)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

type groupTraverser struct {
	backwards bool
	cb        func([]uint64, uint64, *Group) error
	indices   []uint64
}

func (s *groupTraverser) visit(childIdx, atomIdx uint64, g *Group) error {
	if !s.backwards {
		if err := s.cb(append(s.indices, childIdx), atomIdx, g); err != nil {
			return err
		}
	}
	if g != nil {
		s.indices = append(s.indices, childIdx)
		var err error
		if s.backwards {
			err = g.IterateBackwards(g.Count()-1, s.visit)
		} else {
			err = g.IterateForwards(0, s.visit)
		}
		s.indices = s.indices[:len(s.indices)-1]
		if err != nil {
			return err
		}
	}
	if s.backwards {
		if err := s.cb(append(s.indices, childIdx), atomIdx, g); err != nil {
			return err
		}
	}
	return nil
}
