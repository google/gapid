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

package generator

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

type rewriter struct {
	uid   int
	funcs map[string]*entry
	fun   *entry
}

func (info *introspection) rewrite(ctx context.Context) {
	// Do the initial rewrites
	for _, e := range info.entries {
		if e.Func != nil {
			r := &rewriter{
				funcs: info.byName,
				fun:   e,
			}
			r.rewriteBlock(e.Func.Body)
		}
	}
	// Add any additional function forms we need
	for _, e := range info.entries {
		generateFor(ctx, info.fset, e)
	}
	// Map and default values needed to the first file they are seen in
	defined := map[string]bool{}
	for _, e := range info.entries {
		if e.DefaultName != "" {
			if _, seen := defined[e.DefaultName]; !seen {
				defined[e.DefaultName] = true
				generateDefault(e.Source, e.DefaultName, e.ResultType)
			}
		}
	}
}

func (r *rewriter) UID() int {
	r.uid++
	return r.uid
}

func (r *rewriter) rewriteBlock(stmt *ast.BlockStmt) {
	// We need to walk and rewrite all statements in all blocks
	block := []ast.Stmt{}
	for _, stmt := range stmt.List {
		block = append(block, r.rewriteStatement(&block, stmt))
	}
	stmt.List = block
}

func (r *rewriter) rewriteStatement(stmts *[]ast.Stmt, stmt ast.Stmt) ast.Stmt {
	switch stmt := stmt.(type) {
	case *ast.BlockStmt:
		r.rewriteBlock(stmt)
	case *ast.ExprStmt:
		call := r.checkCall(stmt.X)
		if call != nil {
			// Direct call, replace it
			return r.errTest("err", &ast.AssignStmt{
				Lhs: []ast.Expr{
					&ast.Ident{Name: "_"},
					&ast.Ident{Name: "err"},
				},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{call},
			})
		}
	case *ast.AssignStmt:
		for i, expr := range stmt.Lhs {
			stmt.Lhs[i] = r.liftValueExpression(stmts, expr)
		}
		if len(stmt.Lhs) != 2 || len(stmt.Rhs) != 1 || r.checkCall(stmt.Rhs[0]) == nil {
			for i, expr := range stmt.Rhs {
				stmt.Rhs[i] = r.liftValueExpression(stmts, expr)
			}
		}
	case *ast.ReturnStmt:
		if len(stmt.Results) == 1 {
			r.checkCall(stmt.Results[0])
		} else {
			for i, expr := range stmt.Results {
				stmt.Results[i] = r.liftValueExpression(stmts, expr)
			}
		}
	case *ast.IfStmt:
		count := len(*stmts)
		stmt.Cond = r.liftBoolExpression(stmts, stmt.Cond)
		if stmt.Init != nil {
			stmt.Init = r.rewriteStatement(stmts, stmt.Init)
		} else if len(*stmts) == count+1 {
			stmt.Init = (*stmts)[count]
			*stmts = (*stmts)[:count]
		}
		r.rewriteBlock(stmt.Body)
		if stmt.Else != nil {
			count = len(*stmts)
			stmt.Else = r.rewriteStatement(stmts, stmt.Else)
			if len(*stmts) != count {
				stmt.Else = &ast.BadStmt{}
			}
		}
	case *ast.ForStmt:
		id := r.UID()
		mark := &ast.Ident{Name: fmt.Sprint("forMark", id)}
		err := &ast.Ident{Name: fmt.Sprint("forErr", id)}
		*stmts = append(*stmts,
			&ast.AssignStmt{
				Lhs: []ast.Expr{mark},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{
					&ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   &ast.Ident{Name: r.fun.Params[0].Names[0].Name},
							Sel: &ast.Ident{Name: "PreMark"},
						},
						Args: []ast.Expr{},
					},
				},
			},
			&ast.DeclStmt{
				Decl: &ast.GenDecl{
					Tok: token.VAR,
					Specs: []ast.Spec{&ast.ValueSpec{
						Names: []*ast.Ident{err},
						Type:  &ast.Ident{Name: "error"},
					}},
				},
			},
		)
		stmt.Cond = r.liftBoolExpression(stmts, stmt.Cond)
		r.rewriteBlock(stmt.Body)
		stmt.Body.List = append([]ast.Stmt{
			r.errTest(err.Name, &ast.AssignStmt{
				Lhs: []ast.Expr{mark, err},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{
					&ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   &ast.Ident{Name: r.fun.Params[0].Names[0].Name},
							Sel: &ast.Ident{Name: "MustProgress"},
						},
						Args: []ast.Expr{mark},
					},
				},
			}),
		}, stmt.Body.List...)
	case *ast.DeclStmt:
		// TODO:
	case *ast.LabeledStmt:
		stmt.Stmt = r.rewriteStatement(stmts, stmt.Stmt)
	case *ast.SendStmt:
		stmt.Chan = r.liftValueExpression(stmts, stmt.Chan)
		stmt.Value = r.liftValueExpression(stmts, stmt.Value)
	case *ast.IncDecStmt:
		stmt.X = r.liftValueExpression(stmts, stmt.X)
	case *ast.SwitchStmt:
		stmt.Init = r.rewriteStatement(stmts, stmt.Init)
		r.rewriteBlock(stmt.Body)
	case *ast.CaseClause:
		var extras []ast.Stmt
		for i, expr := range stmt.List {
			stmt.List[i] = r.liftBoolExpression(&extras, expr)
		}
		body := []ast.Stmt{}
		if len(extras) != 0 {
			body = append(body, &ast.BadStmt{})
			body = append(body, extras...)
			body = append(body, &ast.BadStmt{})
		}
		for _, stmt := range stmt.Body {
			body = append(body, r.rewriteStatement(&body, stmt))
		}
		stmt.Body = body
	case *ast.TypeSwitchStmt:
		stmt.Init = r.rewriteStatement(stmts, stmt.Init)
		stmt.Assign = r.rewriteStatement(stmts, stmt.Assign)
		r.rewriteBlock(stmt.Body)
	case *ast.SelectStmt:
		r.rewriteBlock(stmt.Body)
	case *ast.CommClause:
		stmt.Comm = r.rewriteStatement(stmts, stmt.Comm)
		body := []ast.Stmt{}
		for _, stmt := range stmt.Body {
			body = append(body, r.rewriteStatement(&body, stmt))
		}
		stmt.Body = body
	case *ast.RangeStmt:
		stmt.X = r.liftValueExpression(stmts, stmt.X)
		r.rewriteBlock(stmt.Body)
	}
	return stmt
}

