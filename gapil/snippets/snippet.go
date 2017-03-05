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

package snippets

// binary: java.source = service
// binary: java.package = com.google.gapid.service.snippets
// binary: java.indent = "  "
// binary: java.member_prefix = my

import (
	"bytes"
	"fmt"

	"github.com/google/gapid/framework/binary"
)

const (
	// Name of the synthetic entity type used for API globals
	GlobalsTypename = "State"
)

// Snippet is static information collected from the semantic tree.
type Snippet interface {
	// isSnippet() used for interface compliance
	isSnippet()

	// IsEmpty returns true if the snippet has no interesting content.
	// Some snippets are created to represent structure, before it is
	// known whether that structure will have interesting content.
	IsEmpty() bool

	// finder returns a finder func which can be used to build a
	// KindredSnippets collection which collects only snippets of the
	// same type as this snippet.
	finder() findFun
}

// Snippets, a collection of snippets.
type Snippets []Snippet

// KindredSnippets, is a collection of snippets which are all of the same
// underlying snippet type. KindredSnippets is binary codable.
type KindredSnippets binary.Object

// KindredSnippetsCast needed for generated code.
func KindredSnippetsCast(o binary.Object) KindredSnippets {
	if o == nil {
		return nil
	} else {
		return o.(KindredSnippets)
	}
}

// findFun is a func which can be used to build a KindredSnippets collection
// which is a binary.Object which contains only snippets of a specified type.
// Snippets of other types are return as an addition return value.
type findFun func(Pathway, Snippets) (KindredSnippets, Snippets)

// Add adds a snippet to the snippets. The snippet is not added if it
// contains no interested information or if it is a duplicate.
func (m *Snippets) Add(snippet Snippet) {
	if snippet == nil || snippet.IsEmpty() {
		return
	}
	for _, s := range *m {
		if s == snippet {
			return
		}
	}
	*m = append(*m, snippet)
}

// addSnippets adds a list of snippets to the receiver. Snippets are not
// added if they contain no interested information or if they are a duplicate.
func (s *Snippets) AddSnippets(snippets Snippets) {
	for _, snip := range snippets {
		s.Add(snip)
	}
}

// IsEmpty returns true if the list of snippets contains no snippet with
// interesting information.
func (s *Snippets) IsEmpty() bool {
	for _, snip := range *s {
		if !snip.IsEmpty() {
			return false
		}
	}
	return true
}

func (m *Snippets) String() string {
	buf := &bytes.Buffer{}
	buf.WriteString(fmt.Sprint("snippets{"))
	for i, s := range *m {
		if i != 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("%s", s))
	}
	buf.WriteString("}")
	return buf.String()
}

// AddKindredGroups collects snippets together based on there underlying
// snippet type. Results are appended to kindreds.
func (m *Snippets) AddKindredGroups(path Pathway, kindreds *[]KindredSnippets) {
	snippets := *m

	for len(snippets) > 0 {
		var kindred KindredSnippets
		kindred, snippets = snippets[0].finder()(path, snippets)
		*kindreds = append(*kindreds, kindred)
	}
}

// AtomSnippets these are all the snippets related to a specific atom.
type AtomSnippets struct {
	binary.Generate
	AtomName string            // name of the atom
	Snippets []KindredSnippets // snippets groups by kind
}
