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

// { annotation } 'class' identifer '{' { field } '}'
func (p *parser) class(b *cst.Branch, a *ast.Annotations) *ast.Class {
	if !p.peekKeyword(ast.KeywordClass) {
		return nil
	}
	c := &ast.Class{}
	consumeAnnotations(&c.Annotations, a)
	p.ParseBranch(b, func(b *cst.Branch) {
		p.mappings.Add(c, b)
		p.requireKeyword(ast.KeywordClass, b)
		c.Name = p.requireIdentifier(b)
		p.requireOperator(ast.OpBlockStart, b)
		for !p.operator(ast.OpBlockEnd, b) {
			c.Fields = append(c.Fields, p.requireField(b, nil))
		}
	})
	return c
}

// { annotation } type identifier [ '=' expression ] [ ',' ]
func (p *parser) requireField(b *cst.Branch, a *ast.Annotations) *ast.Field {
	f := &ast.Field{}
	consumeAnnotations(&f.Annotations, a)
	p.ParseBranch(b, func(b *cst.Branch) {
		p.mappings.Add(f, b)
		p.parseAnnotations(&f.Annotations, b)
		f.Type = p.requireTypeRef(b)
		f.Name = p.requireIdentifier(b)
		if p.operator(ast.OpAssign, b) {
			f.Default = p.requireExpression(b)
		}
		p.operator(ast.OpListSeparator, b)
	})
	return f
}

// api_index number
func (p *parser) apiIndex(b *cst.Branch, a *ast.Annotations) *ast.Number {
	if !p.peekKeyword(ast.KeywordApiIndex) {
		return nil
	}
	if len(*a) != 0 {
		cst := p.mappings.CST((*a)[0])
		p.ErrorAt(cst, "Annotation on api_index not allowed")
		return nil
	}
	var f *ast.Number
	p.ParseBranch(b, func(b *cst.Branch) {
		p.requireKeyword(ast.KeywordApiIndex, b)
		f = p.requireNumber(b)
	})
	return f
}

// { annotation } 'define' identifier expression
func (p *parser) definition(b *cst.Branch, a *ast.Annotations) *ast.Definition {
	if !p.peekKeyword(ast.KeywordDefine) {
		return nil
	}
	d := &ast.Definition{}
	consumeAnnotations(&d.Annotations, a)
	p.ParseBranch(b, func(b *cst.Branch) {
		p.mappings.Add(d, b)
		p.requireKeyword(ast.KeywordDefine, b)
		d.Name = p.requireIdentifier(b)
		d.Expression = p.requireExpression(b)
	})
	return d
}

// { annotation } ( 'enum' | 'bitfield' ) name [ : type ] '{' { identifier '=' expression [ ',' ] } '}'
func (p *parser) enum(b *cst.Branch, a *ast.Annotations) *ast.Enum {
	if !p.peekKeyword(ast.KeywordEnum) && !p.peekKeyword(ast.KeywordBitfield) {
		return nil
	}
	s := &ast.Enum{}
	consumeAnnotations(&s.Annotations, a)
	p.ParseBranch(b, func(b *cst.Branch) {
		p.mappings.Add(s, b)
		if p.keyword(ast.KeywordEnum, b) == nil {
			p.requireKeyword(ast.KeywordBitfield, b)
			s.IsBitfield = true
		}
		s.Name = p.requireIdentifier(b)
		if p.operator(ast.OpExtends, b) {
			s.NumberType = p.requireTypeRef(b)
		}
		p.requireOperator(ast.OpBlockStart, b)
		for !p.operator(ast.OpBlockEnd, b) {
			p.ParseBranch(b, func(b *cst.Branch) {
				entry := &ast.EnumEntry{}
				p.mappings.Add(entry, b)
				entry.Name = p.requireIdentifier(b)
				p.requireOperator(ast.OpAssign, b)
				entry.Value = p.requireNumber(b)
				p.operator(ast.OpListSeparator, b)
				s.Entries = append(s.Entries, entry)
			})
		}
	})
	return s
}

