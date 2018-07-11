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

// type name '(' [ param { ',' param } } ')' [ block ]
func function(f *ast.Function, p *parse.Parser, b *cst.Branch, withBlock bool) *ast.Function {
	p.SetCST(f, b)
	var result *ast.Parameter
	p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
		result = &ast.Parameter{Type: requireTypeRef(p, b)}
		p.SetCST(result, b)
	})
	f.Generic = requireGeneric(p, b)
	requireOperator(ast.OpListStart, p, b)
	for !operator(ast.OpListEnd, p, b) {
		if p.IsEOF() {
			p.Error("end of file reached while looking for '%s'", ast.OpListEnd)
			break
		}
		if len(f.Parameters) > 0 {
			operator(ast.OpListSeparator, p, b)
		}
		p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
			f.Parameters = append(f.Parameters, parameter(p, b))
		})
	}
	f.Parameters = append(f.Parameters, result)
	if withBlock {
		f.Block = requireBlock(p, b)
	}
	return f
}

// [ annotations ] [ 'this' ] type [annotations] name
func parameter(p *parse.Parser, b *cst.Branch) *ast.Parameter {
	param := &ast.Parameter{}
	parseAnnotations(&param.Annotations, p, b)
	p.SetCST(param, b)
	if keyword(ast.KeywordThis, p, b) != nil {
		param.This = true
	}
	param.Type = requireTypeRef(p, b)
	parseAnnotations(&param.Annotations, p, b)
	param.Name = requireIdentifier(p, b)
	return param
}

// { annotation } 'extern' function
func extern(p *parse.Parser, b *cst.Branch, a *ast.Annotations) *ast.Function {
	if !peekKeyword(ast.KeywordExtern, p) {
		return nil
	}
	e := &ast.Function{}
	consumeAnnotations(&e.Annotations, a)
	p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(e, b)
		requireKeyword(ast.KeywordExtern, p, b)
		function(e, p, b, false)
	})
	return e
}

// { annotation } 'cmd' function
func command(p *parse.Parser, b *cst.Branch, a *ast.Annotations) *ast.Function {
	if !peekKeyword(ast.KeywordCmd, p) {
		return nil
	}
	cmd := &ast.Function{}
	consumeAnnotations(&cmd.Annotations, a)
	p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(cmd, b)
		requireKeyword(ast.KeywordCmd, p, b)
		function(cmd, p, b, true)
	})
	return cmd
}

// 'sub' function
func subroutine(p *parse.Parser, b *cst.Branch, a *ast.Annotations) *ast.Function {
	if !peekKeyword(ast.KeywordSub, p) {
		return nil
	}
	cmd := &ast.Function{}
	consumeAnnotations(&cmd.Annotations, a)
	p.ParseBranch(b, func(p *parse.Parser, b *cst.Branch) {
		p.SetCST(cmd, b)
		requireKeyword(ast.KeywordSub, p, b)
		function(cmd, p, b, true)
	})
	return cmd
}
