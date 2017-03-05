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

package annotate

import (
	"bytes"
	"fmt"

	"github.com/google/gapid/gapil/semantic"
	"github.com/google/gapid/gapil/snippets"
)

// Location is a static representation of an assignable location in the
// program. Static information flow is used to propagate snippets to
// locations. It is the conceit of static analysis, that the snippets are
// inferred at the locations.
type Location struct {
	parent *Location         // If nil then this location contains the equivalence set
	snips  snippets.Snippets // Snippets at this location
	nested Nested            // Decomposable
}

// alias aliases two locations. Aliasing allows information collected at
// different locations to be merged into a single equivalence set. All
// mutations always happen on the equivalence set leader.
func alias(left *Location, right *Location) error {
	return left.leader().alias(right.leader())
}

// Make a new location with a snippet associated with it.
func newLocation(snippet snippets.Snippet) *Location {
	l := &Location{}
	l.addSnippet(snippet)
	return l
}

// addSnippet adds a snippet to this location.
func (l *Location) addSnippet(s snippets.Snippet) {
	l = l.leader()
	l.snips.Add(s)
}

// getSnippets returns the snippets.
func (l *Location) getSnippets() snippets.Snippets {
	l = l.leader()
	return l.snips
}

// alias makes l and right refer to the same equivalence set. Snippets are
// merged if necessary.
func (l *Location) alias(right *Location) error {
	l = l.leader()
	r := right.leader()
	if l == r {
		return nil
	}

	if l.nested == nil {
		l.nested = r.nested
	} else {
		l.nested.merge(r.nested)
	}

	if l.snips == nil {
		l.snips = r.snips
	} else {
		l.snips.AddSnippets(r.snips)
	}

	r.parent = l
	return nil
}

// leader returns the leader of the equivalence set for this location. The
// leader maybe this location. The path to the leader is shorten for all
// locations which are traversed to find the leader.
func (l *Location) leader() *Location {
	var backtrace []**Location
	loc := l
	for ; loc.parent != nil; loc = loc.parent {
		backtrace = append(backtrace, &loc.parent)
	}
	// Shorten the traversal for future calls (as in union find)
	// Note backtrace[len(backtrace) - 1] already points to loc
	for i := len(backtrace) - 2; i >= 0; i-- {
		(*backtrace[i]) = loc
	}
	// Shorten the traversal for future kinds (as in union find)
	return loc.checkLeader()
}

// checkLeader returns this location boxed as a Leader. It checks that the
// location is currently a leader and panics if it is not. Note that
// aliasing a leader can make it lose its leadership status. So the
// static type can only be regarded as correct for immediate use.
func (l *Location) checkLeader() *Location {
	if l.parent != nil {
		panic(fmt.Errorf("Location is not leader. Loc=%p Parent=%p Leader=%p",
			l, l.parent, l.leader()))
	}
	return l
}

// isEmpty returns true if the location contains no interesting information.
// Some location exist to represent structure, before it is known whether
// there will be interesting information at this location.
func (l *Location) isEmpty() bool {
	l = l.leader()
	if l.nested == nil || l.nested.isNestedEmpty() {
		return l.snips.IsEmpty()
	}
	// nested is not empty
	return false
}

// getLeaves traverses a potentionally nested location and collects all
// the snippets in a table associated with the path which leads to them.
func (l *Location) getTable(path snippets.Pathway, table *SnippetTable) {
	l = l.leader()
	if len(l.snips) != 0 {
		table.Add(path, l.snips)
	}
	if l.nested != nil {
		l.nested.getTable(path, table)
	}
}

// newEntity creates a new entity nested structure at this location.
func (l *Location) newEntity(t semantic.Type) *entity {
	l = l.leader()
	e := &entity{self: l, typ: t, fieldsByName: SymbolTable{}}
	l.nested = e
	return e
}

// newContainer creates a new container nested structure at this location.
func (l *Location) newContainer() *container {
	l = l.leader()
	c := &container{self: l, key: &Location{}, elem: &Location{}}
	l.nested = c
	return c
}

// newPointer creates a new pointer nested structure at this location.
func (l *Location) newPointer() *pointer {
	l = l.leader()
	p := &pointer{self: l}
	l.nested = p
	return p
}

// container returns a nested container structure for this location. A new
// container will be made, if one does not exist. It will panic, if the
// location already represents a nested structure other than a container.
func (l *Location) container(expr semantic.Expression) *container {
	l = l.leader()
	if l.nested == nil {
		return l.newContainer()
	}
	if c, ok := l.nested.(*container); ok {
		return c
	} else {
		panic(fmt.Errorf("location is not a container %T:%v for %T:%v", l.nested, l.nested, expr, expr))
	}
}

// entity returns a nested entity structure for this location.
// A new entity will be made, if one does not exist. It will panic, if
// the location already represents a nested structure other than an entity.
func (l *Location) entity(expr semantic.Expression) *entity {
	l = l.leader()
	if l.nested == nil {
		typ := expr.ExpressionType()
		switch typ := typ.(type) {
		case *semantic.Reference, *semantic.Class:
			return l.newEntity(typ)
		default:
			panic(fmt.Errorf("Can not make an entity for %T:%v in %T:%v",
				typ, typ, expr, expr))
		}
	}
	if e, ok := l.nested.(*entity); ok {
		return e
	} else {
		panic(fmt.Errorf("location is not an entity %T:%v", l.nested, l.nested))
	}
}

// pointer returns a nested pointer to container structure for this location.
// A new pointer will be made, if one does not exist. It will panic, if the
// location already represents a nested structure other than an entity.
func (l *Location) pointer(expr semantic.Expression) *pointer {
	l = l.leader()
	if l.nested == nil {
		typ := semantic.Underlying(expr.ExpressionType())
		switch typ := typ.(type) {
		case *semantic.Pointer:
			return l.newPointer()
		default:
			panic(fmt.Errorf("Can not make a pointer for %T:%v in %T:%v",
				typ, typ, expr, expr))
		}
	}
	if p, ok := l.nested.(*pointer); ok {
		return p
	} else {
		panic(fmt.Errorf("location is not a pointer %T:%v", l.nested, l.nested))
	}
}

func (l *Location) getNested() Nested {
	if l == nil {
		return nil
	}
	l = l.leader()
	return l.nested
}

func (l *Location) String() string {
	if l.parent != nil {
		return fmt.Sprintf("<alias for %s>", l.leader())
	}
	l.checkLeader()
	buf := &bytes.Buffer{}
	buf.WriteString("location@")
	buf.WriteString(fmt.Sprintf("%p(", l))
	for i, a := range l.snips {
		if i != 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("%s", a))
	}
	buf.WriteString(")")
	if l.nested != nil {
		buf.WriteString(" is ")
		buf.WriteString(fmt.Sprintf("%s", l.nested))
	}
	return buf.String()
}
