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

// lhs { extend }
func (p *parser) requireExpression(b *cst.Branch) ast.Node {
	lhs := p.requireLHSExpression(b)
	for {
		if e := p.extendExpression(lhs); e != nil {
			lhs = e
		} else {
			break
		}
	}
	return lhs
}

// ( group | switch | literal | unary_op | generic)
func (p *parser) requireLHSExpression(b *cst.Branch) ast.Node {
	if g := p.group(b); g != nil {
		return g
	}
	if s := p.switch_(b); s != nil {
		return s
	}
	if l := p.literal(b); l != nil {
		return l
	}
	if u := p.unaryOp(b); u != nil {
		return u
	}
	if g := p.generic(b); g != nil {
		return g
	}
	p.Expected("expression")
	v := &ast.Invalid{}
	p.mappings.Add(v, b)
	return v
}

// lhs (index | call | binary_op | member)
func (p *parser) extendExpression(lhs ast.Node) ast.Node {
	if i := p.index(lhs); i != nil {
		return i
	}
	if c := p.call(lhs); c != nil {
		return c
	}
	if e := p.binaryOp(lhs); e != nil {
		return e
	}
	if m := p.member(lhs); m != nil {
		return m
	}
	return nil
}

// 'null' | 'true' | 'false' | '"' string '"' | '?' | number
func (p *parser) literal(b *cst.Branch) ast.Node {
	if l := p.keyword(ast.KeywordNull, b); l != nil {
		v := &ast.Null{}
		p.mappings.Add(v, l)
		return v
	}
	if l := p.keyword(ast.KeywordTrue, b); l != nil {
		v := &ast.Bool{Value: true}
		p.mappings.Add(v, l)
		return v
	}
	if l := p.keyword(ast.KeywordFalse, b); l != nil {
		v := &ast.Bool{Value: false}
		p.mappings.Add(v, l)
		return v
	}
	if s := p.string_(b); s != nil {
		return s
	}
	if u := p.unknown(b); u != nil {
		return u
	}
	if n := p.number(b); n != nil {
		return n
	}
	return nil
}

func (p *parser) unknown(b *cst.Branch) *ast.Unknown {
	scanned := p.scanOperator()
	if ast.OpUnknown != scanned {
		p.Rollback()
		return nil
	}
	n := &ast.Unknown{}
	p.ParseLeaf(b, func(l *cst.Leaf) {
		p.mappings.Add(n, l)
		l.Token = p.Consume()
	})
	return n
}

// "string" | `string`
func (p *parser) string_(b *cst.Branch) *ast.String {
	quote, backtick := p.Rune(ast.Quote), p.Rune(ast.Backtick)
	var term rune
	switch {
	case quote:
		term = ast.Quote
	case backtick:
		term = ast.Backtick
	default:
		return nil
	}
	n := &ast.String{}
	p.ParseLeaf(b, func(l *cst.Leaf) {
		p.mappings.Add(n, l)
		p.SeekRune(term)
		if !p.Rune(term) {
			n = nil
			return
		}
		l.Token = p.Consume()
		v := l.Token.String()
		n.Value = v[1 : len(v)-1]
	})
	return n
}

func (p *parser) requireString(b *cst.Branch) *ast.String {
	s := p.string_(b)
	if s == nil {
		p.Expected("string")
		s = ast.InvalidString
	}
	return s
}

// standard numeric formats
func (p *parser) number(b *cst.Branch) *ast.Number {
	_ = p.Rune('+') || p.Rune('-') // optional sign
	if p.Numeric() == parse.NotNumeric {
		p.Rollback()
		return nil
	}
	n := &ast.Number{}
	p.ParseLeaf(b, func(l *cst.Leaf) {
		p.mappings.Add(n, l)
		l.Token = p.Consume()
		n.Value = l.Tok().String()
	})
	return n
}

func (p *parser) requireNumber(b *cst.Branch) *ast.Number {
	n := p.number(b)
	if n == nil {
		p.Expected("number")
		n = ast.InvalidNumber
	}
	return n
}

// '(' expression ')'
func (p *parser) group(b *cst.Branch) *ast.Group {
	if !p.peekOperator(ast.OpListStart) {
		return nil
	}
	e := &ast.Group{}
	p.ParseBranch(b, func(b *cst.Branch) {
		p.mappings.Add(e, b)
		p.requireOperator(ast.OpListStart, b)
		e.Expression = p.requireExpression(b)
		p.requireOperator(ast.OpListEnd, b)
	})
	return e
}

