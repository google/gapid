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

// Package resolver implements a semantic resolving for the api language.
// It is responsible for converting from an abstract syntax tree to a typed
// semantic graph ready for code generation.
package resolver

import (
	"fmt"

	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/parser"
	"github.com/google/gapid/gapil/semantic"
)

// Mappings is a two-way relational map of AST nodes to semantic nodes.
type Mappings struct {
	parser.ParseMap
	ASTToSemantic map[ast.Node][]semantic.Node
	SemanticToAST map[semantic.Node][]ast.Node
}

// NewMappings returns a new, initialized Mappings struct.
func NewMappings() *Mappings {
	return &Mappings{
		ParseMap:      parser.NewParseMap(),
		ASTToSemantic: map[ast.Node][]semantic.Node{},
		SemanticToAST: map[semantic.Node][]ast.Node{},
	}
}

func (m *Mappings) add(ast ast.Node, sem semantic.Node) {
	m.ASTToSemantic[ast] = append(m.ASTToSemantic[ast], sem)
	m.SemanticToAST[sem] = append(m.SemanticToAST[sem], ast)
}

// ParseNode returns the primary parse node for the semantic node.
// If the semantic node has no parse node then nil is returned.
func (m *Mappings) ParseNode(sem semantic.Node) parse.Node {
	if asts, ok := m.SemanticToAST[sem]; ok && len(asts) > 0 {
		if cst := m.CST(asts[0]); cst != nil {
			return cst
		}
	}
	return nil
}

// MergeIn merges the mappings in other into m.
func (m *Mappings) MergeIn(other *Mappings) {
	for a, s := range other.ASTToSemantic {
		m.ASTToSemantic[a] = append(m.ASTToSemantic[a], s...)
	}
	for s, a := range other.SemanticToAST {
		m.SemanticToAST[s] = append(m.SemanticToAST[s], a...)
	}
}

// Resolve takes valid asts as produced by the parser and converts them to the
// semantic graph form.
// If the asts are not fully valid (ie there were parse errors) then the results
// are undefined.
// If there are semantic problems with the ast, Resolve will return the set of
// errors it finds, and the returned graph may be incomplete/invalid.
func Resolve(includes []*ast.API, mappings *Mappings, options Options) (*semantic.API, parse.ErrorList) {
	rv := &resolver{
		api: &semantic.API{},
		scope: &scope{
			types: map[string]semantic.Type{},
		},
		mappings:           mappings,
		genericSubroutines: map[string]genericSubroutine{},
		options:            options,
	}
	func() {
		defer func() {
			err := recover()
			if err != nil && err != parse.AbortParse {
				if len(rv.errors) != 0 {
					panic(fmt.Errorf("Panic: %v\nErrors: %v", err, rv.errors))
				} else {
					panic(err)
				}
			}
		}()
		// Register all the built in symbols
		for _, t := range semantic.BuiltinTypes {
			rv.addType(t)
		}
		rv.with(semantic.VoidType, func() {
			for _, api := range includes {
				apiNames(rv, api)
				rv.mappings.add(api, rv.api)
			}
			resolve(rv)
		})
	}()

	return rv.api, rv.errors
}
