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

// Highlight a range inside a text document which deserves special attention.
// Usually a document highlight is visualized by changing the background color
// of its range.
type Highlight struct {
	// The range this highlight applies to.
	Range Range

	// The highlight kind, default is Text.
	Kind HighlightKind
}

// HighlightList is a list of highlights
type HighlightList []Highlight

// Add appends the highlight to the list.
func (l *HighlightList) Add(rng Range, kind HighlightKind) {
	*l = append(*l, Highlight{rng, kind})
}

func (h Highlight) toProtocol() protocol.DocumentHighlight {
	kind := protocol.DocumentHighlightKind(h.Kind)
	return protocol.DocumentHighlight{
		Range: h.Range.toProtocol(),
		Kind:  &kind,
	}
}

func (l HighlightList) toProtocol() []protocol.DocumentHighlight {
	out := make([]protocol.DocumentHighlight, len(l))
	for i, h := range l {
		out[i] = h.toProtocol()
	}
	return out
}

// HighlightKind is a document highlight kind enumerator.
type HighlightKind int

const (
	// TextHighlight represents a textual occurrance.
	TextHighlight = HighlightKind(protocol.TextHighlight)

	// ReadHighlight represents read-access of a symbol, like reading a
	// variable.
	ReadHighlight = HighlightKind(protocol.ReadHighlight)

	// WriteHighlight represents write-access of a symbol, like writing to a
	// variable.
	WriteHighlight = HighlightKind(protocol.WriteHighlight)
)
