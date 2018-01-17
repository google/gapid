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
	"github.com/google/gapid/gapil/ast"
)

// { import | extern | enum | alias | pseudonym | class | command | field }
func requireAPI(p *parse.Parser, cst *parse.Branch) *ast.API {
	api := &ast.API{}
	p.SetCST(api, cst)

	annotations := &ast.Annotations{}
	for !p.IsEOF() {
		parseAnnotations(annotations, p, cst)
		if i := import_(p, cst, annotations); i != nil {
			api.Imports = append(api.Imports, i)
		} else if e := extern(p, cst, annotations); e != nil {
			api.Externs = append(api.Externs, e)
		} else if e := enum(p, cst, annotations); e != nil {
			api.Enums = append(api.Enums, e)
		} else if a := alias(p, cst, annotations); a != nil {
			api.Aliases = append(api.Aliases, a)
		} else if pn := pseudonym(p, cst, annotations); pn != nil {
			api.Pseudonyms = append(api.Pseudonyms, pn)
		} else if c := class(p, cst, annotations); c != nil {
			api.Classes = append(api.Classes, c)
		} else if c := command(p, cst, annotations); c != nil {
			api.Commands = append(api.Commands, c)
		} else if s := subroutine(p, cst, annotations); s != nil {
			api.Subroutines = append(api.Subroutines, s)
		} else if c := definition(p, cst, annotations); c != nil {
			api.Definitions = append(api.Definitions, c)
		} else if c, i := cst, apiIndex(p, cst, annotations); i != nil {
			if api.Index != nil {
				p.ErrorAt(c, "Redefining API index")
			} else {
				api.Index = i
			}
		} else {
			api.Fields = append(api.Fields, requireField(p, cst, annotations))
		}
		if len(*annotations) != 0 {
			p.ErrorAt((*annotations)[0], "Annotation not consumed")
		}
	}
	return api
}

// { '@' name [ '(' { expression ',' } ')' ] }
func parseAnnotations(annotations *ast.Annotations, p *parse.Parser, cst *parse.Branch) {
	for peekOperator(ast.OpAnnotation, p) {
		a := &ast.Annotation{}
		*annotations = append(*annotations, a)
		p.ParseBranch(cst, func(p *parse.Parser, cst *parse.Branch) {
			requireOperator(ast.OpAnnotation, p, cst)
			p.SetCST(a, cst)
			a.Name = requireIdentifier(p, cst)
			if operator(ast.OpListStart, p, cst) {
				for !operator(ast.OpListEnd, p, cst) {
					if p.IsEOF() {
						p.Error("end of file reached while looking for '%s'", ast.OpListEnd)
						break
					}
					if len(a.Arguments) > 0 {
						requireOperator(ast.OpListSeparator, p, cst)
					}
					e := requireExpression(p, cst)
					a.Arguments = append(a.Arguments, e)
				}
			}
		})
	}
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
func import_(p *parse.Parser, cst *parse.Branch, a *ast.Annotations) *ast.Import {
	if !peekKeyword(ast.KeywordImport, p) {
		return nil
	}
	i := &ast.Import{}
	consumeAnnotations(&i.Annotations, a)
	p.ParseBranch(cst, func(p *parse.Parser, cst *parse.Branch) {
		p.SetCST(i, cst)
		requireKeyword(ast.KeywordImport, p, cst)
		i.Path = requireString(p, cst)
	})
	return i
}
