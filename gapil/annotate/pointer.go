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
	"fmt"

	"github.com/google/gapid/gapil/snippets"
)

// pointer represents the nested structure of a pointer type. In the API
// language a pointer can be converted into a slice. As observations happen
// on a slice we need to represent this relationship.
type pointer struct {
	self *Location
	rng  *Location
}

var _ Nested = &pointer{}

// newPointer creates a new pointer nested structure with a new location.
// To create a pointer at an existing location use (*Location).newPointer().
func newPointer() *pointer {
	l := &Location{}
	return l.newPointer()
}

// Range returns the equivalence set leader for a range over the collection
// pointed into by the pointer.
func (p *pointer) Range() *Location {
	if p.rng == nil {
		p.rng = &Location{}
	}
	return p.rng.leader()
}

// leader returns the leader of the equivalence set for the pointer.
func (p *pointer) leader() *Location { return p.self.leader() }

// isNestedEmpty return true if the pointer has no interesting information.
func (p *pointer) isNestedEmpty() bool {
	return p.rng == nil || p.rng.isEmpty()
}

func (p *pointer) String() string {
	return fmt.Sprintf("pointer(range: %s)", p.rng)
}

// getSnippets returns the snippets for this pointer's equivalence set.
func (p *pointer) getSnippets() snippets.Snippets {
	return p.leader().getSnippets()
}

// merge merges nested into the receiver. It is an error if nested is
// not a pointer or if the structure of range does not correspond.
func (p *pointer) merge(nested Nested) error {
	if pp, ok := nested.(*pointer); !ok {
		return fmt.Errorf("expected pointer got %T", nested)
	} else if err := alias(p.rng, pp.rng); err != nil {
		return fmt.Errorf("failed alias of pointer range %v", err)
	}
	return nil
}

// getTable traverses the pointer structure and populates table with
// snippets. Each entry is associated with the path which leads to them.
func (p *pointer) getTable(path snippets.Pathway, table *SnippetTable) {
	if p.rng != nil {
		p.rng.leader().getTable(snippets.Range(path), table)
	}
}
