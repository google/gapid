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

package ast

// ChildVisitor is a callback function used by VisitChildren to visit nodes.
type ChildVisitor func(interface{})

// ChildTransformer is a callback function used by TransformChildren to visit nodes.
type ChildTransformer func(child, parent interface{}) interface{}

// TransformChildren is a helper function which calls the provided callback for
// each child node of the argument, replacing their values with the values
// returned from v.
func TransformChildren(n interface{}, v ChildTransformer) {
	switch n := n.(type) {
	case *BinaryExpr:
		n.Left = v(n.Left, n).(Expression)
		n.Right = v(n.Right, n).(Expression)
	case *IndexExpr:
		n.Base = v(n.Base, n).(Expression)
		n.Index = v(n.Index, n).(Expression)
	case *ConditionalExpr:
		n.Cond = v(n.Cond, n).(Expression)
		n.TrueExpr = v(n.TrueExpr, n).(Expression)
		n.FalseExpr = v(n.FalseExpr, n).(Expression)
	case *UnaryExpr:
		n.Expr = v(n.Expr, n).(Expression)
	case *DotExpr:
		n.Expr = v(n.Expr, n).(Expression)
	case *ConstantExpr:
		n.Value = v(n.Value, n).(Value)
	case *VarRefExpr:
		n.Sym = v(n.Sym, n).(ValueSymbol)
	case *TypeConversionExpr:
		if n.RetType != nil {
			n.RetType = v(n.RetType, n).(Type)
		}
	case *ParenExpr:
		n.Expr = v(n.Expr, n).(Expression)
	case *CallExpr:
		n.Callee = v(n.Callee, n).(Expression)
		for i, c := range n.Args {
			n.Args[i] = v(c, n).(Expression)
		}
	case *DeclarationStmt:
		n.Decl = v(n.Decl, n)
	case *ExpressionStmt:
		n.Expr = v(n.Expr, n).(Expression)
	case *CaseStmt:
		n.Expr = v(n.Expr, n).(Expression)
	case *SwitchStmt:
		n.Expr = v(n.Expr, n).(Expression)
		n.Stmts = v(n.Stmts, n).(*CompoundStmt)
	case *WhileStmt:
		n.Cond = v(n.Cond, n)
		n.Stmt = v(n.Stmt, n)
	case *DoStmt:
		n.Stmt = v(n.Stmt, n)
		n.Expr = v(n.Expr, n).(Expression)
	case *ForStmt:
		n.Init = v(n.Init, n)
		n.Cond = v(n.Cond, n)
		n.Loop = v(n.Loop, n).(Expression)
		n.Body = v(n.Body, n)
	case *ContinueStmt, *BreakStmt, *DiscardStmt, *EmptyStmt, *DefaultStmt:
	case *IfStmt:
		n.IfExpr = v(n.IfExpr, n).(Expression)
		n.ThenStmt = v(n.ThenStmt, n)
		if n.ElseStmt != nil {
			n.ElseStmt = v(n.ElseStmt, n)
		}
	case *CompoundStmt:
		for i, c := range n.Stmts {
			n.Stmts[i] = v(c, n)
		}
	case *ReturnStmt:
		if n.Expr != nil {
			n.Expr = v(n.Expr, n).(Expression)
		}
	case *BuiltinType:
	case *ArrayType:
		if n.Base != nil {
			n.Base = v(n.Base, n).(Type)
		}
		if n.Size != nil {
			n.Size = v(n.Size, n).(Expression)
		}
	case *StructType:
		if n.StructDef {
			n.Sym = v(n.Sym, n).(*StructSym)
		}
	case *LayoutQualifier:
	case *TypeQualifiers:
		if n.Layout != nil {
			n.Layout = v(n.Layout, n).(*LayoutQualifier)
		}
	case *PrecisionDecl:
		if n.Type != nil {
			n.Type = v(n.Type, n).(*BuiltinType)
		}
	case *FunctionDecl:
		if n.RetType != nil {
			n.RetType = v(n.RetType, n).(Type)
		}
		for i, c := range n.Params {
			n.Params[i] = v(c, n).(*FuncParameterSym)
		}
		if n.Stmts != nil {
			n.Stmts = v(n.Stmts, n).(*CompoundStmt)
		}
	case *MultiVarDecl:
		if n.Quals != nil {
			n.Quals = v(n.Quals, n).(*TypeQualifiers)
		}
		if n.Type != nil {
			n.Type = v(n.Type, n).(Type)
		}
		for i, c := range n.Vars {
			n.Vars[i] = v(c, n).(*VariableSym)
		}
	case *LayoutDecl:
		n.Layout = v(n.Layout, n).(*LayoutQualifier)
	case *InvariantDecl:
		for i, c := range n.Vars {
			n.Vars[i] = v(c, n).(*VarRefExpr)
		}
	case *UniformDecl:
		n.Block = v(n.Block, n).(*UniformBlock)
		if n.Size != nil {
			n.Size = v(n.Size, n).(Expression)
		}
	case *VariableSym:
		if n.SymType != nil {
			n.SymType = v(n.SymType, n).(Type)
		}
		if n.Init != nil {
			n.Init = v(n.Init, n).(Expression)
		}
	case *FuncParameterSym:
		if n.SymType != nil {
			n.SymType = v(n.SymType, n).(Type)
		}
	case *StructSym:
		for i, c := range n.Vars {
			n.Vars[i] = v(c, n).(*MultiVarDecl)
		}
	case *UniformBlock:
		if n.Layout != nil {
			n.Layout = v(n.Layout, n).(*LayoutQualifier)
		}
		for i, c := range n.Vars {
			n.Vars[i] = v(c, n).(*MultiVarDecl)
		}
	case *ExpressionCond:
		n.Expr = v(n.Expr, n).(Expression)
	case *VarDeclCond:
		n.Sym = v(n.Sym, n).(*VariableSym)
	case *Ast:
		decls := n.Decls
		for i, c := range decls {
			decls[i] = v(c, n)
		}
	}
}

// VisitChildren is a helper function which calls the provided callback for each child node of
// the argument.
func VisitChildren(n interface{}, v ChildVisitor) {
	TransformChildren(n, func(child, parent interface{}) interface{} {
		v(child)
		return child
	})
}
