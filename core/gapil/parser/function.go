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
	"github.com/google/gapid/core/gapil/ast"
	"github.com/google/gapid/core/text/parse"
)

// type name '(' [ param { ',' param } } ')' [ block ]
func function(f *ast.Function, p *parse.Parser, cst *parse.Branch, withBlock bool) *ast.Function {
	p.SetCST(f, cst)
	var result *ast.Parameter
	p.ParseBranch(cst, func(p *parse.Parser, cst *parse.Branch) {
		result = &ast.Parameter{Type: requireTypeRef(p, cst)}
		p.SetCST(result, cst)
	})
	f.Generic = requireGeneric(p, cst)
	requireOperator(ast.OpListStart, p, cst)
	for !operator(ast.OpListEnd, p, cst) {
		if p.IsEOF() {
			p.Error("end of file reached while looking for '%s'", ast.OpListEnd)
			break
		}
		if len(f.Parameters) > 0 {
			operator(ast.OpListSeparator, p, cst)
		}
		p.ParseBranch(cst, func(p *parse.Parser, cst *parse.Branch) {
			f.Parameters = append(f.Parameters, parameter(p, cst))
		})
	}
	f.Parameters = append(f.Parameters, result)
	if withBlock {
		f.Block = requireBlock(p, cst)
	}
	return f
}

// [ annotations ] [ 'this' ] type [annotations] name
func parameter(p *parse.Parser, cst *parse.Branch) *ast.Parameter {
	param := &ast.Parameter{}
	parseAnnotations(&param.Annotations, p, cst)
	p.SetCST(param, cst)
	if keyword(ast.KeywordThis, p, cst) != nil {
		param.This = true
	}
	param.Type = requireTypeRef(p, cst)
	parseAnnotations(&param.Annotations, p, cst)
	param.Name = requireIdentifier(p, cst)
	return param
}

// { annotation } 'extern' function
func extern(p *parse.Parser, cst *parse.Branch, a *ast.Annotations) *ast.Function {
	if !peekKeyword(ast.KeywordExtern, p) {
		return nil
	}
	e := &ast.Function{}
	consumeAnnotations(&e.Annotations, a)
	p.ParseBranch(cst, func(p *parse.Parser, cst *parse.Branch) {
		p.SetCST(e, cst)
		requireKeyword(ast.KeywordExtern, p, cst)
		function(e, p, cst, false)
	})
	return e
}

// { annotation } 'cmd' function
func command(p *parse.Parser, cst *parse.Branch, a *ast.Annotations) *ast.Function {
	if !peekKeyword(ast.KeywordCmd, p) {
		return nil
	}
	cmd := &ast.Function{}
	consumeAnnotations(&cmd.Annotations, a)
	p.ParseBranch(cst, func(p *parse.Parser, cst *parse.Branch) {
		p.SetCST(cmd, cst)
		requireKeyword(ast.KeywordCmd, p, cst)
		function(cmd, p, cst, true)
	})
	return cmd
}

// 'sub' function
func subroutine(p *parse.Parser, cst *parse.Branch, a *ast.Annotations) *ast.Function {
	if !peekKeyword(ast.KeywordSub, p) {
		return nil
	}
	cmd := &ast.Function{}
	consumeAnnotations(&cmd.Annotations, a)
	p.ParseBranch(cst, func(p *parse.Parser, cst *parse.Branch) {
		p.SetCST(cmd, cst)
		requireKeyword(ast.KeywordSub, p, cst)
		function(cmd, p, cst, true)
	})
	return cmd
}
