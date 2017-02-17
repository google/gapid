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

// Package token holds the tokens generated my the minidown scanner.
package token

import "github.com/google/gapid/core/text/parse"

// Token is the interface implemented by all tokens.
type Token interface {
	CST() parse.Node
}

// Heading represents a run of '#'
type Heading struct {
	*parse.Leaf
}

// NewLine represents a newline.
type NewLine struct {
	*parse.Leaf
}

// Emphasis represets a '*', '_', '**' or '__' adjacent text.
type Emphasis struct {
	*parse.Leaf
}

// Bullet represets a '*' non-adjacent to text.
type Bullet struct {
	*parse.Leaf
}

// Text represents regular text.
type Text struct {
	*parse.Leaf
	Override string // If non-empty then the this should be used instead of the CST string.
}

func (t Text) String() string {
	if len(t.Override) > 0 {
		return t.Override
	}
	return t.CST().Token().String()
}

// Tag represents a alpha-numeric wrapped with double curly brackets.
// For example '{{person}}'.
type Tag struct {
	*parse.Leaf
	Typed bool
}

// OpenBracket represents a '(', '[' or '{'.
type OpenBracket struct {
	*parse.Leaf
}

// Is returns true if the bracket is of the type r.
func (t OpenBracket) Is(r rune) bool { return t.Token().String() == string([]rune{r}) }

// CloseBracket represents a ')', ']' or '}'.
type CloseBracket struct {
	*parse.Leaf
}

// Is returns true if the bracket is of the type r.
func (t CloseBracket) Is(r rune) bool { return t.Token().String() == string([]rune{r}) }

// CST returns the parse.Node of this token.
func (t Heading) CST() parse.Node { return t.Leaf }

// CST returns the parse.Node of this token.
func (t NewLine) CST() parse.Node { return t.Leaf }

// CST returns the parse.Node of this token.
func (t Emphasis) CST() parse.Node { return t.Leaf }

// CST returns the parse.Node of this token.
func (t Bullet) CST() parse.Node { return t.Leaf }

// CST returns the parse.Node of this token.
func (t Text) CST() parse.Node { return t.Leaf }

// CST returns the parse.Node of this token.
func (t Tag) CST() parse.Node { return t.Leaf }

// CST returns the parse.Node of this token.
func (t OpenBracket) CST() parse.Node { return t.Leaf }

// CST returns the parse.Node of this token.
func (t CloseBracket) CST() parse.Node { return t.Leaf }
