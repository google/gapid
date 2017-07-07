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
	"strings"

	"github.com/google/gapid/gapis/api/gles/glsl/ast"
)

// indentLen is the size of indent for pretty printing.
const indentLen = 4

// prettyFormatter is a structure holding the state of the pretty printer.
type prettyFormatter struct {
	f       fmt.State
	indent  int  // The current indentation level.
	newline bool // Whether the previous character was newline.
}

func newPrettyFormatter(f fmt.State) *prettyFormatter {
	return &prettyFormatter{f: f}
}

func makeIndent(indent int) string {
	return fmt.Sprintf("%*s", indent*indentLen, "")
}

type astFormatter struct {
	ast interface{}
}

func (af astFormatter) Format(f fmt.State, c rune) {
	if f.Flag('#') {
		f := verbFormatter{f}
		f.format(af.ast)
	} else {
		f := newPrettyFormatter(f)
		f.format(af.ast)
	}
}

// Formatter is a helper function which turns any AST node into something that can be printed
// with %v. The returned object's default format will print the tree under the ast node in a
// reindented form. The alternate format flag (%#v) will print the node while preserving original
// whitespace, if this is present in the ***Cst nodes of the tree.
func Formatter(e interface{}) fmt.Formatter { return astFormatter{e} }

func (f *prettyFormatter) format(nodes ...interface{}) {
	for _, n := range nodes {
		f.formatOne(n)
	}
}

// A helper class for the pretty printer which handles the different ways of formatting
// statements depending on the context and their type. Field "inner" means this statement is not
// a stand-alone statement (e.g. in a CompoundStmt) but it is a sub-statement of another
// statement (e.g., an IfExpr). Compare the different indentation depending on the statement type
// when inner is true:
//
// if(foo) ;
// if(foo) {
//   bar();
// }
// if(foo)
//   bar();
//
// and when inner is false:
// ;
// {
//   bar();
// }
// bar();
type formatStatement struct {
	inner     bool
	statement interface{}
}

