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

func (p *parser) identifier(b *cst.Branch) *ast.Identifier {
	p.Rune('$')
	if !p.AlphaNumeric() {
		return nil
	}
	n := &ast.Identifier{}
	p.ParseLeaf(b, func(l *cst.Leaf) {
		p.mappings.Add(n, l)
		l.Token = p.Consume()
		n.Value = l.Token.String()
	})
	return n
}

func (p *parser) requireIdentifier(b *cst.Branch) *ast.Identifier {
	n := p.identifier(b)
	if n == nil {
		p.Expected("Identifier")
		return ast.InvalidIdentifier
	}
	return n
}

// name '!' ( type | '(' type [ ',' type ] ')' )
func (p *parser) generic(b *cst.Branch) *ast.Generic {
	name := p.identifier(b)
	if name == nil {
		return nil
	}

	g := &ast.Generic{Name: name}
	p.Extend(p.mappings.CST(name), func(b *cst.Branch) {
		p.mappings.Add(g, b)
		if p.operator(ast.OpGeneric, b) {
			if p.operator(ast.OpListStart, b) {
				for !p.operator(ast.OpListEnd, b) {
					if len(g.Arguments) > 0 {
						p.requireOperator(ast.OpListSeparator, b)
					}
					g.Arguments = append(g.Arguments, p.requireTypeRef(b))
				}
			} else {
				g.Arguments = append(g.Arguments, p.requireTypeRef(b))
			}
		}
	})
	return g
}

func (p *parser) requireGeneric(b *cst.Branch) *ast.Generic {
	n := p.generic(b)
	if n == nil {
		p.Expected("generic identifier")
		n = ast.InvalidGeneric
	}
	return n
}

func (p *parser) peekKeyword(k string) bool {
	if !p.AlphaNumeric() {
		return false
	}
	if p.Token().String() != string(k) {
		p.Rollback()
		return false
	}
	p.Rollback()
	return true
}

func (p *parser) keyword(k string, b *cst.Branch) *cst.Leaf {
	if !p.AlphaNumeric() {
		return nil
	}
	if p.Token().String() != string(k) {
		p.Rollback()
		return nil
	}
	var result *cst.Leaf
	p.ParseLeaf(b, func(l *cst.Leaf) {
		result = l
	})
	return result
}

func (p *parser) requireKeyword(k string, b *cst.Branch) {
	if p.keyword(k, b) == nil {
		p.Expected(string(k))
	}
}
