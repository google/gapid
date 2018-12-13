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
	"github.com/google/gapid/core/text/parse/cst"
	"github.com/google/gapid/gapil/ast"
)

// '{' { statements } '}'
func (p *parser) requireBlock(b *cst.Branch) *ast.Block {
	block := &ast.Block{}
	p.ParseBranch(b, func(b *cst.Branch) {
		p.mappings.Add(block, b)
		if p.operator(ast.OpBlockStart, b) {
			for !p.operator(ast.OpBlockEnd, b) {
				if p.IsEOF() {
					p.Error("end of file reached while looking for '%s'", ast.OpBlockEnd)
					break
				}
				block.Statements = append(block.Statements, p.requireStatement(b))
			}
		} else {
			block.Statements = append(block.Statements, p.requireStatement(b))
		}
	})
	return block
}

// ( branch | iteration | return | expression ) [ declare_local | assign ]
func (p *parser) requireStatement(b *cst.Branch) ast.Node {
	if g := p.branch(b); g != nil {
		return g
	}
	if g := p.delete(b); g != nil {
		return g
	}
	if g := p.clear(b); g != nil {
		return g
	}
	if g := p.iteration(b); g != nil {
		return g
	}
	if g := p.return_(b); g != nil {
		return g
	}
	if g := p.abort(b); g != nil {
		return g
	}
	if g := p.fence(b); g != nil {
		return g
	}
	e := p.requireExpression(b)
	if g := p.declareLocal(b, e); g != nil {
		return g
	}
	if g := p.assign(b, e); g != nil {
		return g
	}
	return e
}

// 'if' expression block [ 'else' block ]
func (p *parser) branch(b *cst.Branch) *ast.Branch {
	if !p.peekKeyword(ast.KeywordIf) {
		return nil
	}
	s := &ast.Branch{}
	p.ParseBranch(b, func(b *cst.Branch) {
		p.mappings.Add(s, b)
		p.requireKeyword(ast.KeywordIf, b)
		s.Condition = p.requireExpression(b)
		s.True = p.requireBlock(b)
		if p.keyword(ast.KeywordElse, b) != nil {
			s.False = p.requireBlock(b)
		}
	})
	return s
}

// 'for' identifier 'in' expresion block
// 'for' identifier, identifier, identifier 'in' expression block
func (p *parser) iteration(b *cst.Branch) ast.Node {
	if !p.peekKeyword(ast.KeywordFor) {
		return nil
	}
	isMap := false
	s := &ast.MapIteration{}
	p.ParseBranch(b, func(b *cst.Branch) {
		p.mappings.Add(s, b)
		p.requireKeyword(ast.KeywordFor, b)
		s.IndexVariable = p.requireIdentifier(b)
		if isMap = p.peekOperator(ast.OpListSeparator); isMap {
			p.requireOperator(ast.OpListSeparator, b)
			s.KeyVariable = p.requireIdentifier(b)
			p.requireOperator(ast.OpListSeparator, b)
			s.ValueVariable = p.requireIdentifier(b)
		}
		p.requireKeyword(ast.KeywordIn, b)
		s.Map = p.requireExpression(b)
		s.Block = p.requireBlock(b)
	})
	if isMap {
		return s
	}
	it := &ast.Iteration{Variable: s.IndexVariable, Iterable: s.Map, Block: s.Block}
	p.mappings.Add(it, p.mappings.CST(s))
	return it
}

// lhs ':=' expression
func (p *parser) declareLocal(b *cst.Branch, lhs ast.Node) *ast.DeclareLocal {
	l, ok := lhs.(*ast.Generic)
	if !ok || len(l.Arguments) > 0 || !p.peekOperator(ast.OpDeclare) {
		return nil
	}
	s := &ast.DeclareLocal{Name: l.Name}
	p.Extend(p.mappings.CST(lhs), func(b *cst.Branch) {
		p.mappings.Add(s, b)
		p.requireOperator(ast.OpDeclare, b)
		s.RHS = p.requireExpression(b)
	})
	return s
}

var assignments = []string{
	ast.OpAssign,
	ast.OpAssignPlus,
	ast.OpAssignMinus,
}

// lhs ( '=' | '+=' | '-=' )  expression
func (p *parser) assign(b *cst.Branch, lhs ast.Node) *ast.Assign {
	op := ""
	for _, test := range assignments {
		if p.peekOperator(test) {
			op = test
			break
		}
	}
	if op == "" {
		return nil
	}
	s := &ast.Assign{LHS: lhs, Operator: op}
	p.Extend(p.mappings.CST(lhs), func(b *cst.Branch) {
		p.mappings.Add(s, b)
		p.requireOperator(op, b)
		s.RHS = p.requireExpression(b)
	})
	return s
}

// 'delete' ( expression, expression )
func (p *parser) delete(b *cst.Branch) *ast.Delete {
	if !p.peekKeyword(ast.KeywordDelete) {
		return nil
	}
	f := &ast.Delete{}
	p.ParseBranch(b, func(b *cst.Branch) {
		p.mappings.Add(f, b)
		p.requireKeyword(ast.KeywordDelete, b)
		p.requireOperator(ast.OpListStart, b)
		f.Map = p.requireExpression(b)
		p.requireOperator(ast.OpListSeparator, b)
		f.Key = p.requireExpression(b)
		p.requireOperator(ast.OpListEnd, b)
	})
	return f
}

// 'clear' ( expression )
func (p *parser) clear(b *cst.Branch) *ast.Clear {
	if !p.peekKeyword(ast.KeywordClear) {
		return nil
	}
	f := &ast.Clear{}
	p.ParseBranch(b, func(b *cst.Branch) {
		p.mappings.Add(f, b)
		p.requireKeyword(ast.KeywordClear, b)
		p.requireOperator(ast.OpListStart, b)
		f.Map = p.requireExpression(b)
		p.requireOperator(ast.OpListEnd, b)
	})
	return f
}

// 'return' expresssion
func (p *parser) return_(b *cst.Branch) *ast.Return {
	if !p.peekKeyword(ast.KeywordReturn) {
		return nil
	}
	s := &ast.Return{}
	p.ParseBranch(b, func(b *cst.Branch) {
		p.mappings.Add(s, b)
		p.requireKeyword(ast.KeywordReturn, b)
		s.Value = p.requireExpression(b)
	})
	return s
}

// 'abort' statement
func (p *parser) abort(b *cst.Branch) *ast.Abort {
	if !p.peekKeyword(ast.KeywordAbort) {
		return nil
	}
	f := &ast.Abort{}
	p.ParseBranch(b, func(b *cst.Branch) {
		p.requireKeyword(ast.KeywordAbort, b)
		p.mappings.Add(f, b)
	})
	return f
}

// 'fence' statement
func (p *parser) fence(b *cst.Branch) *ast.Fence {
	if !p.peekKeyword(ast.KeywordFence) {
		return nil
	}
	f := &ast.Fence{}
	p.ParseBranch(b, func(b *cst.Branch) {
		p.requireKeyword(ast.KeywordFence, b)
		p.mappings.Add(f, b)
	})
	return f
}
