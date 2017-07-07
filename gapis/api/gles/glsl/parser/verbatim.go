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
	"fmt"

	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/gapis/api/gles/glsl/ast"
)

type verbFormatter struct {
	f fmt.State
}

func (f *verbFormatter) writeCommaCst(cst []*parse.Leaf, pos int) {
}

type cstHolder struct {
	cst   *parse.Leaf
	token string
}

type commaCstHolder struct {
	cst []*parse.Leaf
	pos int
}

func (f *verbFormatter) format(n ...interface{}) {
	for _, n := range n {
		f.formatOne(n)
	}
}

func (f *verbFormatter) formatOne(n interface{}) {
	switch n := n.(type) {
	case string:
		f.f.Write([]byte(n))

	case cstHolder:
		if n.cst == nil {
			f.format(" " + n.token + " ")
		} else {
			n.cst.Prefix().WriteTo(f.f)
			f.format(n.token)
			n.cst.Suffix().WriteTo(f.f)
		}

	case commaCstHolder:
		if n.pos > 0 {
			var t *parse.Leaf
			if len(n.cst) > n.pos-1 {
				t = n.cst[n.pos-1]
			}
			f.format(cstHolder{t, ","})
		}

	///////////////////////////////// EXPRESSIONS /////////////////////////////////
	case *ast.BinaryExpr:
		f.format(n.Left, cstHolder{n.OpCst, n.Op.String()}, n.Right)

	case *ast.IndexExpr:
		f.format(n.Base, cstHolder{n.LBracketCst, "["}, n.Index, cstHolder{n.RBracketCst, "]"})

	case *ast.ConditionalExpr:
		f.format(n.Cond, cstHolder{n.QuestionCst, "?"}, n.TrueExpr,
			cstHolder{n.ColonCst, ":"}, n.FalseExpr)

	case *ast.UnaryExpr:
		pre := n.Op != ast.UoPostdec && n.Op != ast.UoPostinc
		if pre {
			f.format(cstHolder{n.OpCst, n.Op.String()})
		}
		f.format(n.Expr)
		if !pre {
			f.format(cstHolder{n.OpCst, n.Op.String()})
		}

	case *ast.DotExpr:
		f.format(n.Expr, cstHolder{n.DotCst, "."}, cstHolder{n.FieldCst, n.Field})

	case *ast.ConstantExpr:
		n.ValCst.WriteTo(f.f)

	case *ast.CallExpr:
		f.format(n.Callee, cstHolder{n.LParenCst, "("})
		if len(n.Args) > 0 || !n.VoidPresent {
			for i, a := range n.Args {
				f.format(commaCstHolder{n.CommaCst, i}, a)
			}
		} else {
			f.format(cstHolder{n.VoidCst, "void"})
		}
		f.format(cstHolder{n.RParenCst, ")"})

	case *ast.VarRefExpr:
		var name string
		if n.Sym == nil {
			name = "(nil variable)"
		} else {
			name = n.Sym.Name()
		}
		f.format(cstHolder{n.SymCst, name})

	case *ast.TypeConversionExpr:
		f.format(n.RetType)

	case *ast.ParenExpr:
		f.format(cstHolder{n.LParenCst, "("}, n.Expr, cstHolder{n.RParenCst, ")"})

	///////////////////////////////// STATEMENTS //////////////////////////////////

	case *ast.DeclarationStmt:
		f.format(n.Decl)

	case *ast.ExpressionStmt:
		f.format(n.Expr, cstHolder{n.SemicolonCst, ";"})

	case *ast.EmptyStmt:
		f.format(cstHolder{n.SemicolonCst, ";"})

	case *ast.IfStmt:
		f.format(cstHolder{n.IfCst, "if"}, cstHolder{n.LParenCst, "("}, n.IfExpr,
			cstHolder{n.RParenCst, ")"}, n.ThenStmt)

		if n.ElseStmt != nil {
			f.format(cstHolder{n.ElseCst, "else"}, n.ElseStmt)
		}

	case *ast.SwitchStmt:
		f.format(cstHolder{n.SwitchCst, "switch"}, cstHolder{n.LParenCst, "("}, n.Expr,
			cstHolder{n.RParenCst, ")"}, n.Stmts)

	case *ast.CompoundStmt:
		f.format(cstHolder{n.LBraceCst, "{"})
		for _, s := range n.Stmts {
			f.format(s)
		}
		f.format(cstHolder{n.RBraceCst, "}"})

	case *ast.CaseStmt:
		f.format(cstHolder{n.CaseCst, "case"}, n.Expr, cstHolder{n.ColonCst, ":"})

	case *ast.DefaultStmt:
		f.format(cstHolder{n.DefaultCst, "default"}, cstHolder{n.ColonCst, ":"})

	case *ast.WhileStmt:
		f.format(cstHolder{n.WhileCst, "while"}, cstHolder{n.LParenCst, "("}, n.Cond,
			cstHolder{n.RParenCst, ")"}, n.Stmt)

	case *ast.DoStmt:
		f.format(cstHolder{n.DoCst, "do"}, n.Stmt, cstHolder{n.WhileCst, "while"},
			cstHolder{n.LParenCst, "("}, n.Expr, cstHolder{n.RParenCst, ")"},
			cstHolder{n.SemicolonCst, ";"})

	case *ast.ForStmt:
		f.format(cstHolder{n.ForCst, "for"}, cstHolder{n.LParenCst, "("}, n.Init, n.Cond,
			cstHolder{n.Semicolon2Cst, ";"}, n.Loop, cstHolder{n.RParenCst, ")"}, n.Body)

	case *ast.ReturnStmt:
		f.format(cstHolder{n.ReturnCst, "return"})
		if n.Expr != nil {
			f.format(n.Expr)
		}
		f.format(cstHolder{n.SemicolonCst, ";"})

	case *ast.ContinueStmt:
		f.format(cstHolder{n.ContinueCst, "continue"}, cstHolder{n.SemicolonCst, ";"})

	case *ast.BreakStmt:
		f.format(cstHolder{n.BreakCst, "break"}, cstHolder{n.SemicolonCst, ";"})

	case *ast.DiscardStmt:
		f.format(cstHolder{n.DiscardCst, "discard"}, cstHolder{n.SemicolonCst, ";"})

	/////////////////////////////////// TYPES /////////////////////////////////////

	case *ast.BuiltinType:
		if n.Precision != ast.NoneP {
			f.format(cstHolder{n.PrecisionCst, n.Precision.String()})
		}
		f.format(cstHolder{n.TypeCst, n.Type.String()})

	case *ast.ArrayType:
		f.format(n.Base, cstHolder{n.LBracketCst, "["})
		if n.Size != nil {
			f.format(n.Size)
		}
		f.format(cstHolder{n.RBracketCst, "]"})

	case *ast.StructType:
		if !n.StructDef {
			f.format(cstHolder{n.NameCst, n.Sym.Name()})
			return
		}
		f.format(n.Sym)

	case *ast.LayoutQualifier:
		f.format(cstHolder{n.LayoutCst, "layout"}, cstHolder{n.LParenCst, "("})
		for i, id := range n.Ids {
			f.format(commaCstHolder{n.CommaCst, i}, cstHolder{id.NameCst, id.Name})
			if id.Value != nil {
				f.format(cstHolder{id.EqualCst, "="})
				id.ValueCst.WriteTo(f.f)
			}
		}
		f.format(cstHolder{n.RParenCst, ")"})

	case *ast.TypeQualifiers:
		if n.Invariant {
			f.format(cstHolder{n.InvariantCst, "invariant"})
		}

		if n.Interpolation != ast.IntNone {
			f.format(cstHolder{n.InterpolationCst, n.Interpolation.String()})
		}

		if n.Layout != nil {
			f.format(n.Layout)
		}

		switch n.Storage {
		case ast.StorOut, ast.StorIn, ast.StorConst, ast.StorUniform,
			ast.StorAttribute, ast.StorVarying:

			f.format(cstHolder{n.StorageCst, n.Storage.String()})
		case ast.StorCentroidIn:
			f.format(cstHolder{n.CentroidStorageCst, "centroid"}, cstHolder{n.StorageCst, "in"})
		case ast.StorCentroidOut:
			f.format(cstHolder{n.CentroidStorageCst, "centroid"}, cstHolder{n.StorageCst, "out"})
		}

	///////////////////////////////// DECLARATIONS ////////////////////////////////

	case *ast.PrecisionDecl:
		f.format(cstHolder{n.PrecisionCst, "precision"}, n.Type, cstHolder{n.SemicolonCst, ";"})

	case *ast.FunctionDecl:
		f.format(n.RetType, cstHolder{n.NameCst, n.SymName}, cstHolder{n.LParenCst, "("})
		if len(n.Params) > 0 || !n.VoidPresent {
			for i, p := range n.Params {
				f.format(commaCstHolder{n.CommaCst, i}, p)
			}
		} else {
			f.format(cstHolder{n.VoidCst, "void"})
		}
		f.format(cstHolder{n.RParenCst, ")"})

		if n.Stmts == nil {
			f.format(cstHolder{n.SemicolonCst, ";"})
		} else {
			f.format(n.Stmts)
		}

	case *ast.MultiVarDecl:
		if n.Quals != nil {
			f.format(n.Quals)
		}
		f.format(n.Type)
		if n.FunnyComma {
			f.format(cstHolder{n.FunnyCommaCst, ","})
		}
		for i, v := range n.Vars {
			f.format(commaCstHolder{n.CommaCst, i}, cstHolder{v.NameCst, v.Name()})
			if array, ok := v.SymType.(*ast.ArrayType); ok && array.Base.Equal(n.Type) {
				f.format(cstHolder{array.LBracketCst, "["})
				if array.Size != nil {
					f.format(array.Size)
				}
				f.format(cstHolder{array.RBracketCst, "]"})
			}
			if v.Init != nil {
				f.format(cstHolder{v.EqualCst, "="}, v.Init)
			}
		}
		f.format(cstHolder{n.SemicolonCst, ";"})

	case *ast.VariableSym:
		f.format(n.SymType, cstHolder{n.NameCst, n.Name()})
		if n.Init != nil {
			f.format(cstHolder{n.EqualCst, "="}, n.Init)
		}

	case *ast.StructSym:
		f.format(cstHolder{n.StructCst, "struct"}, cstHolder{n.NameCst, n.Name()},
			cstHolder{n.LBraceCst, "{"})
		for _, d := range n.Vars {
			f.format(d)
		}
		f.format(cstHolder{n.RBraceCst, "}"})

	case *ast.FuncParameterSym:
		if n.Const {
			f.format(cstHolder{n.ConstCst, "const"})
		}
		if n.Direction != ast.DirNone {
			f.format(cstHolder{n.DirectionCst, n.Direction.String()})
		}
		if t, ok := n.SymType.(*ast.ArrayType); ok && n.Name() != "" && n.ArrayAfterVar {
			f.format(t.Base, cstHolder{n.NameCst, n.Name()}, cstHolder{t.LBracketCst, "["},
				t.Size, cstHolder{t.RBracketCst, "]"})
		} else {
			f.format(n.SymType)
			if n.SymName != "" {
				f.format(cstHolder{n.NameCst, n.Name()})
			}
		}

	case *ast.LayoutDecl:
		f.format(n.Layout, cstHolder{n.UniformCst, "uniform"}, cstHolder{n.SemicolonCst, ";"})

	case *ast.InvariantDecl:
		f.format(cstHolder{n.InvariantCst, "invariant"})
		for i, v := range n.Vars {
			f.format(commaCstHolder{n.CommaCst, i}, cstHolder{v.SymCst, v.Sym.Name()})
		}
		f.format(cstHolder{n.SemicolonCst, ";"})

	case *ast.UniformBlock:
		if n.Layout != nil {
			f.format(n.Layout)
		}
		f.format(cstHolder{n.UniformCst, "uniform"}, cstHolder{n.NameCst, n.Name()},
			cstHolder{n.LBraceCst, "{"})
		for _, d := range n.Vars {
			f.format(d)
		}
		f.format(cstHolder{n.RBraceCst, "}"})

	case *ast.UniformDecl:
		f.format(n.Block)
		if n.SymName != "" {
			f.format(cstHolder{n.NameCst, n.Name()})
			if n.Size != nil {
				f.format(cstHolder{n.LBracketCst, "["}, n.Size, cstHolder{n.RBracketCst, "]"})
			}
		}
		f.format(cstHolder{n.SemicolonCst, ";"})

	///////////////////////////////// CONDITIONS //////////////////////////////////

	case *ast.ExpressionCond:
		f.format(n.Expr)

	case *ast.VarDeclCond:
		f.format(n.Sym)

	///////////////////////////////// OTHER ///////////////////////////////////////
	case *ast.Ast:
		for _, d := range n.Decls {
			f.format(d)
		}

	case fmt.Formatter:
		n.Format(f.f, 'v')

	default:
		f.format(fmt.Sprint(n))
	}
}
