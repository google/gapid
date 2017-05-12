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
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/google/gapid/core/data/slice"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/core/math/sint"
)

// Group represents a named group of atoms with support for sparse sub-groups
// and sub-atom-ranges.
// Groups are ideal for expressing nested hierarchies of atoms.
//
// Groups have the concept of items. An item is either an immediate sub-group,
// or an atom range that is within this group's span but outside of any
// sub-group.
type Group struct {
	Name  string // Name of this group.
	Range Range  // The range of atoms this group (and items) represents.
	Spans Spans  // All sub-groups and sub-ranges of this group.
}

// Spans is a list of Span elements. Functions in this package expect the
// list to be in ascending atom index order, and maintain that order on
// mutation.
type Spans []Span

// IndexOf returns the index of the group that contains the atom index or
// -1 if not found.
func (l *Spans) IndexOf(atomIndex uint64) int {
	return interval.IndexOf(l, atomIndex)
}

// Length returns the number of groups in the list.
func (l Spans) Length() int {
	return len(l)
}

// GetSpan returns the atom index span for the group at index in the list.
func (l Spans) GetSpan(index int) interval.U64Span {
	return l[index].Bounds().Span()
}

// Span is a child of a Group. It is implemented by Group and Range.
type Span interface {
	// Bounds returns the absolute range of atom indices for the span.
	Bounds() Range

	// itemCount returns the number of items this span represents to its parent.
	// For a Range, this is the interval length.
	// For a Group, this is always 1.
	itemCount() uint64

	// item returns the i'th sub-item for this span.
	// For a Range, this is the i'th ID in the interval.
	// For a Group, this is always the group itself.
	item(i uint64) GroupOrID

	// itemIndex returns the item sub-index for the given ID.
	// For a Range, this is i minus the first ID in the interval.
	// For a Group, this is always 0.
	itemIndex(i ID) uint64

	// split returns two spans over the same range as this span, but where the
	// first contains the given number of items and the second the rest.
	split(i uint64) (Span, Span)
}

// GroupOrID is a dummy interface exclusively implemented by Group and ID.
type GroupOrID interface {
	isGroupOrID()
}

func (Group) isGroupOrID() {}
func (ID) isGroupOrID()    {}

func (r Range) Bounds() Range               { return r }
func (r Range) itemCount() uint64           { return r.Length() }
func (r Range) item(i uint64) GroupOrID     { return r.Start + ID(i) }
func (r Range) itemIndex(i ID) uint64       { return uint64(i - r.Start) }
func (r Range) split(i uint64) (Span, Span) { return r.Split(i) }

func (g Group) Bounds() Range               { return g.Range }
func (g Group) itemCount() uint64           { return 1 }
func (g Group) item(uint64) GroupOrID       { return g }
func (g Group) itemIndex(i ID) uint64       { return 0 }
func (g Group) split(i uint64) (Span, Span) { return g, nil }

// Format writes a string representing the group's name, range and sub-groups.
func (g Group) Format(f fmt.State, r rune) {
	align := 12
	pad := strings.Repeat(" ", sint.Max(align+2, 0))

	buf := bytes.Buffer{}
	buf.WriteString("Group '")
	buf.WriteString(g.Name)
	buf.WriteString("' ")
	buf.WriteString(g.Range.String())

	if f.Flag('+') && r == 'v' {
		idx := uint64(0)
		for i, span := range g.Spans {
			var idxSpan string
			itemCount := span.itemCount()
			if itemCount <= 1 {
				idxSpan = fmt.Sprintf("[%d]", idx)
			} else {
				idxSpan = fmt.Sprintf("[%d..%d]", idx, idx+itemCount-1)
			}
			idx += itemCount

			nl := "\n"
			if i < len(g.Spans)-1 {
				buf.WriteString("\n ├─ ")
				nl += " │ " + pad
			} else {
				buf.WriteString("\n └─ ")
				nl += "   " + pad
			}
			buf.WriteString(idxSpan)
			buf.WriteRune(' ')
			buf.WriteString(strings.Repeat("─", sint.Max(align-len(idxSpan), 0)))
			buf.WriteRune(' ')
			switch span.(type) {
			case Range:
				buf.WriteString("Atoms ")
			}
			buf.WriteString(strings.Replace(fmt.Sprintf("%+v", span), "\n", nl, -1))
		}
	}

	f.Write(buf.Bytes())
}

