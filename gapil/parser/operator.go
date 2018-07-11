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

package parser

import (
	"unicode"
	"unicode/utf8"

	"github.com/google/gapid/core/text/parse/cst"
	"github.com/google/gapid/gapil/ast"
)

func (p *parser) peekOperator(op string) bool {
	scanned := p.scanOperator()
	p.Rollback()
	return op == scanned
}

func (p *parser) operator(op string, b *cst.Branch) bool {
	scanned := p.scanOperator()
	if op != scanned {
		p.Rollback()
		return false
	}
	p.ParseLeaf(b, nil)
	return true
}

func (p *parser) requireOperator(op string, b *cst.Branch) {
	if !p.operator(op, b) {
		p.Expected(string(op))
	}
}

func (p *parser) scanOperator() string {
	for _, op := range ast.Operators {
		if p.String(string(op)) {
			r, _ := utf8.DecodeLastRuneInString(string(op))
			if !unicode.IsLetter(r) || !unicode.IsLetter(p.Peek()) {
				return op
			}
		}
	}
	return ""
}

// lhs operator expression
func (p *parser) binaryOp(lhs ast.Node) *ast.BinaryOp {
	op := p.scanOperator()
	if _, found := ast.BinaryOperators[op]; !found {
		p.Rollback()
		return nil
	}
	n := &ast.BinaryOp{LHS: lhs, Operator: op}
	p.Extend(p.mappings.CST(lhs), func(b *cst.Branch) {
		p.mappings.Add(n, b)
		p.ParseLeaf(b, nil)
		n.RHS = p.requireExpression(b)
	})
	return n
}

// operator expression
func (p *parser) unaryOp(b *cst.Branch) *ast.UnaryOp {
	op := p.scanOperator()
	p.Rollback()
	if _, found := ast.UnaryOperators[op]; !found {
		return nil
	}
	n := &ast.UnaryOp{Operator: op}
	p.ParseBranch(b, func(b *cst.Branch) {
		p.requireOperator(op, b)
		p.ParseLeaf(b, nil)
		p.mappings.Add(n, b)
		n.Expression = p.requireExpression(b)
	})
	return n
}
