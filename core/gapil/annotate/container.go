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

	"github.com/google/gapid/core/gapil/snippets"
)

// container represents the nested structure of a container type such
// as Map, Array and Slice. It is also used for pointer, where Key is
// ignored.
type container struct {
	// Map, Array, Slice
	self *Location
	key  *Location
	elem *Location
}

var _ Nested = &container{}

// newContainer creates a new container nested structure with a new location
// (a temporary). To create a container at an existing location use
// (*Location).newContainer()
func newContainer() *container {
	l := &Location{}
	return l.newContainer()
}

// Elem returns the equivalence set leader for an element of the collection.
func (c *container) Elem() *Location {
	return c.elem.leader()
}

// Key returns the equivalence set leader for the key of the collection.
func (c *container) Key() *Location {
	return c.key.leader()
}

// leader returns the leader of the equivalence set for the container.
func (c *container) leader() *Location { return c.self.leader() }

// isNestedEmpty returns true if the container has no interesting information.
func (c *container) isNestedEmpty() bool {
	return c.elem.leader().isEmpty() && c.key.leader().isEmpty()
}

func (c *container) String() string {
	return fmt.Sprintf("container(key: %s, elem: %s)", c.key, c.elem)
}

// getSnippets returns the snippets for this container's equivalence set.
func (c *container) getSnippets() snippets.Snippets {
	return c.leader().getSnippets()
}

// merge merges nested into the receiver. It is an error if nested is
// not a container or if the structure of key or element does not correspond.
func (c *container) merge(nested Nested) error {
	if cc, ok := nested.(*container); !ok {
		return fmt.Errorf("expected container got %T", nested)
	} else {
		if err := alias(c.key, cc.key); err != nil {
			return fmt.Errorf("failed alias of container key %v", err)
		}
		if err := alias(c.elem, cc.elem); err != nil {
			return fmt.Errorf("failed alias of container element %v", err)
		}
	}
	return nil
}

// getTable traverses the container structure and populates table with
// snippets. Each entry is associated with the path which leads to them.
func (c *container) getTable(path snippets.Pathway, table *SnippetTable) {
	c.elem.leader().getTable(snippets.Elem(path), table)
	c.key.leader().getTable(snippets.Key(path), table)
}
