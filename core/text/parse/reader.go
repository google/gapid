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
	"bytes"
	"unicode"
	"unicode/utf8"

	"github.com/google/gapid/core/text/parse/cst"
)

// Reader is the interface to an object that converts a rune array into tokens.
type Reader struct {
	Source *cst.Source // The source being parsed.
	runes  []rune      // The string being parsed.
	offset int         // The start of the current token.
	cursor int         // The offset of the next unparsed rune.
}

func (r *Reader) setData(filename string, data string) {
	r.runes = bytes.Runes([]byte(data))
	r.Source = &cst.Source{Filename: filename, Runes: r.runes}
}

// Token peeks at the current scanned token value. It does not consume anything.
func (r *Reader) Token() cst.Token {
	return cst.Token{Source: r.Source, Start: r.offset, End: r.cursor}
}

// Consume consumes the current token.
func (r *Reader) Consume() cst.Token {
	tok := r.Token()
	r.offset = r.cursor
	return tok
}

// Advance moves the cursor one rune forward.
func (r *Reader) Advance() {
	if r.cursor < len(r.runes) {
		r.cursor++
	}
}

// AdvanceN moves the cursor n runes forward.
func (r *Reader) AdvanceN(n int) {
	if r.cursor+n < len(r.runes) {
		r.cursor += n
	} else {
		r.cursor = len(r.runes)
	}
}

// Rollback sets the cursor back to the last consume point.
func (r *Reader) Rollback() {
	r.cursor = r.offset
}

// IsEOF returns true when the cursor is at the end of the input.
func (r *Reader) IsEOF() bool {
	return r.cursor >= len(r.runes)
}

// IsEOL returns true when the cursor is at the end of a line (\n or \r\n).
// IsEOL does not move the cursor.
func (r *Reader) IsEOL() bool {
	return (r.PeekN(0) == '\n') || (r.PeekN(0) == '\r' && r.PeekN(1) == '\n')
}

// EOL moves the cursor past the EOL and returns true if the cursor is at the
// end of a line (\n or \r\n), otherwise EOL returns false and does nothing.
func (r *Reader) EOL() bool {
	if !r.IsEOL() {
		return false
	}
	r.Rune('\r')
	r.Rune('\n')
	return true
}

// GuessNextToken attempts to do a general purpose consume of a single
// arbitrary token from the stream. It is used by error handlers to indicate
// where the error occurred. It guarantees that if the stream is not finished,
// it will consume at least one character.
func (r *Reader) GuessNextToken() cst.Token {
	if r.cursor == r.offset {
		r.Space()
		switch {
		case r.Peek() == '}': // Don't consume end-brackets on syntax errors.
		case r.AlphaNumeric():
		case r.Numeric() != NotNumeric:
		case r.NotSpace():
		default:
			r.Advance()
		}
	}
	return r.Consume()
}

// Peek returns the next rune without advancing the cursor.
func (r *Reader) Peek() rune {
	return r.PeekN(0)
}

// PeekN returns the n'th next rune without advancing the cursor.
func (r *Reader) PeekN(n int) rune {
	if r.cursor+n >= len(r.runes) {
		return utf8.RuneError
	}
	return r.runes[r.cursor+n]
}

// Rune advances and returns true if the next rune after the cursor matches value.
func (r *Reader) Rune(value rune) bool {
	if r.cursor >= len(r.runes) || r.runes[r.cursor] != value {
		return false
	}
	r.cursor++
	return true
}

// SeekRune advances the cursor until either the value is found or the end
// of stream is reached.
// It returns true if it found value, false otherwise.
func (r *Reader) SeekRune(value rune) bool {
	for i := r.cursor; i < len(r.runes); i++ {
		if r.runes[i] == value {
			r.cursor = i
			return true
		}
	}
	return false
}

// String checks to see if value occurs at cursor, if it does, it advances the
// cursor past it and returns true.
func (r *Reader) String(value string) bool {
	end := r.cursor + len(value)
	if end > len(r.runes) {
		return false
	}
	for i, v := range value {
		if r.runes[r.cursor+i] != v {
			return false
		}
	}
	r.cursor = end
	return true
}

// Space skips over any whitespace, returning true if it advanced the cursor.
func (r *Reader) Space() bool {
	i := r.cursor
	for ; i < len(r.runes); i++ {
		r := r.runes[i]
		if r == '\n' || !unicode.IsSpace(r) {
			break
		}
	}
	if i == r.cursor {
		return false
	}
	r.cursor = i
	return true
}

