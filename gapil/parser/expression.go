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
func requireExpression(p *parse.Parser, b *cst.Branch) ast.Node {
	lhs := requireLHSExpression(p, b)
	for {
		if e := extendExpression(p, lhs); e != nil {
			lhs = e
		} else {
			break
		}
	}
	return lhs
}

// ( group | switch | literal | unary_op | generic)
func requireLHSExpression(p *parse.Parser, b *cst.Branch) ast.Node {
	if g := group(p, b); g != nil {
		return g
	}
	if s := switch_(p, b); s != nil {
		return s
	}
	if l := literal(p, b); l != nil {
		return l
	}
	if u := unaryOp(p, b); u != nil {
		return u
	}
	if g := generic(p, b); g != nil {
		return g
	}
	p.Expected("expression")
	v := &ast.Invalid{}
	p.SetCST(v, b)
	return v
}

// lhs (index | call | binary_op | member)
func extendExpression(p *parse.Parser, lhs ast.Node) ast.Node {
	if i := index(p, lhs); i != nil {
		return i
	}
	if c := call(p, lhs); c != nil {
		return c
	}
	if e := binaryOp(p, lhs); e != nil {
		return e
	}
	if m := member(p, lhs); m != nil {
		return m
	}
	return nil
}

// 'null' | 'true' | 'false' | '"' string '"' | '?' | number
func literal(p *parse.Parser, b *cst.Branch) ast.Node {
	if l := keyword(ast.KeywordNull, p, b); l != nil {
		v := &ast.Null{}
		p.SetCST(v, l)
		return v
	}
	if l := keyword(ast.KeywordTrue, p, b); l != nil {
		v := &ast.Bool{Value: true}
		p.SetCST(v, l)
		return v
	}
	if l := keyword(ast.KeywordFalse, p, b); l != nil {
		v := &ast.Bool{Value: false}
		p.SetCST(v, l)
		return v
	}
	if s := string_(p, b); s != nil {
		return s
	}
	if u := unknown(p, b); u != nil {
		return u
	}
	if n := number(p, b); n != nil {
		return n
	}
	return nil
}

func unknown(p *parse.Parser, b *cst.Branch) *ast.Unknown {
	scanned := scanOperator(p)
	if ast.OpUnknown != scanned {
		p.Rollback()
		return nil
	}
	n := &ast.Unknown{}
	p.ParseLeaf(b, func(p *parse.Parser, l *cst.Leaf) {
		p.SetCST(n, l)
		l.Token = p.Consume()
	})
	return n
}

