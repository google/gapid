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

	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/core/text/parse/cst"
	"github.com/google/gapid/gapil/ast"
)

func peekOperator(op string, p *parse.Parser) bool {
	scanned := scanOperator(p)
	p.Rollback()
	return op == scanned
}

func operator(op string, p *parse.Parser, b *cst.Branch) bool {
	scanned := scanOperator(p)
	if op != scanned {
		p.Rollback()
		return false
	}
	p.ParseLeaf(b, nil)
	return true
}

func requireOperator(op string, p *parse.Parser, b *cst.Branch) {
	if !operator(op, p, b) {
		p.Expected(string(op))
	}
}

func scanOperator(p *parse.Parser) string {
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
func binaryOp(p *parse.Parser, lhs ast.Node) *ast.BinaryOp {
	op := scanOperator(p)
	if _, found := ast.BinaryOperators[op]; !found {
		p.Rollback()
		return nil
	}
	n := &ast.BinaryOp{LHS: lhs, Operator: op}
	p.Extend(lhs, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(n, b)
		p.ParseLeaf(b, nil)
		n.RHS = requireExpression(p, b)
	})
	return n
}

// operator expression
func unaryOp(p *parse.Parser, b *cst.Branch) *ast.UnaryOp {
	op := scanOperator(p)
	p.Rollback()
	if _, found := ast.UnaryOperators[op]; !found {
		return nil
	}
	n := &ast.UnaryOp{Operator: op}
	p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
		requireOperator(op, p, b)
		p.ParseLeaf(b, nil)
		p.SetCST(n, b)
		n.Expression = requireExpression(p, b)
	})
	return n
}
