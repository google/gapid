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
	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/core/text/parse/cst"
	"github.com/google/gapid/gapil/ast"
)

// '{' { statements } '}'
func requireBlock(p *parse.Parser, b *cst.Branch) *ast.Block {
	block := &ast.Block{}
	p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(block, b)
		if operator(ast.OpBlockStart, p, b) {
			for !operator(ast.OpBlockEnd, p, b) {
				if p.IsEOF() {
					p.Error("end of file reached while looking for '%s'", ast.OpBlockEnd)
					break
				}
				block.Statements = append(block.Statements, requireStatement(p, b))
			}
		} else {
			block.Statements = append(block.Statements, requireStatement(p, b))
		}
	})
	return block
}

// ( branch | iteration | return | expression ) [ declare_local | assign ]
func requireStatement(p *parse.Parser, b *cst.Branch) ast.Node {
	if g := branch(p, b); g != nil {
		return g
	}
	if g := delete(p, b); g != nil {
		return g
	}
	if g := iteration(p, b); g != nil {
		return g
	}
	if g := return_(p, b); g != nil {
		return g
	}
	if g := abort(p, b); g != nil {
		return g
	}
	if g := fence(p, b); g != nil {
		return g
	}
	e := requireExpression(p, b)
	if g := declareLocal(p, b, e); g != nil {
		return g
	}
	if g := assign(p, b, e); g != nil {
		return g
	}
	return e
}

// 'if' expression block [ 'else' block ]
func branch(p *parse.Parser, b *cst.Branch) *ast.Branch {
	if !peekKeyword(ast.KeywordIf, p) {
		return nil
	}
	s := &ast.Branch{}
	p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(s, b)
		requireKeyword(ast.KeywordIf, p, b)
		s.Condition = requireExpression(p, b)
		s.True = requireBlock(p, b)
		if keyword(ast.KeywordElse, p, b) != nil {
			s.False = requireBlock(p, b)
		}
	})
	return s
}

// 'for' identifier 'in' expresion block
// 'for' identifier, identifier, identifier 'in' expression block
func iteration(p *parse.Parser, b *cst.Branch) ast.Node {
	if !peekKeyword(ast.KeywordFor, p) {
		return nil
	}
	isMap := false
	s := &ast.MapIteration{}
	p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(s, b)
		requireKeyword(ast.KeywordFor, p, b)
		s.IndexVariable = requireIdentifier(p, b)
		if isMap = peekOperator(ast.OpListSeparator, p); isMap {
			requireOperator(ast.OpListSeparator, p, b)
			s.KeyVariable = requireIdentifier(p, b)
			requireOperator(ast.OpListSeparator, p, b)
			s.ValueVariable = requireIdentifier(p, b)
		}
		requireKeyword(ast.KeywordIn, p, b)
		s.Map = requireExpression(p, b)
		s.Block = requireBlock(p, b)
	})
	if isMap {
		return s
	}
	it := &ast.Iteration{Variable: s.IndexVariable, Iterable: s.Map, Block: s.Block}
	p.SetCST(it, p.CST(s))
	return it
}

// lhs ':=' expression
func declareLocal(p *parse.Parser, b *cst.Branch, lhs ast.Node) *ast.DeclareLocal {
	l, ok := lhs.(*ast.Generic)
	if !ok || len(l.Arguments) > 0 || !peekOperator(ast.OpDeclare, p) {
		return nil
	}
	s := &ast.DeclareLocal{Name: l.Name}
	p.Extend(lhs, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(s, b)
		requireOperator(ast.OpDeclare, p, b)
		s.RHS = requireExpression(p, b)
	})
	return s
}

var assignments = []string{
	ast.OpAssign,
	ast.OpAssignPlus,
	ast.OpAssignMinus,
}

// lhs ( '=' | '+=' | '-=' )  expression
func assign(p *parse.Parser, b *cst.Branch, lhs ast.Node) *ast.Assign {
	op := ""
	for _, test := range assignments {
		if peekOperator(test, p) {
			op = test
			break
		}
	}
	if op == "" {
		return nil
	}
	s := &ast.Assign{LHS: lhs, Operator: op}
	p.Extend(lhs, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(s, b)
		requireOperator(op, p, b)
		s.RHS = requireExpression(p, b)
	})
	return s
}

// 'delete' ( expression, expression )
func delete(p *parse.Parser, b *cst.Branch) *ast.Delete {
	if !peekKeyword(ast.KeywordDelete, p) {
		return nil
	}
	f := &ast.Delete{}
	p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(f, b)
		requireKeyword(ast.KeywordDelete, p, b)
		requireOperator(ast.OpListStart, p, b)
		f.Map = requireExpression(p, b)
		requireOperator(ast.OpListSeparator, p, b)
		f.Key = requireExpression(p, b)
		requireOperator(ast.OpListEnd, p, b)
	})
	return f
}

// 'return' expresssion
func return_(p *parse.Parser, b *cst.Branch) *ast.Return {
	if !peekKeyword(ast.KeywordReturn, p) {
		return nil
	}
	s := &ast.Return{}
	p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(s, b)
		requireKeyword(ast.KeywordReturn, p, b)
		s.Value = requireExpression(p, b)
	})
	return s
}

// 'abort' statement
func abort(p *parse.Parser, b *cst.Branch) *ast.Abort {
	if !peekKeyword(ast.KeywordAbort, p) {
		return nil
	}
	f := &ast.Abort{}
	p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
		requireKeyword(ast.KeywordAbort, p, b)
		p.SetCST(f, b)
	})
	return f
}

// 'fence' statement
func fence(p *parse.Parser, b *cst.Branch) *ast.Fence {
	if !peekKeyword(ast.KeywordFence, p) {
		return nil
	}
	f := &ast.Fence{}
	p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
		requireKeyword(ast.KeywordFence, p, b)
		p.SetCST(f, b)
	})
	return f
}
