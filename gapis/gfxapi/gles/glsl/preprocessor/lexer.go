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

package preprocessor

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/gapis/gfxapi/gles/glsl/ast"
)

type lexer struct {
	err     ast.ErrorCollector
	reader  *parse.Reader
	next    TokenInfo
	current TokenInfo
}

func (l *lexer) makeTokenInfo(t Token) {
	l.current.Token = t
	l.current.Cst.SetToken(l.reader.Consume())

	l.next = l.current
	l.current = TokenInfo{Cst: &parse.Leaf{}}

	l.next.Cst.AddSuffix(l.skip(parse.SkipSuffix))

	whitespace := l.skip(parse.SkipPrefix)
	if l.reader.IsEOF() {
		l.next.Cst.AddSuffix(whitespace)
	} else {
		l.current.Cst.AddPrefix(whitespace)
	}
}

func (l *lexer) readDirective() bool {
	if !l.current.Newline {
		// Preprocessor directives must be at the start of a line.
		return false
	}
	for _, key := range ppKeywords {
		if !l.reader.Rune('#') {
			// Preprocessor directives must start with hash.
			return false
		}
		// White-space is allowed between the hash and the keyword.
		l.reader.Space()
		if l.reader.String(key.String()[1:]) && !l.reader.AlphaNumeric() {
			l.makeTokenInfo(key)
			return true
		}
		l.reader.Rollback()
	}
	return false
}

func (l *lexer) readIndentKeywordType() bool {
	if !l.reader.AlphaNumeric() {
		return false
	}
	alpha := l.reader.Token().String()

	// check for keywords
	for _, key := range keywords {
		if alpha == key.String() {
			l.makeTokenInfo(key)
			return true
		}
	}

	// check for types
	for _, t := range ast.BareTypes {
		if alpha == t.String() {
			l.makeTokenInfo(t)
			return true
		}
	}

	// it's an identifier
	l.makeTokenInfo(Identifier(alpha))
	return true
}

func (l *lexer) readNumber() bool {
	switch l.reader.Numeric() {
	case parse.Floating, parse.Scientific:
		str := strings.TrimRight(l.reader.Token().String(), "fF")
		f, err := strconv.ParseFloat(str, 64)
		if err == nil {
			l.makeTokenInfo(ast.FloatValue(f))
			return true
		}
	case parse.Decimal, parse.Octal, parse.Hexadecimal:
		str := l.reader.Token().String()
		last := len(str) - 1
		if str[last] == 'u' || str[last] == 'U' {
			str = str[0:last]
			i, err := strconv.ParseUint(str, 0, 64)
			if err == nil {
				l.makeTokenInfo(ast.UintValue(i))
				return true
			}
		} else {
			i, err := strconv.ParseInt(str, 0, 64)
			if err == nil {
				l.makeTokenInfo(ast.IntValue(i))
				return true
			}
		}
	}
	l.reader.Rollback()
	return false
}

func (l *lexer) readOperator() bool {
	for _, op := range operators {
		if l.reader.String(op.String()) {
			l.makeTokenInfo(op)
			return true
		}
	}
	for _, op := range ast.Operators {
		if l.reader.String(op.String()) {
			l.makeTokenInfo(op)
			return true
		}
	}
	return false
}

// read performs the lexical analysis of the input and returns a single token for processing.
// This includes preprocessor directives and error tokens.
func (l *lexer) read() {
	if l.reader.IsEOF() {
		l.makeTokenInfo(nil)
		return
	}

	if l.readDirective() {
		return
	}
	if l.readIndentKeywordType() {
		return
	}
	if l.readNumber() {
		return
	}
	if l.readOperator() {
		return
	}

	l.reader.Advance()
	l.err.Errorf("Unknown token in the input stream: %q", l.reader.Consume().String())
	l.read()
}

const (
	lineComment       string = "//"
	blockCommentStart        = "/*"
	blockCommentEnd          = "*/"
)

// A skip function for the lexer. It skips all the comments and whitespace and sets
// the Whitespace and Newline of the current token.
func (l *lexer) skip(mode parse.SkipMode) parse.Separator {
	var sep parse.Separator
	for {
		switch {
		case l.reader.Space():
			l.current.Whitespace = true
			n := parse.NewFragment(l.reader.Consume())
			sep = append(sep, n)
		case mode == parse.SkipPrefix && l.reader.EOL():
			l.current.Whitespace = true
			l.current.Newline = true
			n := parse.NewFragment(l.reader.Consume())
			sep = append(sep, n)
		case l.reader.String(lineComment):
			l.current.Whitespace = true
			if !l.reader.SeekRune('\n') {
				for !l.reader.IsEOF() {
					l.reader.Advance()
				}
			}
			n := parse.NewFragment(l.reader.Consume())
			sep = append(sep, n)
		case l.reader.String(blockCommentStart):
			l.current.Whitespace = true
			first, _ := utf8.DecodeRuneInString(blockCommentEnd)
			for {
				if !l.reader.SeekRune(first) {
					for !l.reader.IsEOF() {
						l.reader.Advance()
					}
					l.err.Errorf("Unterminated block comment")
					break
				}
				if l.reader.String(blockCommentEnd) {
					break
				}
				l.reader.Advance()
			}
			n := parse.NewFragment(l.reader.Consume())
			sep = append(sep, n)
		default:
			return sep
		}
	}
}

func processLineContinuations(input string) string {
	return strings.Replace(input, "\\\n", "", -1)
}

/////////////////////////// Lexer interface below /////////////////////////////

func newLexer(filename, input string) *lexer {
	input = processLineContinuations(input)
	l := &lexer{
		current: TokenInfo{Newline: true, Cst: &parse.Leaf{}},
		reader:  parse.NewReader(filename, input),
	}
	l.current.Cst.AddPrefix(l.skip(parse.SkipPrefix))
	l.read()
	return l
}

func (l *lexer) Next() (ti TokenInfo) {
	ti = l.next
	l.read()
	return
}

func (l *lexer) Peek() TokenInfo {
	return l.next
}
