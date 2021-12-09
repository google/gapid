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

package api

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/google/gapid/core/data/slice"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/core/math/sint"
)

// CmdIDGroup represents a named group of commands with support for sparse
// sub-groups and sub-command-ranges.
// Groups are ideal for expressing nested hierarchies of commands.
//
// Groups have the concept of items. An item is either an immediate sub-group,
// or a command range that is within this group's span but outside of any
// sub-group.
type CmdIDGroup struct {
	Name               string      // Name of this group.
	Range              CmdIDRange  // The range of commands this group (and items) represents.
	Spans              Spans       // All sub-groups and sub-ranges of this group.
	ExperimentableCmds []SubCmdIdx // Indices of commands under this group that can be disabled for experiments.
	UserData           interface{}
}

// SubCmdRoot is a new namespace under which subcommands live.
type SubCmdRoot struct {
	Id       SubCmdIdx  // The fully qualified index of the node
	SubGroup CmdIDGroup // The range of subcommands in this range
}

// CmdGroupOrRoot represents either a named group of commands, or a
// new SubCmdRoot under which new commands live.
type CmdGroupOrRoot interface {
	SpanItem
	// Index returns the child at the given index. This can either be another
	// group, CmdId or root.
	Index(index uint64) SpanItem
}

// NewRoot sets up a new root object.
func NewRoot(idx []uint64, nameLookUp *SubCmdIdxTrie) *SubCmdRoot {
	var name = "Subgroup"
	if nameVal := nameLookUp.Value(idx); nameVal != nil {
		if n, ok := nameVal.(string); ok {
			name = n
		}
	}
	return &SubCmdRoot{Id: append(slice.Clone(idx).([]uint64)),
		SubGroup: CmdIDGroup{Name: name}}
}

// Spans is a list of Span elements. Functions in this package expect the
// list to be in ascending command index order, and maintain that order on
// mutation.
type Spans []Span

// IndexOf returns the index of the group that contains the command id or
// -1 if not found.
func (l *Spans) IndexOf(id CmdID) int {
	return interval.IndexOf(l, uint64(id))
}

// Length returns the number of groups in the list.
func (l Spans) Length() int {
	return len(l)
}

// GetSpan returns the command index span for the group at index in the list.
func (l Spans) GetSpan(index int) interval.U64Span {
	return l[index].Bounds().Span()
}

// Span is a child of a CmdIDGroup. It is implemented by CmdIDGroup and CmdIDRange
// and SubCmdRoot
type Span interface {
	// Bounds returns the absolute range of command indices for the span.
	Bounds() CmdIDRange

	// itemCount returns the number of items this span represents to its parent.
	// For a CmdIDRange, this is the interval length.
	// For a CmdIDGroup, this is always 1.
	itemCount() uint64

	// item returns the i'th sub-item for this span.
	// For a CmdIDRange, this is the i'th CmdID in the interval.
	// For a CmdIDGroup, this is always the group itself.
	// For a SubCmdRoot, this is the subcommand index within the command
	item(i uint64) SpanItem

	// split returns two spans over the same range as this span, but where the
	// first contains the given number of items and the second the rest.
	split(i uint64) (Span, Span)
}

// SpanItem is a placeholder interface exclusively implemented by CmdIDGroup,
// SubCmdIdx and SubCmdRoot
type SpanItem interface {
	isGroupOrIDOrRoot()
}

func (CmdIDGroup) isGroupOrIDOrRoot() {}
func (SubCmdIdx) isGroupOrIDOrRoot()  {}
func (SubCmdRoot) isGroupOrIDOrRoot() {}

func (r *CmdIDRange) Bounds() CmdIDRange { return *r }
func (r *CmdIDRange) itemCount() uint64  { return r.Length() }
func (r *CmdIDRange) item(i uint64) SpanItem {
	return SubCmdIdx{uint64(r.Start + CmdID(i))}
}
func (r *CmdIDRange) split(i uint64) (Span, Span) { return r.Split(i) }

func (c *SubCmdRoot) Bounds() CmdIDRange {
	return CmdIDRange{CmdID(c.Id[len(c.Id)-1]), CmdID(c.Id[len(c.Id)-1] + 1)}
}
func (c *SubCmdRoot) itemCount() uint64           { return 1 }
func (c *SubCmdRoot) item(uint64) SpanItem        { return *c }
func (c *SubCmdRoot) split(i uint64) (Span, Span) { return c, nil }

