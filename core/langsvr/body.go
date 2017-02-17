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

import (
	"sort"

	"github.com/google/gapid/core/langsvr/protocol"
	"github.com/google/gapid/core/math/sint"
)

// Body represents the body of a document.
type Body struct {
	runes []rune
	eol   []int
}

// NewBody returns a body containing the specified text.
func NewBody(text string) Body {
	return NewBodyFromRunes([]rune(text))
}

// NewBodyFromRunes returns a body containing the specified text in runes.
func NewBodyFromRunes(runes []rune) Body {
	b := Body{runes: runes}
	b.eol = []int{-1}
	for i, r := range runes {
		if r == '\n' {
			b.eol = append(b.eol, i)
		}
	}
	b.eol = append(b.eol, len(runes))
	return b

}

// Runes returns the body's runes.
func (b Body) Runes() []rune {
	return b.runes
}

// Text returns the body's text.
func (b Body) Text() string {
	return string(b.runes)
}

// FullRange returns the entire range of the document.
func (b Body) FullRange() Range {
	return Range{b.Position(0), b.Position(len(b.runes))}
}

// GetRange returns a chunk of the body.
func (b Body) GetRange(rng Range) string {
	s, e := b.Offset(rng.Start), b.Offset(rng.End)
	return string(b.runes[s:e])
}

// Position returns the position at the specified offset
func (b Body) Position(offset int) Position {
	offset = sint.Min(offset, len(b.runes))
	l := sort.SearchInts(b.eol, offset)
	c := offset - b.eol[l-1]
	return Position{l, c}
}

// Range returns the range for the given start and end offsets.
func (b Body) Range(start, end int) Range {
	return Range{b.Position(start), b.Position(end)}
}

// Offset returns the byte offset of p.
func (b Body) Offset(p Position) int {
	return b.offset(p.toProtocol())
}

func (b Body) offset(p protocol.Position) int {
	nextEOL := len(b.runes)
	if p.Line >= len(b.eol) {
		return nextEOL
	}
	if p.Line+1 < len(b.eol) {
		nextEOL = b.eol[p.Line+1]
	}
	return sint.Min(b.eol[p.Line]+p.Column+1, nextEOL)
}
