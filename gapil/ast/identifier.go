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

package ast

// Identifier holds a parsed identifier in the parse tree.
type Identifier struct {
	Value string // the identifier
}

func (Identifier) isNode() {}

// Generic represents a identifier modified by type arguments. It looks like:
// «identifier ! ( arg | <arg {, arg} )>»
type Generic struct {
	Name      *Identifier // the generic identifier.
	Arguments []Node      // the type arguments to the generic.
}

func (Generic) isNode() {}

const (
	// Keyword strings represent places in the syntax where a word has special
	// meaning.
	KeywordAbort     = "abort"
	KeywordAPI       = "api"
	KeywordAlias     = "alias"
	KeywordBitfield  = "bitfield"
	KeywordCase      = "case"
	KeywordClass     = "class"
	KeywordCmd       = "cmd"
	KeywordConst     = "const"
	KeywordDefault   = "default"
	KeywordDefine    = "define"
	KeywordDelete    = "delete"
	KeywordClear     = "clear"
	KeywordElse      = "else"
	KeywordEnum      = "enum"
	KeywordExtern    = "extern"
	KeywordFalse     = "false"
	KeywordFence     = "fence"
	KeywordFor       = "for"
	KeywordIf        = "if"
	KeywordImport    = "import"
	KeywordIn        = "in"
	KeywordLabel     = "label"
	KeywordNull      = "null"
	KeywordReturn    = "return"
	KeywordPseudonym = "type"
	KeywordSwitch    = "switch"
	KeywordSub       = "sub"
	KeywordThis      = "this"
	KeywordTrue      = "true"
	KeywordWhen      = "when"
	KeywordApiIndex  = "api_index"
)