// "string" | `string`
func string_(p *parse.Parser, b *cst.Branch) *ast.String {
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
	p.ParseLeaf(b, func(p *parse.Parser, l *cst.Leaf) {
		p.SetCST(n, l)
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

func requireString(p *parse.Parser, b *cst.Branch) *ast.String {
	s := string_(p, b)
	if s == nil {
		p.Expected("string")
		s = ast.InvalidString
	}
	return s
}

// standard numeric formats
func number(p *parse.Parser, b *cst.Branch) *ast.Number {
	_ = p.Rune('+') || p.Rune('-') // optional sign
	if p.Numeric() == parse.NotNumeric {
		p.Rollback()
		return nil
	}
	n := &ast.Number{}
	p.ParseLeaf(b, func(p *parse.Parser, l *cst.Leaf) {
		p.SetCST(n, l)
		l.Token = p.Consume()
		n.Value = l.Tok().String()
	})
	return n
}

func requireNumber(p *parse.Parser, b *cst.Branch) *ast.Number {
	n := number(p, b)
	if n == nil {
		p.Expected("number")
		n = ast.InvalidNumber
	}
	return n
}

// '(' expression ')'
func group(p *parse.Parser, b *cst.Branch) *ast.Group {
	if !peekOperator(ast.OpListStart, p) {
		return nil
	}
	e := &ast.Group{}
	p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(e, b)
		requireOperator(ast.OpListStart, p, b)
		e.Expression = requireExpression(p, b)
		requireOperator(ast.OpListEnd, p, b)
	})
	return e
}

// switch '{' { 'case' { expresion } ':' block } '}'
func switch_(p *parse.Parser, b *cst.Branch) *ast.Switch {
	if !peekKeyword(ast.KeywordSwitch, p) {
		return nil
	}
	e := &ast.Switch{}
	p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(e, b)
		requireKeyword(ast.KeywordSwitch, p, b)
		e.Value = requireExpression(p, b)
		requireOperator(ast.OpBlockStart, p, b)
		annotations := &ast.Annotations{}
		parseAnnotations(annotations, p, b)
		for peekKeyword(ast.KeywordCase, p) {
			p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
				entry := &ast.Case{Annotations: *annotations}
				p.SetCST(entry, b)
				requireKeyword(ast.KeywordCase, p, b)
				for !operator(ast.OpInitialise, p, b) {
					if len(entry.Conditions) > 0 {
						requireOperator(ast.OpListSeparator, p, b)
					}
					entry.Conditions = append(entry.Conditions, requireExpression(p, b))
				}
				entry.Block = requireBlock(p, b)
				e.Cases = append(e.Cases, entry)
			})
			annotations = &ast.Annotations{}
			parseAnnotations(annotations, p, b)
		}
		if peekKeyword(ast.KeywordDefault, p) {
			p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
				entry := &ast.Default{}
				p.SetCST(entry, b)
				requireKeyword(ast.KeywordDefault, p, b)
				requireOperator(ast.OpInitialise, p, b)
				entry.Block = requireBlock(p, b)
				e.Default = entry
			})
		}
		requireOperator(ast.OpBlockEnd, p, b)
	})
	return e
}

// lhs '[' expression [ ':' [ expression ] ] ']'
func index(p *parse.Parser, lhs ast.Node) *ast.Index {
	if !peekOperator(ast.OpIndexStart, p) {
		return nil
	}
	e := &ast.Index{Object: lhs}
	p.Extend(lhs, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(e, b)
		requireOperator(ast.OpIndexStart, p, b)
		e.Index = requireExpression(p, b)
		if operator(ast.OpSlice, p, b) {
			n := &ast.BinaryOp{LHS: e.Index, Operator: ast.OpSlice}
			if !peekOperator(ast.OpIndexEnd, p) {
				p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
					p.SetCST(n, b)
					n.RHS = requireExpression(p, b)
				})
			}
			e.Index = n
		}
		requireOperator(ast.OpIndexEnd, p, b)
	})
	return e
}

// lhs '(' [ expression { ',' expression } ] ')'
func call(p *parse.Parser, lhs ast.Node) *ast.Call {
	if !peekOperator(ast.OpListStart, p) {
		return nil
	}
	e := &ast.Call{Target: lhs}
	p.Extend(lhs, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(e, b)
		requireOperator(ast.OpListStart, p, b)
		for !operator(ast.OpListEnd, p, b) {
			arg := requireExpression(p, b)
			if i, ok := arg.(*ast.Generic); ok && operator(ast.OpInitialise, p, b) {
				n := &ast.NamedArg{Name: i.Name}
				p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
					p.SetCST(n, b)
					n.Value = requireExpression(p, b)
				})
				arg = n
			}
			e.Arguments = append(e.Arguments, arg)
			if operator(ast.OpListEnd, p, b) {
				break
			}
			requireOperator(ast.OpListSeparator, p, b)
		}
	})
	return e
}

// lhs '.' identifier
func member(p *parse.Parser, lhs ast.Node) *ast.Member {
	if !peekOperator(ast.OpMember, p) {
		return nil
	}
	e := &ast.Member{Object: lhs}
	p.Extend(lhs, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(e, b)
		requireOperator(ast.OpMember, p, b)
		e.Name = requireIdentifier(p, b)
	})
	return e
}