// NotSpace skips over any non whitespace, returning true if it advanced the cursor.
func (r *Reader) NotSpace() bool {
	i := r.cursor
	for ; i < len(r.runes); i++ {
		if unicode.IsSpace(r.runes[i]) {
			break
		}
	}
	if i == r.cursor {
		return false
	}
	r.cursor = i
	return true
}

// NumberKind is a type used by Reader.Numeric for identifying various kinds of numbers.
type NumberKind uint8

const (
	// No number was found.
	NotNumeric NumberKind = iota
	// A decimal number.
	Decimal
	// An octal number, starting with "0". PS: A lone "0" is classified as octal.
	Octal
	// A hexadecimal number, starting with "0x".
	Hexadecimal
	// A floating point number: "123.456". Whole and the fractional parts are optional (but
	// not both at the same time).
	Floating
	// A floating point number in scientific notation: "123.456e±789". The fractional part,
	// the dot and the exponent sign are all optional.
	Scientific

	atDot   // Internally used to represent the state after reading ".".
	atE     // Internally used to represent the state after reading "e".
	atESign // Internally used to represent the state after reading "e±".
)

// Numeric tries to move past the common number pattern. It returns a constant of type NumberKind
// describing the kind of number it found.
func (r *Reader) Numeric() NumberKind {
	state := NotNumeric
	i := r.cursor
	for {
		var next = '?'
		if i < len(r.runes) {
			next = unicode.ToLower(r.runes[i])
		}
		i++
		switch state {
		case NotNumeric:
			switch {
			case next == '0':
				state = Octal
			case next >= '1' && next <= '9':
				state = Decimal
			case next == '.':
				state = atDot
			default:
				return NotNumeric // We have read nothing
			}
		case Decimal:
			switch {
			case next >= '0' && next <= '9': // do nothing
			case next == '.':
				state = atDot
			case next == 'e':
				state = atE
			case next == 'u':
				r.cursor = i
				return Decimal
			default:
				r.cursor = i - 1
				return Decimal
			}
		case Octal:
			switch {
			case next >= '0' && next <= '7': // do nothing
			case next == 'x':
				state = Hexadecimal
			case next == 'u':
				r.cursor = i
				return Octal
			case next == '.' && i == r.cursor+2: // We have read "0."
				state = atDot
			default:
				r.cursor = i - 1
				return Octal
			}
		case Hexadecimal:
			switch {
			case (next >= '0' && next <= '9') || (next >= 'a' && next <= 'f'): // do nothing
			case next == 'u':
				r.cursor = i
				return Hexadecimal
			default:
				r.cursor = i - 1
				return Hexadecimal
			}
		case atDot:
			switch {
			case next >= '0' && next <= '9':
				state = Floating
			case next == 'f', next == 'F':
				r.cursor = i
				return Floating
			case i > r.cursor+2: // There is at least one digit before the dot.
				if next == 'e' {
					state = atE
				} else {
					r.cursor = i - 1
					return Floating
				}
			default:
				return NotNumeric // We have only read ".". This is bad.
			}
		case Floating:
			switch {
			case next >= '0' && next <= '9': // do nothing
			case next == 'f', next == 'F':
				r.cursor = i
				return Floating
			case next == 'e':
				state = atE
			default:
				r.cursor = i - 1
				return Floating
			}
		case atE:
			switch {
			case next >= '0' && next <= '9':
				state = Scientific
			case next == '+' || next == '-':
				state = atESign
			default:
				return NotNumeric // We need at least one digit after "e"
			}
		case atESign:
			switch {
			case next >= '0' && next <= '9':
				state = Scientific
			default:
				return NotNumeric // We need at least one digit after "e±"
			}
		case Scientific:
			switch {
			case next >= '0' && next <= '9': // do nothing
			default:
				r.cursor = i - 1
				return Scientific
			}
		}
	}
}

// AlphaNumeric moves past anything that starts with a letter or underscore,
// and consists of letters, numbers or underscores. It returns true if the
// pattern was matched, false otherwise.
func (r *Reader) AlphaNumeric() bool {
	i := r.cursor
	if i >= len(r.runes) {
		return false
	}
	next := r.runes[r.cursor]
	if next == '_' || unicode.IsLetter(next) {
		for i++; i < len(r.runes); i++ {
			next := r.runes[i]
			if next != '_' && !unicode.IsLetter(next) && !unicode.IsDigit(next) {
				break
			}
		}
	}
	if i == r.cursor {
		return false
	}
	r.cursor = i
	return true
}

// NewReader creates a new reader which reads from the supplied string.
func NewReader(filename string, data string) *Reader {
	r := &Reader{}
	r.setData(filename, data)
	return r
}