func (g *CmdIDGroup) Bounds() CmdIDRange          { return g.Range }
func (g *CmdIDGroup) itemCount() uint64           { return 1 }
func (g *CmdIDGroup) item(uint64) SpanItem        { return *g }
func (g *CmdIDGroup) split(i uint64) (Span, Span) { return g, nil }

func (c SubCmdRoot) Index(index uint64) SpanItem {
	return c.SubGroup.Index(index)
}

// Format writes a string representing the group's name, range and sub-groups.
func (g CmdIDGroup) Format(f fmt.State, r rune) {
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
			case *CmdIDRange:
				buf.WriteString("Commands ")
			}
			buf.WriteString(strings.Replace(fmt.Sprintf("%+v", span), "\n", nl, -1))
		}
	}

	f.Write(buf.Bytes())
}

// Count returns the number of immediate items this group contains.
func (g CmdIDGroup) Count() uint64 {
	var count uint64
	for _, s := range g.Spans {
		count += s.itemCount()
	}
	return count
}

// DeepCount returns the total (recursive) number of items this group contains.
// The given predicate determines wheter the tested group is counted as 1 or
// is recursed into.
func (g CmdIDGroup) DeepCount(pred func(g CmdIDGroup) bool) uint64 {
	var count uint64
	for _, s := range g.Spans {
		switch s := s.(type) {
		case *CmdIDGroup:
			if pred(*s) {
				count += s.DeepCount(pred)
			} else {
				count++
			}
		default:
			count += s.itemCount()
		}
	}
	return count
}