// { annotation } 'type' type identifier
func (p *parser) pseudonym(b *cst.Branch, a *ast.Annotations) *ast.Pseudonym {
	if !p.peekKeyword(ast.KeywordPseudonym) {
		return nil
	}
	s := &ast.Pseudonym{}
	consumeAnnotations(&s.Annotations, a)
	p.ParseBranch(b, func(b *cst.Branch) {
		p.mappings.Add(s, b)
		p.requireKeyword(ast.KeywordPseudonym, b)
		s.To = p.requireTypeRef(b)
		s.Name = p.requireIdentifier(b)
	})
	return s
}

// [const] generic [.name] { extend_type }
func (p *parser) typeRef(b *cst.Branch) ast.Node {
	var ref ast.Node

	if p.peekKeyword(ast.KeywordConst) {
		c := &ast.PreConst{}
		p.ParseBranch(b, func(b *cst.Branch) {
			p.mappings.Add(c, b)
			p.requireKeyword(ast.KeywordConst, b)
			el := p.requireTypeBase(b)
			ptr := &ast.PointerType{To: el}
			p.Extend(p.mappings.CST(el), func(b *cst.Branch) {
				p.mappings.Add(ptr, b)
				p.requireOperator(ast.OpPointer, b)
			})
			c.Type = ptr
		})
		ref = c
	} else {
		ref = p.typeBase(b)
	}

	if ref == nil {
		return nil
	}

	for {
		if t := p.extendTypeRef(ref); t != nil {
			ref = t
		} else {
			break
		}
	}
	return ref
}

// generic [.name]
func (p *parser) typeBase(b *cst.Branch) ast.Node {
	g := p.generic(b)
	if g == nil {
		return nil
	}

	if p.peekOperator(ast.OpMember) {
		t := &ast.Imported{From: g.Name}
		p.Extend(p.mappings.CST(g), func(b *cst.Branch) {
			p.mappings.Add(t, b)
			p.requireOperator(ast.OpMember, b)
			t.Name = p.requireIdentifier(b)
		})
		return t
	}
	return g
}

func (p *parser) requireTypeBase(b *cst.Branch) ast.Node {
	t := p.typeBase(b)
	if t == nil {
		p.Expected("type")
		return ast.InvalidGeneric
	}
	return t
}

// ref ( pointer_type | static_array_type )
func (p *parser) extendTypeRef(ref ast.Node) ast.Node {
	if e := p.extendPointerType(ref); e != nil {
		return e
	}
	if s := p.indexedType(ref); s != nil {
		return s
	}
	return nil
}

func (p *parser) requireTypeRef(b *cst.Branch) ast.Node {
	t := p.typeRef(b)
	if t == nil {
		p.Expected("type reference")
		return ast.InvalidGeneric
	}
	return t
}

// lhs_type ['const'] '*'
func (p *parser) extendPointerType(ref ast.Node) *ast.PointerType {
	if !p.peekOperator(ast.OpPointer) && !p.peekKeyword(ast.KeywordConst) {
		return nil
	}
	t := &ast.PointerType{To: ref}
	p.Extend(p.mappings.CST(ref), func(b *cst.Branch) {
		p.mappings.Add(t, b)
		t.Const = p.keyword(ast.KeywordConst, b) != nil
		p.requireOperator(ast.OpPointer, b)
	})
	return t
}

// lhs_type '[' [ expression ] ']'
func (p *parser) indexedType(ref ast.Node) *ast.IndexedType {
	if !p.peekOperator(ast.OpIndexStart) {
		return nil
	}
	t := &ast.IndexedType{ValueType: ref}
	p.Extend(p.mappings.CST(ref), func(b *cst.Branch) {
		p.mappings.Add(t, b)
		p.requireOperator(ast.OpIndexStart, b)
		if !p.peekOperator(ast.OpIndexEnd) {
			t.Index = p.requireExpression(b)
		}
		p.requireOperator(ast.OpIndexEnd, b)
	})
	return t
}
