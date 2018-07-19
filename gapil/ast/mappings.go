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

import "github.com/google/gapid/core/text/parse/cst"

// Mappings is a two-way map of AST nodes to semantic nodes.
type Mappings struct {
	ASTToCST map[Node]cst.Node
	CSTToAST map[cst.Node]Node
}

func (m *Mappings) init() {
	if m.ASTToCST == nil {
		*m = Mappings{
			ASTToCST: map[Node]cst.Node{},
			CSTToAST: map[cst.Node]Node{},
		}
	}
}

// Add creates an association between the AST and CST nodes.
func (m *Mappings) Add(a Node, c cst.Node) {
	m.init()
	m.ASTToCST[a] = c
	m.CSTToAST[c] = a
}

// CST returns the CST node for the given AST node.
func (m *Mappings) CST(ast Node) cst.Node {
	return m.ASTToCST[ast]
}

// MergeIn merges the mappings in other into m.
func (m *Mappings) MergeIn(other *Mappings) {
	m.init()
	for a, c := range other.ASTToCST {
		m.ASTToCST[a] = c
	}
	for c, a := range other.CSTToAST {
		m.CSTToAST[c] = a
	}
}
