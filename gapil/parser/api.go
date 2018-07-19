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

// { import | extern | enum | pseudonym | class | command | field }
func (p *parser) requireAPI(b *cst.Branch) *ast.API {
	api := &ast.API{}
	p.mappings.Add(api, b)

	annotations := &ast.Annotations{}
	for !p.IsEOF() {
		p.parseAnnotations(annotations, b)
		if i := p.import_(b, annotations); i != nil {
			api.Imports = append(api.Imports, i)
		} else if e := p.extern(b, annotations); e != nil {
			api.Externs = append(api.Externs, e)
		} else if e := p.enum(b, annotations); e != nil {
			api.Enums = append(api.Enums, e)
		} else if pn := p.pseudonym(b, annotations); pn != nil {
			api.Pseudonyms = append(api.Pseudonyms, pn)
		} else if c := p.class(b, annotations); c != nil {
			api.Classes = append(api.Classes, c)
		} else if c := p.command(b, annotations); c != nil {
			api.Commands = append(api.Commands, c)
		} else if s := p.subroutine(b, annotations); s != nil {
			api.Subroutines = append(api.Subroutines, s)
		} else if c := p.definition(b, annotations); c != nil {
			api.Definitions = append(api.Definitions, c)
		} else if c, i := b, p.apiIndex(b, annotations); i != nil {
			if api.Index != nil {
				p.ErrorAt(c, "Redefining API index")
			} else {
				api.Index = i
			}
		} else {
			api.Fields = append(api.Fields, p.requireField(b, annotations))
		}
		if len(*annotations) != 0 {
			cst := p.mappings.CST((*annotations)[0])
			p.ErrorAt(cst, "Annotation not consumed")
		}
	}
	return api
}

// { '@' name [ '(' { expression ',' } ')' ] }
func (p *parser) parseAnnotations(annotations *ast.Annotations, b *cst.Branch) {
	for p.peekOperator(ast.OpAnnotation) {
		a := &ast.Annotation{}
		*annotations = append(*annotations, a)
		p.ParseBranch(b, func(b *cst.Branch) {
			p.requireOperator(ast.OpAnnotation, b)
			p.mappings.Add(a, b)
			a.Name = p.requireIdentifier(b)
			if p.operator(ast.OpListStart, b) {
				for !p.operator(ast.OpListEnd, b) {
					if p.IsEOF() {
						p.Error("end of file reached while looking for '%s'", ast.OpListEnd)
						break
					}
					if len(a.Arguments) > 0 {
						p.requireOperator(ast.OpListSeparator, b)
					}
					e := p.requireExpression(b)
					a.Arguments = append(a.Arguments, e)
				}
			}
		})
	}
}

func annotationsOrNil(a ast.Annotations) ast.Annotations {
	if len(a) > 0 {
		return a
	}
	return nil
}

func consumeAnnotations(dst *ast.Annotations, src *ast.Annotations) {
	if src == nil || len(*src) == 0 {
		return
	}
	l := append(*dst, (*src)...)
	*src = (*src)[0:0]
	*dst = l
}

// { annotation } 'import' [ identifier ] '"' path '""'
func (p *parser) import_(b *cst.Branch, a *ast.Annotations) *ast.Import {
	if !p.peekKeyword(ast.KeywordImport) {
		return nil
	}
	i := &ast.Import{}
	consumeAnnotations(&i.Annotations, a)
	p.ParseBranch(b, func(b *cst.Branch) {
		p.mappings.Add(i, b)
		p.requireKeyword(ast.KeywordImport, b)
		i.Path = p.requireString(b)
	})
	return i
}
