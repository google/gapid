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

// Package parser implements a parser for converting the api language into
// abstract syntax trees.
package parser

import (
	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/core/text/parse/cst"
	"github.com/google/gapid/gapil/ast"
)

// ParseMap is the interface to an object into which ast<->cts mappings are stored.
type ParseMap interface {
	// The map object passed to parsers must support the interface used by the
	// parsing library.
	cst.Map
}

// NewParseMap returns a simple implementation of ParseMap sufficient for basic
// mapping use cases.
func NewParseMap() ParseMap {
	return cst.NewMap()
}

// Parse takes a string containing a complete api description and
// returns the abstract syntax tree representation of it.
// If the string is not syntactically valid, it will also return the
// errors encountered. If errors are returned, the ast returned will be
// the incomplete tree so far, and may not be structurally valid.
func Parse(filename, data string, m ParseMap) (*ast.API, parse.ErrorList) {
	var api *ast.API
	parser := func(p *parse.Parser, b *cst.Branch) {
		api = requireAPI(p, b)
	}
	errors := parse.Parse(parser, filename, data, parse.NewSkip("//", "/*", "*/"), m)
	return api, errors
}
