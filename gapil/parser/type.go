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

// { annotation } 'class' identifer '{' { field } '}'
func class(p *parse.Parser, b *cst.Branch, a *ast.Annotations) *ast.Class {
	if !peekKeyword(ast.KeywordClass, p) {
		return nil
	}
	c := &ast.Class{}
	consumeAnnotations(&c.Annotations, a)
	p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(c, b)
		requireKeyword(ast.KeywordClass, p, b)
		c.Name = requireIdentifier(p, b)
		requireOperator(ast.OpBlockStart, p, b)
		for !operator(ast.OpBlockEnd, p, b) {
			c.Fields = append(c.Fields, requireField(p, b, nil))
		}
	})
	return c
}

// { annotation } type identifier [ '=' expression ] [ ',' ]
func requireField(p *parse.Parser, b *cst.Branch, a *ast.Annotations) *ast.Field {
	f := &ast.Field{}
	consumeAnnotations(&f.Annotations, a)
	p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(f, b)
		parseAnnotations(&f.Annotations, p, b)
		f.Type = requireTypeRef(p, b)
		f.Name = requireIdentifier(p, b)
		if operator(ast.OpAssign, p, b) {
			f.Default = requireExpression(p, b)
		}
		operator(ast.OpListSeparator, p, b)
	})
	return f
}

// api_index number
func apiIndex(p *parse.Parser, b *cst.Branch, a *ast.Annotations) *ast.Number {
	if !peekKeyword(ast.KeywordApiIndex, p) {
		return nil
	}
	if len(*a) != 0 {
		p.ErrorAt((*a)[0], "Annotation on api_index not allowed")
		return nil
	}
	var f *ast.Number
	p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
		requireKeyword(ast.KeywordApiIndex, p, b)
		f = requireNumber(p, b)
	})
	return f
}

// { annotation } 'define' identifier expression
func definition(p *parse.Parser, b *cst.Branch, a *ast.Annotations) *ast.Definition {
	if !peekKeyword(ast.KeywordDefine, p) {
		return nil
	}
	d := &ast.Definition{}
	consumeAnnotations(&d.Annotations, a)
	p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(d, b)
		requireKeyword(ast.KeywordDefine, p, b)
		d.Name = requireIdentifier(p, b)
		d.Expression = requireExpression(p, b)
	})
	return d
}

// { annotation } ( 'enum' | 'bitfield' ) name [ : type ] '{' { identifier '=' expression [ ',' ] } '}'
func enum(p *parse.Parser, b *cst.Branch, a *ast.Annotations) *ast.Enum {
	if !peekKeyword(ast.KeywordEnum, p) && !peekKeyword(ast.KeywordBitfield, p) {
		return nil
	}
	s := &ast.Enum{}
	consumeAnnotations(&s.Annotations, a)
	p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(s, b)
		if keyword(ast.KeywordEnum, p, b) == nil {
			requireKeyword(ast.KeywordBitfield, p, b)
			s.IsBitfield = true
		}
		s.Name = requireIdentifier(p, b)
		if operator(ast.OpExtends, p, b) {
			s.NumberType = requireTypeRef(p, b)
		}
		requireOperator(ast.OpBlockStart, p, b)
		for !operator(ast.OpBlockEnd, p, b) {
			p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
				entry := &ast.EnumEntry{}
				p.SetCST(entry, b)
				entry.Name = requireIdentifier(p, b)
				requireOperator(ast.OpAssign, p, b)
				entry.Value = requireNumber(p, b)
				operator(ast.OpListSeparator, p, b)
				s.Entries = append(s.Entries, entry)
			})
		}
	})
	return s
}

// { annotation } 'alias' type identifier
func alias(p *parse.Parser, b *cst.Branch, a *ast.Annotations) *ast.Alias {
	if !peekKeyword(ast.KeywordAlias, p) {
		return nil
	}
	s := &ast.Alias{}
	consumeAnnotations(&s.Annotations, a)
	p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(s, b)
		requireKeyword(ast.KeywordAlias, p, b)
		s.To = requireTypeRef(p, b)
		s.Name = requireIdentifier(p, b)
	})
	return s
}

