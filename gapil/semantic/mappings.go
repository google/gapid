// Copyright (C) 2018 Google Inc.
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

package semantic

import (
	"github.com/google/gapid/core/text/parse/cst"
	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/parser"
)

// Mappings is a two-way relational map of AST nodes to semantic nodes.
type Mappings struct {
	parser.ParseMap
	ASTToSemantic map[ast.Node][]Node
	SemanticToAST map[Node][]ast.Node
}

// NewMappings returns a new, initialized Mappings struct.
func NewMappings() *Mappings {
	return &Mappings{
		ParseMap:      parser.NewParseMap(),
		ASTToSemantic: map[ast.Node][]Node{},
		SemanticToAST: map[Node][]ast.Node{},
	}
}

// Add creates a binding between the AST and semantic nodes.
func (m *Mappings) Add(ast ast.Node, sem Node) {
	m.ASTToSemantic[ast] = append(m.ASTToSemantic[ast], sem)
	m.SemanticToAST[sem] = append(m.SemanticToAST[sem], ast)
}

// ParseNode returns the primary parse node for the semantic node.
// If the semantic node has no parse node then nil is returned.
func (m *Mappings) ParseNode(sem Node) cst.Node {
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