// Index returns the item at the specified index.
func (g CmdIDGroup) Index(index uint64) SpanItem {
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
func (g CmdIDGroup) IndexOf(id CmdID) uint64 {
	idx := uint64(0)
	for _, s := range g.Spans {
		if s.Bounds().Contains(id) {
			if group, ok := s.(*CmdIDRange); ok {
				// ranges are flattened inline.
				return idx + uint64(id-group.Start)
			}
			return idx
		}
		idx += s.itemCount()
	}
	return 0
}

// IterateForwards calls cb with each contained command index or group starting
// with the item at index. If cb returns an error then traversal is stopped and
// the error is returned.
func (g CmdIDGroup) IterateForwards(index uint64, cb func(childIdx uint64, item SpanItem) error) error {
	childIndex := uint64(0)
	visit := func(item SpanItem) error {
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

// IterateBackwards calls cb with each contained command index or group starting
// with the item at index. If cb returns an error then traversal is stopped and
// the error is returned.
func (g CmdIDGroup) IterateBackwards(index uint64, cb func(childIdx uint64, item SpanItem) error) error {
	childIndex := g.Count() - 1
	visit := func(item SpanItem) error {
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

// If the new group does not overlap any existing groups in the list then it is
// inserted into the list, keeping ascending command-identifier order.
// If the new group sits completely within an existing group then this new group
// will be added to the existing group's sub-groups.
// If the new group completely wraps one or more existing groups in the list
// then these existing groups are added as sub-groups to the new group and then
// the new group is added to the list, keeping ascending command-identifier order.
// If the new group partially overlaps any existing group then the function will
// return an error.
//
// *** Warning ***
// All groups must be added before commands.
// Attemping to call this function after commands have been added may result in
// panics!
func (g *CmdIDGroup) AddGroup(start, end CmdID, name string, experimentableCmds []SubCmdIdx) (*CmdIDGroup, error) {
	if start > end {
		return nil, fmt.Errorf("sub-group start (%d) is greater than end (%v)", start, end)
	}
	if start < g.Range.Start {
		return nil, fmt.Errorf("sub-group start (%d) is earlier than group start (%v)", start, g.Range.Start)
	}
	if end > g.Range.End {
		return nil, fmt.Errorf("sub-group end (%d) is later than group end (%v)", end, g.Range.End)
	}
	r := CmdIDRange{Start: start, End: end}
	s, c := interval.Intersect(&g.Spans, r.Span())
	var out *CmdIDGroup
	var err error
	if c == 0 {
		// No overlaps, clean insertion
		i := sort.Search(len(g.Spans), func(i int) bool {
			return g.Spans[i].Bounds().Start > start
		})
		out = &CmdIDGroup{Name: name, Range: r, ExperimentableCmds: append(g.ExperimentableCmds, experimentableCmds...)}
		slice.InsertBefore(&g.Spans, i, out)
	} else {
		// At least one overlap
		first := g.Spans[s].(*CmdIDGroup)
		last := g.Spans[s+c-1].(*CmdIDGroup)
		sIn, eIn := first.Bounds().Contains(start), last.Bounds().Contains(end-1)
		switch {
		case c == 1 && g.Spans[s].Bounds() == r:
			// New group exactly matches already existing group. Wrap the exiting group.
			out = &CmdIDGroup{Name: name, Range: r, Spans: Spans{g.Spans[s]}, ExperimentableCmds: append(g.ExperimentableCmds, experimentableCmds...)}
			g.Spans[s] = out
		case c == 1 && sIn && eIn:
			// New group fits entirely within an existing group. Add as subgroup.
			out, err = first.AddGroup(start, end, name, append(g.ExperimentableCmds, experimentableCmds...))
		case sIn && start != first.Range.Start:
			return nil, fmt.Errorf("New group '%v' %v overlaps with existing group '%v'", name, r, first)
		case eIn && end != last.Range.End:
			return nil, fmt.Errorf("New group '%v' %v overlaps with existing group '%v'", name, r, last)
		default:
			// New group completely wraps one or more existing groups. Add the
			// existing group(s) as subgroups to the new group, and add to the list.
			out = &CmdIDGroup{Name: name, Range: r, Spans: make(Spans, c), ExperimentableCmds: append(g.ExperimentableCmds, experimentableCmds...)}
			copy(out.Spans, g.Spans[s:s+c])
			slice.Replace(&g.Spans, s, c, out)
		}
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AddRoot adds a new Subcommand Root for the given index.
// It returns the span for this SubcommandGroup
func (g *CmdIDGroup) AddRoot(rootidx []uint64, nameLookUp *SubCmdIdxTrie) *SubCmdRoot {
	r := CmdIDRange{Start: CmdID(rootidx[len(rootidx)-1]), End: CmdID(rootidx[len(rootidx)-1] + 1)}
	s, c := interval.Intersect(&g.Spans, r.Span())
	if c == 0 {
		// No groups to put this in
		i := sort.Search(len(g.Spans), func(i int) bool {
			return g.Spans[i].Bounds().Start > CmdID(rootidx[0])
		})
		slice.InsertBefore(&g.Spans, i, NewRoot(rootidx, nameLookUp))
		return g.Spans[i].(*SubCmdRoot)
	}
	if c != 1 {
		panic("This should not happen, a single command cannot span more than one group")
	}
	// We should insert into one of the spans.
	// At least one overlap
	switch first := g.Spans[s].(type) {
	case *CmdIDGroup:
		return first.AddRoot(rootidx, nameLookUp)
	case *CmdIDRange:
		firstHalf := &CmdIDRange{first.Start, CmdID(rootidx[len(rootidx)-1])}
		if firstHalf.End > firstHalf.Start {
			slice.InsertBefore(&g.Spans, s, firstHalf)
			s++
		}
		slice.Replace(&g.Spans, s, 1, NewRoot(rootidx, nameLookUp))
		secondHalf := &CmdIDRange{CmdID(rootidx[len(rootidx)-1] + 1), first.End}
		slice.InsertBefore(&g.Spans, s+1, secondHalf)
		return g.Spans[s].(*SubCmdRoot)
	default:
		x := fmt.Sprintf("Inserting root into non-group/non-range %+v, %+v", first, rootidx)
		panic(x)
	}
}

// newChildSubCmdRoots adds child SubCmdRoots to the SubCmdRoot's subgroup. If
// subcomamnds are skipped, create SubCmdRoots for them.
func (c *SubCmdRoot) newChildSubCmdRoots(r []uint64, nameLookUp *SubCmdIdxTrie) *SubCmdRoot {
	if len(r) == 0 {
		return c
	}
	nextRootRelativeIndex := r[0]
	oldEnd := c.SubGroup.Range.End
	if CmdID(nextRootRelativeIndex) >= oldEnd {
		c.SubGroup.Range.End = CmdID(nextRootRelativeIndex) + 1
	}
	if c.SubGroup.Range.End > oldEnd {
		for i := uint64(oldEnd); i < uint64(c.SubGroup.Range.End); i++ {
			c.SubGroup.AddRoot(append(c.Id, i), nameLookUp)
		}
	}
	sg := c.SubGroup.FindSubCommandRoot(CmdID(nextRootRelativeIndex))
	if sg == nil {
		sg = c.SubGroup.AddRoot(append(c.Id, nextRootRelativeIndex), nameLookUp)
	}
	return sg.newChildSubCmdRoots(r[1:], nameLookUp)
}

// Insert adds a new subcommand into the SubCmdRoot. The subcommand is specified
// with its relative hierarchy to the target SubCmdRoot. If the subcommand is
// not an immediate child of the target SubCmdRoot (i.e. len(r) > 1) , new
// child SubCmdRoots will be created under the target SubCmdRoot, until the
// immediate parent of the subcommand is created.
func (c *SubCmdRoot) Insert(r []uint64, nameLookUp *SubCmdIdxTrie) {
	childRoot := c.newChildSubCmdRoots(r[0:len(r)-1], nameLookUp)
	// Add subcommands one-by-one to the SubCmdRoot and its subgroups/child
	// SubCmdRoots
	id := r[len(r)-1]
	if CmdID(id) > childRoot.SubGroup.Range.End {
		childRoot.SubGroup.Range.End = CmdID(id + 1)
	}
	for i := CmdID(0); i < CmdID(id); i++ {
		childRoot.SubGroup.AddCommand(i)
	}
}

func (c *SubCmdRoot) InsertWithFilter(r []uint64, nameLookUp *SubCmdIdxTrie, filter func(CmdID) bool) {
	childRoot := c.newChildSubCmdRoots(r[0:len(r)-1], nameLookUp)
	// Add subcommands one-by-one to the SubCmdRoot and its subgroups/child
	// SubCmdRoots
	id := r[len(r)-1]
	if CmdID(id) > childRoot.SubGroup.Range.End {
		childRoot.SubGroup.Range.End = CmdID(id + 1)
	}
	for i := CmdID(0); i < CmdID(id); i++ {
		if filter(i) {
			childRoot.SubGroup.AddCommand(i)
		}
	}
}

// AddSubCmdMarkerGroups adds the given groups to the target SubCmdRoot
// with the relative hierarchy specified in r. If the groups are not added as
// immediate children of the target SubCmdRoot (r is not empty), child
// SubCmdRoots will be created under the target SubCmdRoot recursively until
// the immediate parent SubCmdRoot is created.
func (c *SubCmdRoot) AddSubCmdMarkerGroups(r []uint64, groups []*CmdIDGroup, nameLookUp *SubCmdIdxTrie) error {
	childRoot := c.newChildSubCmdRoots(r, nameLookUp)
	for _, g := range groups {
		if g.Range.Start < childRoot.SubGroup.Range.Start {
			childRoot.SubGroup.Range.Start = g.Range.Start
		}
		if g.Range.End > childRoot.SubGroup.Range.End {
			childRoot.SubGroup.Range.End = g.Range.End
		}
		_, err := childRoot.SubGroup.AddGroup(g.Range.Start, g.Range.End, g.Name, g.ExperimentableCmds)
		if err != nil {
			return err
		}
	}
	return nil
}

// FindSubCommandRoot returns the SubCmdRoot that represents the given CmdID.
func (g CmdIDGroup) FindSubCommandRoot(id CmdID) *SubCmdRoot {
	for _, x := range g.Spans {
		switch k := x.(type) {
		case *SubCmdRoot:
			if CmdID(k.Id[len(k.Id)-1]) == id {
				return k
			}
		case *CmdIDGroup:
			y := k.FindSubCommandRoot(id)
			if y != nil {
				return y
			}
		}
	}
	return nil
}

// AddCommand adds the command to the groups.
func (g *CmdIDGroup) AddCommand(id CmdID) bool {
	i := sort.Search(len(g.Spans), func(i int) bool {
		return id < g.Spans[i].Bounds().Start
	})

	var prev, next *CmdIDRange
	if i > 0 {
		if span := g.Spans[i-1]; span.Bounds().Contains(id) {
			// id is within an existing span
			switch span := span.(type) {
			case *CmdIDGroup:
				return span.AddCommand(id)
			default:
				return false // Collision
			}
		}

		// id is not inside an existing span.

		switch span := g.Spans[i-1].(type) {
		case *CmdIDRange:
			if span.End == id {
				prev = span
			}
		}
	}

	if i < len(g.Spans) {
		switch span := g.Spans[i].(type) {
		case *CmdIDRange:
			if span.Start == id+1 {
				next = span
			}
		}
	}

	switch {
	case prev != nil && next != nil: // merge
		prev.End = next.End
		slice.RemoveAt(&g.Spans, i, 1)
	case prev != nil: // grow prev
		prev.End++
	case next != nil: // grow next
		next.Start--
	default: // insert
		slice.InsertBefore(&g.Spans, i, &CmdIDRange{id, id + 1})
	}

	return true
}

// Cluster groups together chains of command using the limits maxChildren and
// maxNeighbours.
//
// If maxChildren is positive, the group, and any of it's decendent
// groups, which have more than maxChildren child elements, will have their
// children grouped into new synthetic groups of at most maxChildren children.
//
// If maxNeighbours is positive, we will group long list of ungrouped commands,
// which are next to a group. This ensures the group is not lost in noise.
func (g *CmdIDGroup) Cluster(maxChildren, maxNeighbours uint64) {
	if maxNeighbours > 0 {
		spans := Spans{}
		accum := Spans{}
		flush := func() {
			if len(accum) > 0 {
				rng := CmdIDRange{accum[0].Bounds().Start, accum[len(accum)-1].Bounds().End}
				group := CmdIDGroup{"Sub Group", rng, accum, []SubCmdIdx{}, nil}
				if group.Count() > maxNeighbours {
					spans = append(spans, &group)
				} else {
					spans = append(spans, accum...)
				}
				accum = Spans{}
			}
		}
		for _, s := range g.Spans {
			switch s.(type) {
			case *CmdIDRange, *SubCmdRoot:
				accum = append(accum, s)
			default:
				flush()
				spans = append(spans, s)
			}
		}
		if len(spans) > 0 { // Do not make one big nested group
			flush()
			g.Spans = spans
		}
	}

	if maxChildren > 0 && g.Count() > maxChildren {
		g.Spans = g.Spans.Split(maxChildren)
	}

	if maxNeighbours > 0 || maxChildren > 0 {
		for _, s := range g.Spans {
			switch c := s.(type) {
			case *CmdIDGroup:
				c.Cluster(maxChildren, maxNeighbours)
			case *SubCmdRoot:
				c.SubGroup.Cluster(maxChildren, maxNeighbours)
			}
		}
	}
}

// Flatten replaces this node's children with its grandchildren.
func (g *CmdIDGroup) Flatten() {
	var newSpans Spans
	for _, s := range g.Spans {
		switch v := s.(type) {
		case *SubCmdRoot:
			newSpans = append(newSpans, v.SubGroup.Spans...)
		case *CmdIDGroup:
			newSpans = append(newSpans, v.Spans...)
		default:
			newSpans = append(newSpans, s)
		}
	}
	g.Spans = newSpans
}

// Split returns a new list of spans where each new span will represent no more
// than the given number of items.
func (l Spans) Split(max uint64) Spans {
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
			out = append(out, &CmdIDGroup{
				fmt.Sprintf("Sub Group %d", idx),
				CmdIDRange{current[0].Bounds().Start, current[len(current)-1].Bounds().End},
				current,
				[]SubCmdIdx{},
				nil,
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
		out = append(out, &CmdIDGroup{
			fmt.Sprintf("Sub Group %d", idx),
			CmdIDRange{current[0].Bounds().Start, current[len(current)-1].Bounds().End},
			current,
			[]SubCmdIdx{},
			nil,
		})
	}
	return out
}

// TraverseCallback is the function that's called for each traversed item in a
// group.
type TraverseCallback func(indices []uint64, item SpanItem) error

// Traverse traverses the command group starting with the specified index,
// calling cb for each encountered node.
func (g CmdIDGroup) Traverse(backwards bool, start []uint64, cb TraverseCallback) error {
	t := groupTraverser{backwards: backwards, cb: cb}

	// Make a copy of start as traversal alters the slice.
	indices := make([]uint64, len(start))
	copy(indices, start)

	groups := make([]CmdIDGroup, 1, len(indices)+1)
	groups[0] = g
	for i := range indices {
		item := groups[i].Index(indices[i])
		g, ok := item.(CmdIDGroup)
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
			// CmdIDGroup doesn't have an index specifiying a child to search from.
			// Search the entire group.
			if backwards {
				err = g.IterateBackwards(g.Count()-1, t.visit)
			} else {
				err = g.IterateForwards(0, t.visit)
			}
		case i == len(groups)-1:
			// CmdIDGroup is the deepest.
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
			// CmdIDGroup is not the deepest.
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

func (s *groupTraverser) visit(childIdx uint64, item SpanItem) error {
	if !s.backwards {
		if err := s.cb(append(s.indices, childIdx), item); err != nil {
			return err
		}
	}
	if g, ok := item.(CmdIDGroup); ok {
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