// switch '{' { 'case' { expresion } ':' block } '}'
func (p *parser) switch_(b *cst.Branch) *ast.Switch {
	if !p.peekKeyword(ast.KeywordSwitch) {
		return nil
	}
	e := &ast.Switch{}
	p.ParseBranch(b, func(b *cst.Branch) {
		p.mappings.Add(e, b)
		p.requireKeyword(ast.KeywordSwitch, b)
		e.Value = p.requireExpression(b)
		p.requireOperator(ast.OpBlockStart, b)
		annotations := &ast.Annotations{}
		p.parseAnnotations(annotations, b)
		for p.peekKeyword(ast.KeywordCase) {
			p.ParseBranch(b, func(b *cst.Branch) {
				entry := &ast.Case{Annotations: annotationsOrNil(*annotations)}
				p.mappings.Add(entry, b)
				p.requireKeyword(ast.KeywordCase, b)
				for !p.operator(ast.OpInitialise, b) {
					if len(entry.Conditions) > 0 {
						p.requireOperator(ast.OpListSeparator, b)
					}
					entry.Conditions = append(entry.Conditions, p.requireExpression(b))
				}
				entry.Block = p.requireBlock(b)
				e.Cases = append(e.Cases, entry)
			})
			annotations = &ast.Annotations{}
			p.parseAnnotations(annotations, b)
		}
		if p.peekKeyword(ast.KeywordDefault) {
			p.ParseBranch(b, func(b *cst.Branch) {
				entry := &ast.Default{}
				p.mappings.Add(entry, b)
				p.requireKeyword(ast.KeywordDefault, b)
				p.requireOperator(ast.OpInitialise, b)
				entry.Block = p.requireBlock(b)
				e.Default = entry
			})
		}
		p.requireOperator(ast.OpBlockEnd, b)
	})
	return e
}

// lhs '[' expression [ ':' [ expression ] ] ']'
func (p *parser) index(lhs ast.Node) *ast.Index {
	if !p.peekOperator(ast.OpIndexStart) {
		return nil
	}
	e := &ast.Index{Object: lhs}
	p.Extend(p.mappings.CST(lhs), func(b *cst.Branch) {
		p.mappings.Add(e, b)
		p.requireOperator(ast.OpIndexStart, b)
		e.Index = p.requireExpression(b)
		if p.operator(ast.OpSlice, b) {
			n := &ast.BinaryOp{LHS: e.Index, Operator: ast.OpSlice}
			if !p.peekOperator(ast.OpIndexEnd) {
				p.ParseBranch(b, func(b *cst.Branch) {
					p.mappings.Add(n, b)
					n.RHS = p.requireExpression(b)
				})
			}
			e.Index = n
		}
		p.requireOperator(ast.OpIndexEnd, b)
	})
	return e
}

// lhs '(' [ expression { ',' expression } ] ')'
func (p *parser) call(lhs ast.Node) *ast.Call {
	if !p.peekOperator(ast.OpListStart) {
		return nil
	}
	e := &ast.Call{Target: lhs}
	p.Extend(p.mappings.CST(lhs), func(b *cst.Branch) {
		p.mappings.Add(e, b)
		p.requireOperator(ast.OpListStart, b)
		for !p.operator(ast.OpListEnd, b) {
			arg := p.requireExpression(b)
			if i, ok := arg.(*ast.Generic); ok && p.operator(ast.OpInitialise, b) {
				n := &ast.NamedArg{Name: i.Name}
				p.ParseBranch(b, func(b *cst.Branch) {
					p.mappings.Add(n, b)
					n.Value = p.requireExpression(b)
				})
				arg = n
			}
			e.Arguments = append(e.Arguments, arg)
			if p.operator(ast.OpListEnd, b) {
				break
			}
			p.requireOperator(ast.OpListSeparator, b)
		}
	})
	return e
}

// lhs '.' identifier
func (p *parser) member(lhs ast.Node) *ast.Member {
	if !p.peekOperator(ast.OpMember) {
		return nil
	}
	e := &ast.Member{Object: lhs}
	p.Extend(p.mappings.CST(lhs), func(b *cst.Branch) {
		p.mappings.Add(e, b)
		p.requireOperator(ast.OpMember, b)
		e.Name = p.requireIdentifier(b)
	})
	return e
}
