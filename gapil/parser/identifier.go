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

func identifier(p *parse.Parser, b *cst.Branch) *ast.Identifier {
	p.Rune('$')
	if !p.AlphaNumeric() {
		return nil
	}
	n := &ast.Identifier{}
	p.ParseLeaf(b, func(p *parse.Parser, l *cst.Leaf) {
		p.SetCST(n, l)
		l.Token = p.Consume()
		n.Value = l.Token.String()
	})
	return n
}

func requireIdentifier(p *parse.Parser, b *cst.Branch) *ast.Identifier {
	n := identifier(p, b)
	if n == nil {
		p.Expected("Identifier")
		return ast.InvalidIdentifier
	}
	return n
}

// name '!' ( type | '(' type [ ',' type ] ')' )
func generic(p *parse.Parser, b *cst.Branch) *ast.Generic {
	name := identifier(p, b)
	if name == nil {
		return nil
	}

	g := &ast.Generic{Name: name}
	p.Extend(name, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(g, b)
		if operator(ast.OpGeneric, p, b) {
			if operator(ast.OpListStart, p, b) {
				for !operator(ast.OpListEnd, p, b) {
					if len(g.Arguments) > 0 {
						requireOperator(ast.OpListSeparator, p, b)
					}
					g.Arguments = append(g.Arguments, requireTypeRef(p, b))
				}
			} else {
				g.Arguments = append(g.Arguments, requireTypeRef(p, b))
			}
		}
	})
	return g
}

func requireGeneric(p *parse.Parser, b *cst.Branch) *ast.Generic {
	n := generic(p, b)
	if n == nil {
		p.Expected("generic identifier")
		n = ast.InvalidGeneric
	}
	return n
}

func peekKeyword(k string, p *parse.Parser) bool {
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

func keyword(k string, p *parse.Parser, b *cst.Branch) *cst.Leaf {
	if !p.AlphaNumeric() {
		return nil
	}
	if p.Token().String() != string(k) {
		p.Rollback()
		return nil
	}
	var result *cst.Leaf
	p.ParseLeaf(b, func(p *parse.Parser, l *cst.Leaf) {
		result = l
	})
	return result
}

func requireKeyword(k string, p *parse.Parser, b *cst.Branch) {
	if keyword(k, p, b) == nil {
		p.Expected(string(k))
	}
}