// Count returns the number of immediate items this group contains.
func (g Group) Count() uint64 {
	var count uint64
	for _, s := range g.Spans {
		count += s.itemCount()
	}
	return count
}

// DeepCount returns the total (recursive) number of items this group contains.
// The given predicate determines wheter the tested group is counted as 1 or
// is recursed into.
func (g Group) DeepCount(pred func(g Group) bool) uint64 {
	var count uint64
	for _, s := range g.Spans {
		switch s := s.(type) {
		case Group:
			if pred(s) {
				count += s.DeepCount(pred)
			} else {
				count += 1
			}
		default:
			count += s.itemCount()
		}
	}
	return count
}

// Index returns the item at the specified index.
func (g Group) Index(index uint64) GroupOrID {
	for _, s := range g.Spans {
		c := s.itemCount()
		if index < c {
			return s.item(index)
		}
		index -= c
	}
	return nil // Out of range.
}

// IndexOf returns the item index that id refers directly to, or contains id.
func (g Group) IndexOf(id ID) uint64 {
	idx := uint64(0)
	for _, s := range g.Spans {
		if s.Bounds().Contains(id) {
			return idx + s.itemIndex(id)
		}
		idx += s.itemCount()
	}
	return 0
}

// IterateForwards calls cb with each contained atom index or group starting
// with the item at index. If cb returns an error then traversal is stopped and
// the error is returned.
func (g Group) IterateForwards(index uint64, cb func(childIdx uint64, item GroupOrID) error) error {
	childIndex := uint64(0)
	visit := func(item GroupOrID) error {
		idx := childIndex
		childIndex++
		if idx < index {
			return nil
		}
		return cb(idx, item)
	}

	for _, s := range g.Spans {
		for i, c := uint64(0), s.itemCount(); i < c; i++ {
			if err := visit(s.item(i)); err != nil {
				return err
			}
		}
	}
	return nil
}

// IterateBackwards calls cb with each contained atom index or group starting
// with the item at index. If cb returns an error then traversal is stopped and
// the error is returned.
func (g Group) IterateBackwards(index uint64, cb func(childIdx uint64, item GroupOrID) error) error {
	childIndex := g.Count() - 1
	visit := func(item GroupOrID) error {
		idx := childIndex
		childIndex--
		if idx > index {
			return nil
		}
		return cb(idx, item)
	}

	for i := range g.Spans {
		s := g.Spans[len(g.Spans)-i-1]
		for i, c := uint64(0), s.itemCount(); i < c; i++ {
			if err := visit(s.item(c - i - 1)); err != nil {
				return err
			}
		}
	}
	return nil
}

// AddGroup inserts a new sub-group with the specified range and name.
//
// If the new group does not overlap any existing groups in the list then it is
// inserted into the list, keeping ascending atom-identifier order.
// If the new group sits completely within an existing group then this new group
// will be added to the existing group's sub-groups.
// If the new group completely wraps one or more existing groups in the list
// then these existing groups are added as sub-groups to the new group and then
// the new group is added to the list, keeping ascending atom-identifier order.
// If the new group partially overlaps any existing group then the function will
// return an error.
//
// *** Warning ***
// All groups must be added before atoms.
// Attemping to call this function after atoms have been added may result in
// panics!
func (g *Group) AddGroup(start, end ID, name string) error {
	r := Range{Start: start, End: end}
	s, c := interval.Intersect(&g.Spans, r.Span())
	if c == 0 {
		// No overlaps, clean insertion
		i := sort.Search(len(g.Spans), func(i int) bool {
			return g.Spans[i].Bounds().Start > start
		})
		slice.InsertBefore(&g.Spans, i, Group{Name: name, Range: r})
	} else {
		// At least one overlap
		first := g.Spans[s].(Group)
		last := g.Spans[s+c-1].(Group)
		sIn, eIn := first.Bounds().Contains(start), last.Bounds().Contains(end-1)
		switch {
		case c == 1 && sIn && eIn:
			// New group fits entirely within an existing group. Add as subgroup.
			first.AddGroup(start, end, name)
			g.Spans[s] = first
		case sIn && start != first.Range.Start:
			return fmt.Errorf("New group '%s' overlaps with existing group '%s'", name, first)
		case eIn && end != last.Range.End:
			return fmt.Errorf("New group '%s' overlaps with existing group '%s'", name, last)
		default:
			// New group completely wraps one or more existing groups. Add the
			// existing group(s) as subgroups to the new group, and add to the list.
			n := Group{Name: name, Range: r, Spans: make(Spans, c)}
			copy(n.Spans, g.Spans[s:s+c])
			slice.Replace(&g.Spans, s, c, n)
		}
	}
	return nil
}