func (f *prettyFormatter) formatOne(n interface{}) {
	switch n := n.(type) {
	case string:
		f.f.Write([]byte(n))
		if l := len(n); l > 0 {
			f.newline = n[l-1] == '\n'
		}

	case formatStatement:
		if _, ok := n.statement.(*ast.CompoundStmt); ok {
			if n.inner {
				f.format(" ")
			} else {
				f.format(makeIndent(f.indent))
			}
			f.format(n.statement)
		} else if _, ok := n.statement.(*ast.EmptyStmt); ok {
			if n.inner {
				f.format(" ")
			} else {
				f.format(makeIndent(f.indent))
			}
			f.format(n.statement)
		} else {
			if n.inner {
				f.indent++
				f.format("\n")
			}
			f.format(makeIndent(f.indent))
			f.format(n.statement)
			if n.inner {
				f.indent--
			}
		}

	///////////////////////////////// EXPRESSIONS /////////////////////////////////
	case *ast.BinaryExpr:
		f.format(n.Left)
		if n.Op != ast.BoComma {
			f.format(" ")
		}
		f.format(n.Op.String(), " ", n.Right)

	case *ast.IndexExpr:
		f.format(n.Base, "[", n.Index, "]")

	case *ast.ConditionalExpr:
		f.format(n.Cond, " ? ", n.TrueExpr, " : ", n.FalseExpr)

	case *ast.UnaryExpr:
		pre := n.Op != ast.UoPostdec && n.Op != ast.UoPostinc
		if pre {
			f.format(" " + n.Op.String())
		}
		f.format(n.Expr)
		if !pre {
			f.format(n.Op.String() + " ")
		}

	case *ast.DotExpr:
		f.format(n.Expr, ".", n.Field)

	case *ast.ConstantExpr:
		f.format(n.Value)

	case *ast.CallExpr:
		f.format(n.Callee, "(")
		if len(n.Args) > 0 || !n.VoidPresent {
			for i, a := range n.Args {
				if i > 0 {
					f.format(", ")
				}
				f.format(a)
			}
		} else {
			f.format("void")
		}
		f.format(")")

	case *ast.VarRefExpr:
		if n.Sym == nil {
			f.format("@(nil variable)")
		} else {
			f.format(n.Sym.Name())
		}

	case *ast.TypeConversionExpr:
		f.format(n.RetType)

	case *ast.ParenExpr:
		f.format("(", n.Expr, ")")

	///////////////////////////////// STATEMENTS //////////////////////////////////

	case *ast.DeclarationStmt:
		f.format(n.Decl)

	case *ast.ExpressionStmt:
		f.format(n.Expr, ";")

	case *ast.EmptyStmt:
		f.format(";")

	case *ast.IfStmt:
		f.format("if(", n.IfExpr, ")", formatStatement{true, n.ThenStmt})

		if n.ElseStmt != nil {
			if _, ok := n.ThenStmt.(*ast.CompoundStmt); ok {
				f.format(" ")
			} else {
				f.format("\n" + makeIndent(f.indent))
			}
			f.format("else", formatStatement{true, n.ElseStmt})
		}

	case *ast.SwitchStmt:
		f.format("switch(", n.Expr, ")", formatStatement{true, n.Stmts})

	case *ast.CompoundStmt:
		f.format("{\n")
		f.indent++
		for _, s := range n.Stmts {
			f.format(formatStatement{false, s}, "\n")
		}
		f.indent--
		f.format(makeIndent(f.indent), "}")

	case *ast.CaseStmt:
		f.format("case ", n.Expr, ":")

	case *ast.DefaultStmt:
		f.format("default:")

	case *ast.WhileStmt:
		f.format("while", "(", n.Cond, ")", formatStatement{true, n.Stmt})

	case *ast.DoStmt:
		f.format("do", formatStatement{true, n.Stmt})

		if _, ok := n.Stmt.(*ast.CompoundStmt); ok {
			f.format(" ")
		} else {
			f.format("\n" + makeIndent(f.indent))
		}
		f.format("while(", n.Expr, ");")

	case *ast.ForStmt:
		f.format("for(")
		f.format(n.Init, " ", n.Cond, "; ", n.Loop, ")", formatStatement{true, n.Body})

	case *ast.ReturnStmt:
		f.format("return")
		if n.Expr != nil {
			f.format(" ", n.Expr)
		}
		f.format(";")

	case *ast.ContinueStmt:
		f.format("continue;")

	case *ast.BreakStmt:
		f.format("break;")

	case *ast.DiscardStmt:
		f.format("discard;")

	/////////////////////////////////// TYPES /////////////////////////////////////

	case *ast.BuiltinType:
		if n.Precision != ast.NoneP {
			f.format(n.Precision.String() + " ")
		}
		f.format(n.Type.String())

	case *ast.ArrayType:
		f.format(n.Base, "[")
		if n.Size != nil {
			f.format(n.Size)
		}
		f.format("]")

	case *ast.StructType:
		if !n.StructDef {
			f.format(n.Sym.Name())
		} else {
			f.format(n.Sym)
		}

	case *ast.LayoutQualifier:
		f.format("layout(")
		for i, id := range n.Ids {
			if i > 0 {
				f.format(", ")
			}
			f.format(id.Name)
			if id.Value != nil {
				f.format(" = ", id.Value)
			}
		}
		f.format(") ")

	case *ast.TypeQualifiers:
		if n.Invariant {
			f.format("invariant ")
		}

		if n.Interpolation != ast.IntNone {
			f.format(n.Interpolation.String() + " ")
		}

		if n.Layout != nil {
			f.format(n.Layout)
		}

		if n.Storage != ast.StorNone {
			f.format(n.Storage.String() + " ")
		}

	///////////////////////////////// DECLARATIONS ////////////////////////////////

	case *ast.PrecisionDecl:
		f.format("precision ", n.Type, ";")

	case *ast.FunctionDecl:
		f.format(n.RetType, " ", n.SymName, "(")
		if len(n.Params) > 0 || !n.VoidPresent {
			for i, p := range n.Params {
				if i > 0 {
					f.format(", ")
				}
				f.format(p)
			}
		} else {
			f.format("void")
		}
		f.format(")")
		if n.Stmts == nil {
			f.format(";")
		} else {
			f.format(formatStatement{true, n.Stmts})
		}

	case *ast.MultiVarDecl:
		if n.Quals != nil {
			f.format(n.Quals)
		}
		f.format(n.Type)
		for i, v := range n.Vars {
			if i > 0 || n.FunnyComma {
				f.format(",")
			}
			f.format(" ", v.Name())
			if array, ok := v.SymType.(*ast.ArrayType); ok && array.Base.Equal(n.Type) {
				f.format("[")
				if array.Size != nil {
					f.format(array.Size)
				}
				f.format("]")
			}
			if v.Init != nil {
				f.format(" = ", v.Init)
			}
		}
		f.format(";")

	case *ast.VariableSym:
		f.format(n.SymType, " ", n.Name())
		if n.Init != nil {
			f.format(" = ", n.Init)
		}

	case *ast.StructSym:
		f.format("struct ", n.Name(), " {\n")
		f.indent++
		for _, d := range n.Vars {
			f.format(makeIndent(f.indent), d, "\n")
		}
		f.indent--
		f.format(makeIndent(f.indent), "}")

	case *ast.FuncParameterSym:
		if n.Const {
			f.format("const ")
		}
		if n.Direction != ast.DirNone {
			f.format(n.Direction.String() + " ")
		}
		if t, ok := n.SymType.(*ast.ArrayType); ok && n.Name() != "" && n.ArrayAfterVar {
			f.format(t.Base, " ", n.Name(), "[", t.Size, "]")
		} else {
			f.format(n.SymType)
			if n.SymName != "" {
				f.format(" ", n.SymName)
			}
		}

	case *ast.LayoutDecl:
		f.format(n.Layout, "uniform;")

	case *ast.InvariantDecl:
		f.format("invariant ")
		for i, s := range n.Vars {
			if i > 0 {
				f.format(", ")
			}
			f.format(s.Sym.Name())
		}
		f.format(";")

	case *ast.UniformBlock:
		if n.Layout != nil {
			f.format(n.Layout)
		}
		f.format("uniform ", n.SymName, " {\n")
		f.indent++
		for _, d := range n.Vars {
			f.format(makeIndent(f.indent), d, "\n")
		}
		f.indent--
		f.format(makeIndent(f.indent), "}")

	case *ast.UniformDecl:
		f.format(n.Block)
		if n.SymName != "" {
			f.format(" ", n.SymName)
			if n.Size != nil {
				f.format("[", n.Size, "]")
			}
		}
		f.format(";")

	///////////////////////////////// CONDITIONS //////////////////////////////////

	case *ast.ExpressionCond:
		f.format(n.Expr)

	case *ast.VarDeclCond:
		f.format(n.Sym)

	///////////////////////////////// VALUES //////////////////////////////////////

	case ast.FloatValue:
		str := fmt.Sprintf("%g", float64(n))
		if !strings.ContainsAny(str, ".eE") {
			str += "."
		}
		f.format(str)

	case ast.IntValue:
		f.format(fmt.Sprintf("%d", int64(n)))

	case ast.UintValue:
		f.format(fmt.Sprintf("%du", uint64(n)))

	///////////////////////////////// OTHER ///////////////////////////////////////
	case *ast.Ast:
		for _, d := range n.Decls {
			f.format(d, "\n")
		}

	case fmt.Formatter:
		n.Format(f.f, 'v')

	default:
		f.format(fmt.Sprint(n))
	}
}
