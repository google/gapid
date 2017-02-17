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
	"github.com/google/gapid/core/gapil/ast"
	"github.com/google/gapid/core/text/parse"
)

// '{' { statements } '}'
func requireBlock(p *parse.Parser, cst *parse.Branch) *ast.Block {
	block := &ast.Block{}
	p.ParseBranch(cst, func(p *parse.Parser, cst *parse.Branch) {
		p.SetCST(block, cst)
		if operator(ast.OpBlockStart, p, cst) {
			for !operator(ast.OpBlockEnd, p, cst) {
				if p.IsEOF() {
					p.Error("end of file reached while looking for '%s'", ast.OpBlockEnd)
					break
				}
				block.Statements = append(block.Statements, requireStatement(p, cst))
			}
		} else {
			block.Statements = append(block.Statements, requireStatement(p, cst))
		}
	})
	return block
}

// ( branch | iteration | return | expression ) [ declare_local | assign ]
func requireStatement(p *parse.Parser, cst *parse.Branch) ast.Node {
	if g := branch(p, cst); g != nil {
		return g
	}
	if g := delete(p, cst); g != nil {
		return g
	}
	if g := iteration(p, cst); g != nil {
		return g
	}
	if g := return_(p, cst); g != nil {
		return g
	}
	if g := abort(p, cst); g != nil {
		return g
	}
	if g := fence(p, cst); g != nil {
		return g
	}
	e := requireExpression(p, cst)
	if g := declareLocal(p, cst, e); g != nil {
		return g
	}
	if g := assign(p, cst, e); g != nil {
		return g
	}
	return e
}

// 'if' expression block [ 'else' block ]
func branch(p *parse.Parser, cst *parse.Branch) *ast.Branch {
	if !peekKeyword(ast.KeywordIf, p) {
		return nil
	}
	s := &ast.Branch{}
	p.ParseBranch(cst, func(p *parse.Parser, cst *parse.Branch) {
		p.SetCST(s, cst)
		requireKeyword(ast.KeywordIf, p, cst)
		s.Condition = requireExpression(p, cst)
		s.True = requireBlock(p, cst)
		if keyword(ast.KeywordElse, p, cst) != nil {
			s.False = requireBlock(p, cst)
		}
	})
	return s
}

// 'for' identifier 'in' expresion block
// 'for' identifier, identifier, identifier 'in' expression block
func iteration(p *parse.Parser, cst *parse.Branch) ast.Node {
	if !peekKeyword(ast.KeywordFor, p) {
		return nil
	}
	isMap := false
	s := &ast.MapIteration{}
	p.ParseBranch(cst, func(p *parse.Parser, cst *parse.Branch) {
		p.SetCST(s, cst)
		requireKeyword(ast.KeywordFor, p, cst)
		s.IndexVariable = requireIdentifier(p, cst)
		if isMap = peekOperator(ast.OpListSeparator, p); isMap {
			requireOperator(ast.OpListSeparator, p, cst)
			s.KeyVariable = requireIdentifier(p, cst)
			requireOperator(ast.OpListSeparator, p, cst)
			s.ValueVariable = requireIdentifier(p, cst)
		}
		requireKeyword(ast.KeywordIn, p, cst)
		s.Map = requireExpression(p, cst)
		s.Block = requireBlock(p, cst)
	})
	if isMap {
		return s
	}
	it := &ast.Iteration{Variable: s.IndexVariable, Iterable: s.Map, Block: s.Block}
	p.SetCST(it, p.CST(s))
	return it
}

// lhs ':=' expression
func declareLocal(p *parse.Parser, cst *parse.Branch, lhs ast.Node) *ast.DeclareLocal {
	l, ok := lhs.(*ast.Generic)
	if !ok || len(l.Arguments) > 0 || !peekOperator(ast.OpDeclare, p) {
		return nil
	}
	s := &ast.DeclareLocal{Name: l.Name}
	p.Extend(lhs, func(p *parse.Parser, cst *parse.Branch) {
		p.SetCST(s, cst)
		requireOperator(ast.OpDeclare, p, cst)
		s.RHS = requireExpression(p, cst)
	})
	return s
}

var assignments = []string{
	ast.OpAssign,
	ast.OpAssignPlus,
	ast.OpAssignMinus,
}

// lhs ( '=' | '+=' | '-=' )  expression
func assign(p *parse.Parser, cst *parse.Branch, lhs ast.Node) *ast.Assign {
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
	p.Extend(lhs, func(p *parse.Parser, cst *parse.Branch) {
		p.SetCST(s, cst)
		requireOperator(op, p, cst)
		s.RHS = requireExpression(p, cst)
	})
	return s
}

// 'delete' ( expression, expression )
func delete(p *parse.Parser, cst *parse.Branch) *ast.Delete {
	if !peekKeyword(ast.KeywordDelete, p) {
		return nil
	}
	f := &ast.Delete{}
	p.ParseBranch(cst, func(p *parse.Parser, cst *parse.Branch) {
		p.SetCST(f, cst)
		requireKeyword(ast.KeywordDelete, p, cst)
		requireOperator(ast.OpListStart, p, cst)
		f.Map = requireExpression(p, cst)
		requireOperator(ast.OpListSeparator, p, cst)
		f.Key = requireExpression(p, cst)
		requireOperator(ast.OpListEnd, p, cst)
	})
	return f
}

// 'return' expresssion
func return_(p *parse.Parser, cst *parse.Branch) *ast.Return {
	if !peekKeyword(ast.KeywordReturn, p) {
		return nil
	}
	s := &ast.Return{}
	p.ParseBranch(cst, func(p *parse.Parser, cst *parse.Branch) {
		p.SetCST(s, cst)
		requireKeyword(ast.KeywordReturn, p, cst)
		s.Value = requireExpression(p, cst)
	})
	return s
}

// 'abort' statement
func abort(p *parse.Parser, cst *parse.Branch) *ast.Abort {
	if !peekKeyword(ast.KeywordAbort, p) {
		return nil
	}
	f := &ast.Abort{}
	p.ParseBranch(cst, func(p *parse.Parser, cst *parse.Branch) {
		requireKeyword(ast.KeywordAbort, p, cst)
		p.SetCST(f, cst)
	})
	return f
}

// 'fence' statement
func fence(p *parse.Parser, cst *parse.Branch) *ast.Fence {
	if !peekKeyword(ast.KeywordFence, p) {
		return nil
	}
	f := &ast.Fence{}
	p.ParseBranch(cst, func(p *parse.Parser, cst *parse.Branch) {
		requireKeyword(ast.KeywordFence, p, cst)
		p.SetCST(f, cst)
	})
	return f
}