// AddAtoms fills the group and sub-groups with atoms based on the predicate
// pred. If maxChildren is positive, the group, and any of it's decendent
// groups, which have more than maxChildren child elements, will have their
// children grouped into new synthetic groups of at most maxChildren children.
func (g *Group) AddAtoms(pred func(id ID) bool, maxChildren uint64) error {
	rng := g.Range
	spans := make(Spans, 0, len(g.Spans))

	scan := func(to ID) {
		for id := rng.Start; id < to; id++ {
			if !pred(id) {
				rng.End = id
				if rng.Start != rng.End {
					spans = append(spans, rng)
				}
				rng.Start = id + 1
			}
		}
		if rng.Start != to {
			rng.End = to
			spans = append(spans, rng)
			rng.Start = to
		}
	}

	for _, s := range g.Spans {
		switch s := s.(type) {
		case Group:
			scan(s.Bounds().Start)
			s.AddAtoms(pred, maxChildren)
			rng.Start = s.Bounds().End
			spans = append(spans, s)
		}
	}
	scan(g.Range.End)

	g.Spans = spans

	if maxChildren > 0 && g.Count() > maxChildren {
		g.Spans = g.Spans.split(maxChildren)
	}

	return nil
}

// split returns a new list of spans where each new span will represent no more
// than the given number of items.
func (l Spans) split(max uint64) Spans {
	out, current, idx, count := make([]Span, 0), make([]Span, 0), 1, uint64(0)
outer:
	for _, span := range l {
		space := max - count
		for space < span.itemCount() {
			head, tail := Span(nil), span
			if space > 0 {
				head, tail = span.split(space)
				current = append(current, head)
			}
			out = append(out, Group{
				fmt.Sprintf("Sub Group %d", idx),
				Range{current[0].Bounds().Start, current[len(current)-1].Bounds().End},
				current,
			})
			current, idx, count, space, span = nil, idx+1, 0, max, tail
			if span == nil {
				continue outer
			}
		}
		current = append(current, span)
		count += span.itemCount()
	}

	if len(current) > 0 {
		out = append(out, Group{
			fmt.Sprintf("Sub Group %d", idx),
			Range{current[0].Bounds().Start, current[len(current)-1].Bounds().End},
			current,
		})
	}
	return out
}

// TraverseCallback is the function that's called for each traversed item in a
// group.
type TraverseCallback func(indices []uint64, item GroupOrID) error

// Traverse traverses the atom group starting with the specified index,
// calling cb for each encountered node.
func (g Group) Traverse(backwards bool, start []uint64, cb TraverseCallback) error {
	t := groupTraverser{backwards: backwards, cb: cb}

	// Make a copy of start as traversal alters the slice.
	indices := make([]uint64, len(start))
	copy(indices, start)

	groups := make([]Group, 1, len(indices)+1)
	groups[0] = g
	for i := range indices {
		item := groups[i].Index(indices[i])
		g, ok := item.(Group)
		if !ok {
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
				err = cb(t.indices, g)
			} else {
				err = g.IterateForwards(indices[i], t.visit)
			}
		default:
			// Group is not the deepest.
			// Search after / before the index that passes through this group.
			if backwards {
				if indices[i] > 0 {
					if err := g.IterateBackwards(indices[i]-1, t.visit); err != nil {
						return err
					}
				}
				err = cb(t.indices, g)
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
	cb        TraverseCallback
	indices   []uint64
}

func (s *groupTraverser) visit(childIdx uint64, item GroupOrID) error {
	if !s.backwards {
		if err := s.cb(append(s.indices, childIdx), item); err != nil {
			return err
		}
	}
	if g, ok := item.(Group); ok {
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
		if err := s.cb(append(s.indices, childIdx), item); err != nil {
			return err
		}
	}
	return nil
}
