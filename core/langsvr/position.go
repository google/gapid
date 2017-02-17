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

// Position represents a location in a document.
type Position struct {
	// Line index (1-based)
	Line int
	// Column index (1-based)
	Column int
}

func pos(p protocol.Position) Position {
	return Position{
		Line:   p.Line + 1,
		Column: p.Column + 1,
	}
}

func rng(p protocol.Range) Range {
	return Range{
		Start: pos(p.Start),
		End:   pos(p.End),
	}
}

func (p Position) toProtocol() protocol.Position {
	return protocol.Position{
		Line:   p.Line - 1,
		Column: p.Column - 1,
	}
}

// Range represents a span of a document.
type Range struct {
	// The start of the span.
	Start Position
	// The end of the span.
	End Position
}

func (r Range) toProtocol() protocol.Range {
	return protocol.Range{
		Start: r.Start.toProtocol(),
		End:   r.End.toProtocol(),
	}
}

// Location represents a location inside a resource, such as a line inside a text file.
type Location struct {
	URI   string
	Range Range
}

func (l Location) toProtocol() protocol.Location {
	return protocol.Location{
		URI:   l.URI,
		Range: l.Range.toProtocol(),
	}
}

// NoLocation represents no location
var NoLocation Location
