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

package cst

import (
	"fmt"
	"io"

	"github.com/google/gapid/core/data/compare"
)

// A Token represents the smallest consumed unit input.
type Token struct {
	Source *Source // The source object this token is from (including the full rune array).
	Start  int     // The start of the token in the full rune array.
	End    int     // One past the end of the token.
}

// Tok returns t.
func (t Token) Tok() Token {
	return t
}

// Write writes the token to the writer.
func (t Token) Write(w io.Writer) error {
	_, err := io.WriteString(w, t.String())
	return err
}

// Less returns true if t comes before rhs when considering source files and
// offsets within the same file.
func (t Token) Less(rhs Token) bool {
	if lhs, rhs := t.Source, rhs.Source; lhs != nil && rhs != nil {
		switch {
		case lhs.Filename < rhs.Filename:
			return true
		case rhs.Filename < lhs.Filename:
			return false
		}
	}
	return t.Start < rhs.Start
}

// At returns the token's location in the form:
// filename:line:column
func (t Token) At() string {
	file := t.Source.RelativeFilename()
	line, column := t.Cursor()
	return fmt.Sprintf("%v:%v:%v", file, line, column)
}

// Format implements fmt.Formatter writing the start end and value of the token.
func (t Token) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "%d:%d:%s", t.Start, t.End, t.String())
}

// String returns the string form of the rune range the token represents.
func (t Token) String() string {
	if t.Start >= t.End || len(t.Source.Runes) == 0 {
		return ""
	}
	return string(t.Source.Runes[t.Start:t.End])
}

// Cursor is used to calculate the line and column of the start of the token.
// It may be very expensive to call, and is intended to be used sparingly in
// producing human readable error messages only.
func (t Token) Cursor() (line int, column int) {
	line = 1
	column = 1
	for _, r := range t.Source.Runes[:t.Start] {
		if r == '\n' {
			line++
			column = 0
		}
		column++
	}
	return line, column
}

// Len returns the length of the token in runes.
func (t Token) Len() int {
	return t.End - t.Start
}

func compareTokens(c compare.Comparator, reference, value Token) {
	if reference.String() != value.String() {
		c.AddDiff(reference, value)
	}
}

func init() {
	compare.Register(compareTokens)
}
