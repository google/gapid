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

	"github.com/google/gapid/core/gapil/snippets"
)

// Nested is an interface which is provided by any type structure where
// there are sub-types.
type Nested interface {
	// leader, allows a nested structures to be aliased.
	leader() *Location

	// merge merges nested into the receiver. It returns an error if
	// nested does not have the same structure as the receiver.
	merge(nested Nested) error

	// getTable traverses the nested structure and populates table with
	// snippets. Each entry is associated with the path which leads to them.
	getTable(path snippets.Pathway, table *SnippetTable)

	// isNestedEmpty returns true if nested structure contains no interesting
	// information. Note that isNestedEmpty() is called by Location.isEmpty()
	// and it does not call Self.isEmpty(), which would cause a loop.
	isNestedEmpty() bool
}

type entry struct {
	path     snippets.Pathway
	snippets snippets.Snippets
}

// SnippetTable is a table associating a path to the snippets at that path.
type SnippetTable []entry

// Add adds snippets to the snippet table and the specified path.
func (t *SnippetTable) Add(path snippets.Pathway, snippets snippets.Snippets) {
	*t = append(*t, entry{path, snippets})
}

func (t *SnippetTable) String() string {
	buf := bytes.Buffer{}
	for _, e := range *t {
		buf.WriteString(fmt.Sprintf("%s", e.path))
		buf.WriteString(fmt.Sprintf(": %s\n", e.snippets))
	}
	return buf.String()
}

// Group the snippets in the snippet table by their underlying type.
// Make the paths relative to API type name typeName.
func (t *SnippetTable) KindredGroups(typeName string) []snippets.KindredSnippets {
	kindreds := make([]snippets.KindredSnippets, 0, len(*t))
	for _, e := range *t {
		e.snippets.AddKindredGroups(snippets.MakeRelative(e.path, typeName), &kindreds)
	}
	return kindreds
}

// Globals return a snippet table which just includes snippets with
// paths in the global state.
func (t SnippetTable) Globals() SnippetTable {
	var table SnippetTable
	for _, entry := range t {
		if snippets.IsGlobal(entry.path) {
			table = append(table, entry)
		}
	}
	return table
}

// NonGlobals return a snippet table which just includes snippets with
// paths to non-global state.
func (t SnippetTable) NonGlobals() SnippetTable {
	var table SnippetTable
	for _, entry := range t {
		if !snippets.IsGlobal(entry.path) {
			table = append(table, entry)
		}
	}
	return table
}
