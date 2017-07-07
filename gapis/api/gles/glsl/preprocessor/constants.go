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
	"fmt"

	"github.com/google/gapid/core/text/parse"
)

// An interface for all the tokens returned by the preprocessor.
type Token interface {
	fmt.Stringer
}

// Identifier represents a language identifier returned by the lexer.
type Identifier string

func (id Identifier) String() string { return string(id) }

// TokenInfo contains whitespace information about the contained token.
type TokenInfo struct {
	Token      Token       // The token.
	Whitespace bool        // Whether this token was preceeded by any whitespace.
	Newline    bool        // Whether this token was preceeded by a newline character.
	Cst        *parse.Leaf // Structure containing the preceeding and following whitespace.
}

// Keyword represents a language keyword token returned by the preprocessor.
type Keyword struct {
	word *string
}

func (key Keyword) String() string { return *key.word }

// Keywords is an array of all the keywords of the language.
var keywords []Token

func appendKeyword(word string) Keyword {
	key := Keyword{&word}
	keywords = append(keywords, key)
	return key
}

var qualifierKeywords = map[Token]struct{}{}

func appendQualifier(word string) Keyword {
	key := appendKeyword(word)
	qualifierKeywords[key] = struct{}{}
	return key
}

// Is token a qualifier keyword?
func IsQualifier(token Token) bool { _, present := qualifierKeywords[token]; return present }

// All the non-type keywords in the language, one variable per keyword.
var (
	KwCentroid  = appendQualifier("centroid")
	KwConst     = appendQualifier("const")
	KwFlat      = appendQualifier("flat")
	KwIn        = appendQualifier("in")
	KwInvariant = appendQualifier("invariant")
	KwLayout    = appendQualifier("layout")
	KwOut       = appendQualifier("out")
	KwSmooth    = appendQualifier("smooth")
	KwUniform   = appendQualifier("uniform")
	KwAttribute = appendQualifier("attribute") // 1.0
	KwVarying   = appendQualifier("varying")   // 1.0

	KwBreak     = appendKeyword("break")
	KwCase      = appendKeyword("case")
	KwContinue  = appendKeyword("continue")
	KwDefault   = appendKeyword("default")
	KwDiscard   = appendKeyword("discard")
	KwDo        = appendKeyword("do")
	KwElse      = appendKeyword("else")
	KwFalse     = appendKeyword("false")
	KwFor       = appendKeyword("for")
	KwIf        = appendKeyword("if")
	KwInout     = appendKeyword("inout")
	KwLowp      = appendKeyword("lowp")
	KwMediump   = appendKeyword("mediump")
	KwHighp     = appendKeyword("highp")
	KwPrecision = appendKeyword("precision")
	KwReturn    = appendKeyword("return")
	KwStruct    = appendKeyword("struct")
	KwSwitch    = appendKeyword("switch")
	KwTrue      = appendKeyword("true")
	KwWhile     = appendKeyword("while")
)

var precisionKeywords = map[Token]struct{}{
	KwHighp:   {},
	KwMediump: {},
	KwLowp:    {},
}

// Is token a precision keyword?
func IsPrecision(token Token) bool { _, present := precisionKeywords[token]; return present }

// Operator represents an operator returned by the preprocessor.
type Operator struct {
	op *string
}

func (op Operator) String() string { return *op.op }

// operators is an array of all the operators of the language.
var operators []Token

func appendOperator(opname string) Operator {
	op := Operator{&opname}
	operators = append(operators, op)
	return op
}

// Operator tokens not defined in the ast package.
var (
	OpColon     = appendOperator(":")
	OpDot       = appendOperator(".")
	OpLBrace    = appendOperator("{")
	OpLBracket  = appendOperator("[")
	OpLParen    = appendOperator("(")
	OpQuestion  = appendOperator("?")
	OpRBrace    = appendOperator("}")
	OpRBracket  = appendOperator("]")
	OpRParen    = appendOperator(")")
	OpSemicolon = appendOperator(";")
	OpGlue      = appendOperator("##")
)

// ppKeyword represents preprocessor directives. These are processed by the preprocessor
// internally and not passed on to the parser.
type ppKeyword Keyword

func (key ppKeyword) String() string { return Keyword(key).String() }

// ppKeywords is an array of all the preprocessor directives of the language.
var ppKeywords []ppKeyword

func appendPPKeyword(word string) (key ppKeyword) {
	key.word = &word
	ppKeywords = append(ppKeywords, key)
	return
}

// All preprocessor directives of the language, one variable per directive.
var (
	ppDefine    = appendPPKeyword("#define")
	ppUndef     = appendPPKeyword("#undef")
	ppIf        = appendPPKeyword("#if")
	ppIfdef     = appendPPKeyword("#ifdef")
	ppIfndef    = appendPPKeyword("#ifndef")
	ppElse      = appendPPKeyword("#else")
	ppElif      = appendPPKeyword("#elif")
	ppEndif     = appendPPKeyword("#endif")
	ppError     = appendPPKeyword("#error")
	ppPragma    = appendPPKeyword("#pragma")
	ppExtension = appendPPKeyword("#extension")
	ppLine      = appendPPKeyword("#line")
	ppVersion   = appendPPKeyword("#version")
	ppEmpty     = appendPPKeyword("#")
)
