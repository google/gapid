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

package parse

import (
	"unicode/utf8"

	"github.com/google/gapid/core/text/parse/cst"
)

type SkipMode int

const (
	// SkipPrefix is the skip mode that skips tokens that are associated with
	// the following lexically relevant token. This is mostly important for
	// comment association.
	SkipPrefix SkipMode = iota
	// SkipSuffix is the skip mode that skips tokens that are associated with
	// the preceding lexically relevant token. This is mostly important for
	// comment association.
	SkipSuffix
)

// Skip is the function used to skip separating tokens.
// A separating token is one where, as far as the parser is concerned, the
// tokens do not exist, even though the tokens may have been necessary to
// separate the lexical tokens (whitespace), or carry useful information
// (comments).
type Skip func(parser *Parser, mode SkipMode) cst.Separator

// NewSkip builds a Skip function for the common case of a parser that has one
// type of line comment, one type of block comment, and want to treat all
// unicode space characters as skippable.
func NewSkip(line, blockstart, blockend string) Skip {
	return func(p *Parser, mode SkipMode) cst.Separator {
		var sep cst.Separator
		for {
			switch {
			case p.Space():
				sep = append(sep, p.Consume())
			case mode == SkipPrefix && p.EOL():
				sep = append(sep, p.Consume())
			case p.String(line):
				if !p.SeekRune('\n') {
					for !p.IsEOF() {
						p.Advance()
					}
				}
				sep = append(sep, p.Consume())
			case p.String(blockstart):
				first, _ := utf8.DecodeRuneInString(blockend)
				for {
					if !p.SeekRune(first) {
						p.Error("Unterminated block comment")
						break
					}
					if p.String(blockend) {
						break
					}
					p.Advance()
				}
				sep = append(sep, p.Consume())
			default:
				return sep
			}
		}
	}
}
