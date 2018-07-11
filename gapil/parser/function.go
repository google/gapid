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

// type name '(' [ param { ',' param } } ')' [ block ]
func (p *parser) function(f *ast.Function, b *cst.Branch, withBlock bool) *ast.Function {
	p.mappings.Add(f, b)
	var result *ast.Parameter
	p.ParseBranch(b, func(b *cst.Branch) {
		result = &ast.Parameter{Type: p.requireTypeRef(b)}
		p.mappings.Add(result, b)
	})
	f.Generic = p.requireGeneric(b)
	p.requireOperator(ast.OpListStart, b)
	for !p.operator(ast.OpListEnd, b) {
		if p.IsEOF() {
			p.Error("end of file reached while looking for '%s'", ast.OpListEnd)
			break
		}
		if len(f.Parameters) > 0 {
			p.operator(ast.OpListSeparator, b)
		}
		p.ParseBranch(b, func(b *cst.Branch) {
			f.Parameters = append(f.Parameters, p.parameter(b))
		})
	}
	f.Parameters = append(f.Parameters, result)
	if withBlock {
		f.Block = p.requireBlock(b)
	}
	return f
}

// [ annotations ] [ 'this' ] type [annotations] name
func (p *parser) parameter(b *cst.Branch) *ast.Parameter {
	param := &ast.Parameter{}
	p.parseAnnotations(&param.Annotations, b)
	p.mappings.Add(param, b)
	if p.keyword(ast.KeywordThis, b) != nil {
		param.This = true
	}
	param.Type = p.requireTypeRef(b)
	p.parseAnnotations(&param.Annotations, b)
	param.Name = p.requireIdentifier(b)
	return param
}

// { annotation } 'extern' function
func (p *parser) extern(b *cst.Branch, a *ast.Annotations) *ast.Function {
	if !p.peekKeyword(ast.KeywordExtern) {
		return nil
	}
	e := &ast.Function{}
	consumeAnnotations(&e.Annotations, a)
	p.ParseBranch(b, func(b *cst.Branch) {
		p.mappings.Add(e, b)
		p.requireKeyword(ast.KeywordExtern, b)
		p.function(e, b, false)
	})
	return e
}

// { annotation } 'cmd' function
func (p *parser) command(b *cst.Branch, a *ast.Annotations) *ast.Function {
	if !p.peekKeyword(ast.KeywordCmd) {
		return nil
	}
	cmd := &ast.Function{}
	consumeAnnotations(&cmd.Annotations, a)
	p.ParseBranch(b, func(b *cst.Branch) {
		p.mappings.Add(cmd, b)
		p.requireKeyword(ast.KeywordCmd, b)
		p.function(cmd, b, true)
	})
	return cmd
}

// 'sub' function
func (p *parser) subroutine(b *cst.Branch, a *ast.Annotations) *ast.Function {
	if !p.peekKeyword(ast.KeywordSub) {
		return nil
	}
	cmd := &ast.Function{}
	consumeAnnotations(&cmd.Annotations, a)
	p.ParseBranch(b, func(b *cst.Branch) {
		p.mappings.Add(cmd, b)
		p.requireKeyword(ast.KeywordSub, b)
		p.function(cmd, b, true)
	})
	return cmd
}
