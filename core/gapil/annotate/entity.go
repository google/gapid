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
	"sort"

	"github.com/google/gapid/core/gapil/semantic"
	"github.com/google/gapid/core/gapil/snippets"
)

type entity struct {
	self         *Location
	typ          semantic.Type
	fieldsByName SymbolTable
}

var _ Nested = &entity{}

// newEntity creates a new entity nested structure with a new location
// (a temporary). To create an entity at an existing location use
// (*Location).newEntity()
func newEntity(t semantic.Type) *entity {
	l := &Location{}
	return l.newEntity(t)
}

// Field returns the equivalence set leader for the field of the entity which
// is named name.
func (e *entity) Field(name string) *Location {
	if l, ok := e.fieldsByName[name]; ok {
		return l.leader()
	}
	l := &Location{}
	e.fieldsByName[name] = l
	return l.checkLeader()
}

// leader returns the leader of the equivalence set for the entity.
func (e *entity) leader() *Location {
	return e.self.leader()
}

// isNestedEmpty returns true if the entity has no interesting information.
func (e *entity) isNestedEmpty() bool {
	if len(e.fieldsByName) == 0 {
		return true
	}

	for _, l := range e.fieldsByName {
		if !l.leader().isEmpty() {
			return false
		}
	}
	return true
}

// getSnippets returns the snippets for this entities' equivalence set.
func (e *entity) getSnippets() snippets.Snippets {
	return e.leader().getSnippets()
}

// merge merges nested into the receiver. It is an error if nested is not
// an entity or if the structure of any field does not correspond.
func (e *entity) merge(nested Nested) error {
	// Note no need to merge e.self as merge() is called from Location.alias()
	ee, ok := nested.(*entity)
	if !ok {
		return fmt.Errorf("expected entity got %T", nested)
	}
	if e.typ != ee.typ {
		return fmt.Errorf("type mismatch %v with %v", e.typ, ee.typ)
	}
	// Alias all the fields which are in both
	for f, l := range e.fieldsByName {
		if ll, ok := ee.fieldsByName[f]; ok {
			if err := alias(l, ll); err != nil {
				return fmt.Errorf("Failed to alias entity field %s: %v", f, err)
			}
		}
	}
	// Take in the right which are not in the left and add them to the left
	for ff, ll := range ee.fieldsByName {
		if _, ok := e.fieldsByName[ff]; !ok {
			e.fieldsByName[ff] = ll
		}
	}
	return nil
}

// getTable traverses the entity structure and populates table with
// snippets. Each entry is associated with the path which leads to them.
func (e *entity) getTable(path snippets.Pathway, table *SnippetTable) {
	names := make(sort.StringSlice, len(e.fieldsByName))
	i := 0
	for f := range e.fieldsByName {
		names[i] = f
		i++
	}
	sort.Sort(names)
	for _, f := range names {
		l := e.fieldsByName[f]
		l.leader().getTable(snippets.Field(path, f), table)
	}
}

func (e *entity) String() string {
	buf := &bytes.Buffer{}
	buf.WriteString("entity(")
	buf.WriteString(fmt.Sprintf("%s{", nodeString(e.typ, nil)))
	buf.WriteString(fmt.Sprintf("%s", e.fieldsByName))
	buf.WriteString(fmt.Sprintf("})"))
	return buf.String()
}
