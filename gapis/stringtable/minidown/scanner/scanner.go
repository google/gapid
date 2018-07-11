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

// Package scanner implements the minidown token scanner.
package scanner

import (
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/core/text/parse/cst"
	"github.com/google/gapid/gapis/stringtable/minidown/token"
)

func skip(p *parse.Parser, mode parse.SkipMode) cst.Separator {
	for i := 0; ; i++ {
		r := p.PeekN(i)
		if !unicode.IsSpace(r) || r == '\r' || r == '\n' || r == utf8.RuneError {
			if i == 0 {
				return nil
			}
			p.AdvanceN(i)
			return cst.Separator{p.Consume()}
		}
	}
}

// Scan scans a minidown file producing a token list.
func Scan(filename, data string) ([]token.Token, parse.ErrorList) {
	tokens := []token.Token{}
	parser := func(p *parse.Parser, root *cst.Branch) {
		i := 0
		for !p.IsEOF() {
			i++
			if i > 1000000 {
				panic(fmt.Sprintf("Scanner cannot progress when scanning: '%s'\n"+
					"Stuck at rune: 0x%x '%c'", data, p.Peek(), p.Peek()))
			}
			if p.IsEOL() {
				// Keep the \r out of ParseLeaf so we normalize CRLF -> LF.
				p.Rune('\r')
				p.Consume()

				p.ParseLeaf(root, func(cst *cst.Leaf) {
					p.Rune('\n')
					tokens = append(tokens, token.NewLine{Leaf: cst})
				})
				continue
			}
			switch p.Peek() {
			case '\\':
				p.ParseLeaf(root, func(cst *cst.Leaf) {
					p.Advance()
					r := p.Peek()
					switch r {
					case '{', '}', '(', ')', '[', ']', '*', '\\':
						p.Advance()
						tokens = append(tokens, token.Text{Leaf: cst, Override: string([]rune{r})})
					default:
						tokens = append(tokens, token.Text{Leaf: cst})
					}
				})

			case '#':
				p.ParseLeaf(root, func(cst *cst.Leaf) {
					for p.Rune('#') {
					}
					if unicode.IsSpace(p.Peek()) {
						tokens = append(tokens, token.Heading{Leaf: cst})
					} else {
						tokens = append(tokens, token.Text{Leaf: cst})
					}
				})

			case '*':
				p.ParseLeaf(root, func(cst *cst.Leaf) {
					if r := p.PeekN(1); unicode.IsSpace(r) || r == utf8.RuneError {
						// No text following the '*'? Then treat as bullet.
						p.Advance()
						tokens = append(tokens, token.Bullet{Leaf: cst})
						return
					}

					for p.Rune('*') {
					}
					tokens = append(tokens, token.Emphasis{Leaf: cst})
				})

			case '_':
				p.ParseLeaf(root, func(cst *cst.Leaf) {
					if r := p.PeekN(1); unicode.IsSpace(r) || r == utf8.RuneError {
						// No text following the '_'? Then treat as text.
						p.Advance()
						tokens = append(tokens, token.Text{Leaf: cst})
						return
					}

					for p.Rune('_') {
					}
					tokens = append(tokens, token.Emphasis{Leaf: cst})
				})

			case '{':
				p.ParseLeaf(root, func(cst *cst.Leaf) {
					if ok, typed := parseTag(p); ok {
						tokens = append(tokens, token.Tag{Leaf: cst, Typed: typed})
					} else {
						p.Advance()
						tokens = append(tokens, token.OpenBracket{Leaf: cst})
					}
				})

			case '[', '(':
				p.ParseLeaf(root, func(cst *cst.Leaf) {
					p.Advance()
					tokens = append(tokens, token.OpenBracket{Leaf: cst})
				})

			case ']', ')', '}':
				p.ParseLeaf(root, func(cst *cst.Leaf) {
					p.Advance()
					tokens = append(tokens, token.CloseBracket{Leaf: cst})
				})

			default: // assume text
				trailingEmphasis := false
				p.ParseLeaf(root, func(cst *cst.Leaf) {
					parseText(p)
					tokens = append(tokens, token.Text{Leaf: cst})
					if r := p.Peek(); r == '_' || r == '*' {
						trailingEmphasis = true
					}
				})

				// Trailing _ and * should be treated as emphasis, not text or bullets.
				if trailingEmphasis {
					for r := p.Peek(); r == '_' || r == '*'; r = p.Peek() {
						p.ParseLeaf(root, func(cst *cst.Leaf) {
							p.Advance()
							for p.Rune(r) {
							}
							tokens = append(tokens, token.Emphasis{Leaf: cst})
						})
					}
				}
			}
		}
	}
	errors := parse.Parse(filename, data, skip, parser)
	return tokens, errors
}

func parseText(p *parse.Parser) {
	for {
		r := p.Peek()
		switch {
		case unicode.IsSpace(r),
			r == '*', r == '\\',
			r == '[', r == ']',
			r == '(', r == ')',
			r == '{', r == '}',
			r == utf8.RuneError:
			return // End of text.
		case r == '_':
			// Is this ' foo_bar ', or ' foobar_ ' ?
			// Look ahead to see if this is inside the text, or an
			// end-emphasis.
			for i, done := 1, false; !done; i++ {
				r := p.PeekN(i)
				switch {
				case unicode.IsSpace(r), r == '*', r == utf8.RuneError:
					return
				case r == '_':
				default:
					p.AdvanceN(i)
					done = true
				}
			}
		default:
			p.Advance()
		}
	}
}

func parseTag(p *parse.Parser) (bool, bool) {
	if p.PeekN(1) != '{' {
		return false, false
	}
	typed := false
	for i := 2; ; i++ {
		r := p.PeekN(i)
		if !unicode.IsNumber(r) && !unicode.IsLetter(r) && r != '_' {
			// Possible type declaration
			if r == ':' {
				typed = true
				continue
			}
			// End of tag
			if r == '}' || p.PeekN(i+1) == '}' {
				p.AdvanceN(i + 2)
				return true, typed
			}
			return false, false
		}
	}
}
