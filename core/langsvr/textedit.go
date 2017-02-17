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

package langsvr

import "github.com/google/gapid/core/langsvr/protocol"

// WorkspaceEdit is a collection of edits across an entire workspace.
type WorkspaceEdit map[string]TextEditList

// Add appends an edit at the specified location
func (w WorkspaceEdit) Add(loc Location, newText string) {
	l := w[loc.URI]
	l.Add(loc.Range, newText)
	w[loc.URI] = l
}

// TextEdit is a textual edit applicable to a text document.
type TextEdit struct {
	// The range of the text document to be manipulated.
	Range Range

	// The string to be inserted. For delete operations use an empty string.
	NewText string
}

// TextEditList is a list of TextEdits
type TextEditList []TextEdit

// Add appends a TextEdit to the list.
func (l *TextEditList) Add(rng Range, newText string) {
	*l = append(*l, TextEdit{rng, newText})
}

func (w WorkspaceEdit) toProtocol() protocol.WorkspaceEdit {
	changes := map[string][]protocol.TextEdit{}
	for uri, edits := range w {
		changes[uri] = edits.toProtocol()
	}
	return protocol.WorkspaceEdit{Changes: changes}
}

func (e TextEdit) toProtocol() protocol.TextEdit {
	return protocol.TextEdit{
		Range:   e.Range.toProtocol(),
		NewText: e.NewText,
	}
}

func (l TextEditList) toProtocol() []protocol.TextEdit {
	out := make([]protocol.TextEdit, len(l))
	for i, e := range l {
		out[i] = e.toProtocol()
	}
	return out
}
