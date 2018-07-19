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
	"github.com/google/gapid/core/data/slice"
	"github.com/google/gapid/core/text/parse/cst"
	"github.com/google/gapid/gapil/ast"
)

// Mappings is a two-way map of AST nodes to semantic nodes.
type Mappings struct {
	AST           ast.Mappings
	ASTToSemantic map[ast.Node][]Node
	SemanticToAST map[Node][]ast.Node
}

func (m *Mappings) init() {
	if m.ASTToSemantic == nil {
		m.ASTToSemantic = map[ast.Node][]Node{}
		m.SemanticToAST = map[Node][]ast.Node{}
	}
}

// Add creates an association between the AST and semantic nodes.
func (m *Mappings) Add(a ast.Node, s Node) {
	m.init()
	m.ASTToSemantic[a] = append(m.ASTToSemantic[a], s)
	m.SemanticToAST[s] = append(m.SemanticToAST[s], a)
}

// Remove removes the semantic node from the mappings.
func (m *Mappings) Remove(s Node) {
	for _, a := range m.SemanticToAST[s] {
		l := m.ASTToSemantic[a]
		slice.Remove(&l, s)
		if len(l) > 0 {
			m.ASTToSemantic[a] = l
		} else {
			delete(m.ASTToSemantic, a)
		}
	}
	delete(m.SemanticToAST, s)
}

// CST returns the primary CST for the semantic node.
// If the semantic node has no associated CST then nil is returned.
func (m *Mappings) CST(sem Node) cst.Node {
	if asts, ok := m.SemanticToAST[sem]; ok && len(asts) > 0 {
		if cst := m.AST.CST(asts[0]); cst != nil {
			return cst
		}
	}
	return nil
}

// MergeIn merges the mappings in other into m.
func (m *Mappings) MergeIn(other *Mappings) {
	m.init()
	m.AST.MergeIn(&other.AST)
	for a, s := range other.ASTToSemantic {
		m.ASTToSemantic[a] = append(m.ASTToSemantic[a], s...)
	}
	for s, a := range other.SemanticToAST {
		m.SemanticToAST[s] = append(m.SemanticToAST[s], a...)
	}
}