func (r *rewriter) liftBoolExpression(stmts *[]ast.Stmt, expr ast.Expr) ast.Expr {
	call := r.checkCall(expr)
	if call == nil {
		r.liftFromExpression(stmts, expr)
		return expr
	}
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: "lingo"},
			Sel: &ast.Ident{Name: "WasOk"},
		},
		Args: []ast.Expr{call},
	}
}

func (r *rewriter) liftValueExpression(stmts *[]ast.Stmt, expr ast.Expr) ast.Expr {
	r.liftFromExpression(stmts, expr)
	call := r.checkCall(expr)
	if call == nil {
		return expr
	}
	if len(call.Args) == 0 {
		return call.Fun
	}
	val := &ast.Ident{Name: fmt.Sprint("val", r.UID())}
	errName := fmt.Sprint("err", r.UID())
	*stmts = append(*stmts,
		&ast.AssignStmt{
			Lhs: []ast.Expr{val, &ast.Ident{Name: errName}},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{call},
		},
		r.errTest(errName, nil),
	)
	return val
}

func (r *rewriter) liftFromExpression(stmts *[]ast.Stmt, expr ast.Expr) {
	switch expr := expr.(type) {
	case *ast.UnaryExpr:
		if expr.Op == token.NOT {
			expr.X = r.liftBoolExpression(stmts, expr.X)
		} else {
			expr.X = r.liftValueExpression(stmts, expr.X)
		}
	case *ast.BinaryExpr:
		if expr.Op == token.LAND || expr.Op == token.LOR {
			expr.X = r.liftBoolExpression(stmts, expr.X)
			expr.Y = r.liftBoolExpression(stmts, expr.Y)
		} else {
			expr.X = r.liftValueExpression(stmts, expr.X)
			expr.Y = r.liftValueExpression(stmts, expr.Y)
		}
	case *ast.CallExpr:
		for i, arg := range expr.Args {
			expr.Args[i] = r.liftValueExpression(stmts, arg)
		}
	case *ast.CompositeLit:
		for i, arg := range expr.Elts {
			expr.Elts[i] = r.liftValueExpression(stmts, arg)
		}
	case *ast.ParenExpr:
		expr.X = r.liftValueExpression(stmts, expr.X)
	case *ast.SelectorExpr:
		expr.X = r.liftValueExpression(stmts, expr.X)
	case *ast.IndexExpr:
		expr.X = r.liftValueExpression(stmts, expr.X)
		expr.Index = r.liftValueExpression(stmts, expr.Index)
	case *ast.SliceExpr:
		expr.X = r.liftValueExpression(stmts, expr.X)
		expr.Low = r.liftValueExpression(stmts, expr.Low)
		expr.High = r.liftValueExpression(stmts, expr.High)
		expr.Max = r.liftValueExpression(stmts, expr.Max)
	case *ast.StarExpr:
		expr.X = r.liftValueExpression(stmts, expr.X)
	case *ast.KeyValueExpr:
		expr.Value = r.liftValueExpression(stmts, expr.Value)
	}
}

func (r *rewriter) errTest(errName string, init ast.Stmt) *ast.IfStmt {
	results := []ast.Expr{}
	if r.fun.ResultType == "" {
	} else if !r.fun.IsPointer {
		if r.fun.DefaultName == "" {
			r.fun.DefaultName = strings.Replace(r.fun.ResultType, ".", "_", -1) + "Default"
		}
		results = []ast.Expr{
			&ast.Ident{Name: r.fun.DefaultName},
			&ast.Ident{Name: errName},
		}
	} else {
		results = []ast.Expr{
			&ast.Ident{Name: "nil"},
			&ast.Ident{Name: errName},
		}
	}
	return &ast.IfStmt{
		Init: init,
		Cond: &ast.BinaryExpr{
			X:  &ast.Ident{Name: errName},
			Op: token.NEQ,
			Y:  &ast.Ident{Name: "nil"},
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.ReturnStmt{Results: results},
		}},
	}
}

func (r *rewriter) checkCall(expr ast.Expr) *ast.CallExpr {
	call, found := expr.(*ast.CallExpr)
	if !found {
		return nil
	}
	name, found := call.Fun.(*ast.Ident)
	if !found {
		return nil
	}
	invoke, found := r.funcs[name.Name]
	if !found || !invoke.Parser {
		return nil
	}
	invoke.Called = true
	call.Fun = &ast.Ident{Name: invoke.Name + parserName}
	return call
}