// { annotation } 'type' type identifier
func pseudonym(p *parse.Parser, b *cst.Branch, a *ast.Annotations) *ast.Pseudonym {
	if !peekKeyword(ast.KeywordPseudonym, p) {
		return nil
	}
	s := &ast.Pseudonym{}
	consumeAnnotations(&s.Annotations, a)
	p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(s, b)
		requireKeyword(ast.KeywordPseudonym, p, b)
		s.To = requireTypeRef(p, b)
		s.Name = requireIdentifier(p, b)
	})
	return s
}

// [const] generic [.name] { extend_type }
func typeRef(p *parse.Parser, b *cst.Branch) ast.Node {
	var ref ast.Node

	if peekKeyword(ast.KeywordConst, p) {
		c := &ast.PreConst{}
		p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
			p.SetCST(c, b)
			requireKeyword(ast.KeywordConst, p, b)
			el := requireTypeBase(p, b)
			ptr := &ast.PointerType{To: el}
			p.Extend(el, func(p *parse.Parser, b *cst.Branch) {
				p.SetCST(ptr, b)
				requireOperator(ast.OpPointer, p, b)
			})
			c.Type = ptr
		})
		ref = c
	} else {
		ref = typeBase(p, b)
	}

	if ref == nil {
		return nil
	}

	for {
		if t := extendTypeRef(p, ref); t != nil {
			ref = t
		} else {
			break
		}
	}
	return ref
}

// generic [.name]
func typeBase(p *parse.Parser, b *cst.Branch) ast.Node {
	g := generic(p, b)
	if g == nil {
		return nil
	}

	if peekOperator(ast.OpMember, p) {
		t := &ast.Imported{From: g.Name}
		p.Extend(g, func(p *parse.Parser, b *cst.Branch) {
			p.SetCST(t, b)
			requireOperator(ast.OpMember, p, b)
			t.Name = requireIdentifier(p, b)
		})
		return t
	}
	return g
}

func requireTypeBase(p *parse.Parser, b *cst.Branch) ast.Node {
	t := typeBase(p, b)
	if t == nil {
		p.Expected("type")
		return ast.InvalidGeneric
	}
	return t
}

// ref ( pointer_type | static_array_type )
func extendTypeRef(p *parse.Parser, ref ast.Node) ast.Node {
	if e := extendPointerType(p, ref); e != nil {
		return e
	}
	if s := indexedType(p, ref); s != nil {
		return s
	}
	return nil
}

func requireTypeRef(p *parse.Parser, b *cst.Branch) ast.Node {
	t := typeRef(p, b)
	if t == nil {
		p.Expected("type reference")
		return ast.InvalidGeneric
	}
	return t
}

// lhs_type ['const'] '*'
func extendPointerType(p *parse.Parser, ref ast.Node) *ast.PointerType {
	if !peekOperator(ast.OpPointer, p) && !peekKeyword(ast.KeywordConst, p) {
		return nil
	}
	t := &ast.PointerType{To: ref}
	p.Extend(ref, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(t, b)
		t.Const = keyword(ast.KeywordConst, p, b) != nil
		requireOperator(ast.OpPointer, p, b)
	})
	return t
}

// lhs_type '[' [ expression ] ']'
func indexedType(p *parse.Parser, ref ast.Node) *ast.IndexedType {
	if !peekOperator(ast.OpIndexStart, p) {
		return nil
	}
	t := &ast.IndexedType{ValueType: ref}
	p.Extend(ref, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(t, b)
		requireOperator(ast.OpIndexStart, p, b)
		if !peekOperator(ast.OpIndexEnd, p) {
			t.Index = requireExpression(p, b)
		}
		requireOperator(ast.OpIndexEnd, p, b)
	})
	return t
}
