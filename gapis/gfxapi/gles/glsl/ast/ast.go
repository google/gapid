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

/*
Package ast defines the abstract syntax tree of OpenGL ES Shading Language programs.

It contains classes for individual types of expressions, statements and declarations. The code
manipulating (parsing, serializing, evaluating, ...) the AST are located in other packages.

The type Ast represents the entire parsed program tree. It consists of a sequence of declarations
(types ***Decl), which can be variable, function, interface declarations, etc.

Function bodies consist of a list of statements (types ***Stmt), for if, while, for statements,
etc. There are two special kinds of statements: DeclarationStmt is a statement holding a
declaration (described above) and ExpressionStmt is a statement holding an expression.

The expressions of the program are represented by types ***Expr. Typical examples are binary
expressions, function call expressions, variable reference expressions, etc.

The AST contains enough information to enable serialization to text while preserving all
whitespace and comments.  The ***Cst members in all AST nodes are used for storing this
whitespace information. This way, whitespace gets automatically preserved during AST
modification. It is not necessary (but is possible) to populate the ***Cst members when creating
new nodes, as the serialization is expected to succeed even without them.

The package also defines the type system of the GLES Shading Language programs (***Type), their
values (***Value) and several helper functions for manipulating them.
*/
package ast

import (
	"github.com/google/gapid/core/text/parse"
)

// binary: ignore

type Language uint8

// Constants identifying various sub-languages.
const (
	LangVertexShader   Language = iota // Vertex Shader language
	LangFragmentShader                 // Fragment Shader language
	LangPreprocessor                   // Language of the #if preprocessor expressions
)

///////////////////////////////// EXPRESSIONS /////////////////////////////////

// The interface for all expressions in the program. It defines methods retrieving the expression
// type, its const-ness, and lvalued-ness. These are to be called only after a semantic analysis
// pass over the program.
type Expression interface {
	LValue() bool
	Type() Type
	Constant() bool
}

// Callee(Arg_1, ... Arg_n).
//
// If the function is called with no arguments the VoidPresent field is set to indicate the
// usage of the f(void) call syntax.
type CallExpr struct {
	Callee Expression
	Args   []Expression

	LParenCst   *parse.Leaf
	CommaCst    []*parse.Leaf
	RParenCst   *parse.Leaf
	VoidPresent bool
	VoidCst     *parse.Leaf

	// Found by semantic analysis.
	ExprType     Type
	ExprConstant bool
}

func (e *CallExpr) LValue() bool   { return false }
func (e *CallExpr) Type() Type     { return e.ExprType }
func (e *CallExpr) Constant() bool { return e.ExprConstant }

// E_Left op E_Right
type BinaryExpr struct {
	Left  Expression
	Op    BinaryOp
	Right Expression

	OpCst *parse.Leaf

	ExprType Type // Found by semantic analysis.
}

func (e *BinaryExpr) Type() Type   { return e.ExprType }
func (e *BinaryExpr) LValue() bool { return false }
func (e *BinaryExpr) Constant() bool {
	// Assignment expressions like '1=2' will be deemed constant, but these expressions will
	// fail semantic checks as LHS must be assignable.
	return e.Left.Constant() && e.Right.Constant() && e.Op != BoComma
}

// Base[Index]
type IndexExpr struct {
	Base  Expression
	Index Expression

	LBracketCst *parse.Leaf
	RBracketCst *parse.Leaf

	ExprType Type // Found by semantic analysis.
}

func (e *IndexExpr) Type() Type     { return e.ExprType }
func (e *IndexExpr) LValue() bool   { return e.Base.LValue() }
func (e *IndexExpr) Constant() bool { return e.Base.Constant() && e.Index.Constant() }

// UnaryExpr represents both prefix and postfix unary operators.
type UnaryExpr struct {
	Op   UnaryOp
	Expr Expression

	OpCst *parse.Leaf
}

func (e *UnaryExpr) Type() Type   { return e.Expr.Type() }
func (e *UnaryExpr) LValue() bool { return false }

func (e *UnaryExpr) Constant() bool {
	// Expressions like '++1' will be deemed constant but fail semantic checks.
	return e.Expr.Constant()
}

// E_c ? E_t : E_f
type ConditionalExpr struct {
	Cond      Expression
	TrueExpr  Expression
	FalseExpr Expression

	QuestionCst *parse.Leaf
	ColonCst    *parse.Leaf
}

func (e *ConditionalExpr) Type() Type   { return e.TrueExpr.Type() }
func (e *ConditionalExpr) LValue() bool { return false }
func (e *ConditionalExpr) Constant() bool {
	return e.Cond.Constant() && e.TrueExpr.Constant() && e.FalseExpr.Constant()
}

// E . field
//
// This expression represents struct member selection (struct.member), vector swizzle (vec4.xyz)
// and array length (array.length) expressions.
type DotExpr struct {
	Expr  Expression
	Field string

	DotCst   *parse.Leaf
	FieldCst *parse.Leaf

	// Found by semantic analysis.
	ExprType     Type
	ExprConstant bool
	ExprLValue   bool
}

func (e *DotExpr) Type() Type     { return e.ExprType }
func (e *DotExpr) LValue() bool   { return e.ExprLValue }
func (e *DotExpr) Constant() bool { return e.ExprConstant }

// ConstantExpr holds integer, floating point and boolean constants.
type ConstantExpr struct {
	Value Value

	ValCst *parse.Leaf
}

func (e *ConstantExpr) Type() Type     { return e.Value.Type() }
func (e *ConstantExpr) LValue() bool   { return false }
func (e *ConstantExpr) Constant() bool { return true }

// VarRefExpr represents a reference to a declared symbol used in an expression.
type VarRefExpr struct {
	Sym ValueSymbol

	SymCst *parse.Leaf
}

func (e *VarRefExpr) Type() Type     { return e.Sym.Type() }
func (e *VarRefExpr) LValue() bool   { return e.Sym.LValue() }
func (e *VarRefExpr) Constant() bool { return e.Sym.Constant() }

// TypeConversionExpr represents a type conversion expression. It is only used as the "callee"
// child of CallExpr. The RetType member is the type specified in the program (the type we are
// converting to). The ArgTypes member is computed by semantic analysis from the arguments. The
// full type of the expression (what Type() returns) is then a FunctionType{RetType, ArgTypes}.
type TypeConversionExpr struct {
	RetType Type

	ArgTypes []Type // Found by semantic analysis.
}

func (e *TypeConversionExpr) LValue() bool   { return false }
func (e *TypeConversionExpr) Constant() bool { return true }
func (e *TypeConversionExpr) Type() Type {
	return &FunctionType{RetType: e.RetType, ArgTypes: e.ArgTypes}
}

// ( E )
type ParenExpr struct {
	Expr Expression

	LParenCst *parse.Leaf
	RParenCst *parse.Leaf
}

func (e *ParenExpr) Type() Type     { return e.Expr.Type() }
func (e *ParenExpr) LValue() bool   { return e.Expr.LValue() }
func (e *ParenExpr) Constant() bool { return e.Expr.Constant() }

///////////////////////////////// STATEMENTS //////////////////////////////////

// ExpressionStmt is a statement holding an expression.
type ExpressionStmt struct {
	Expr Expression

	SemicolonCst *parse.Leaf
}

// DeclarationStmt is a statement holding a declaration.
type DeclarationStmt struct {
	Decl interface{}
}

// ;
type EmptyStmt struct {
	SemicolonCst *parse.Leaf
}

// if(ifExpr) thenStmt else elseStmt
//
// if ElseStms is nil, the statement has no else clause.
type IfStmt struct {
	IfExpr   Expression
	ThenStmt interface{}
	ElseStmt interface{}

	IfCst     *parse.Leaf
	LParenCst *parse.Leaf
	RParenCst *parse.Leaf
	ElseCst   *parse.Leaf
}

// switch(expr) stmts
type SwitchStmt struct {
	Expr  Expression
	Stmts *CompoundStmt

	SwitchCst *parse.Leaf
	LParenCst *parse.Leaf
	RParenCst *parse.Leaf
}

// { stmts... }
type CompoundStmt struct {
	Stmts []interface{}

	LBraceCst *parse.Leaf
	RBraceCst *parse.Leaf
}

// case expr:
type CaseStmt struct {
	Expr Expression

	CaseCst  *parse.Leaf
	ColonCst *parse.Leaf
}

// default:
type DefaultStmt struct {
	DefaultCst *parse.Leaf
	ColonCst   *parse.Leaf
}

// while(cond) stmt
type WhileStmt struct {
	Cond interface{}
	Stmt interface{}

	WhileCst  *parse.Leaf
	LParenCst *parse.Leaf
	RParenCst *parse.Leaf
}

// do stmt while(expr);
type DoStmt struct {
	Stmt interface{}
	Expr Expression

	DoCst        *parse.Leaf
	LParenCst    *parse.Leaf
	RParenCst    *parse.Leaf
	WhileCst     *parse.Leaf
	SemicolonCst *parse.Leaf
}

// for(init; cond; loop) body
//
// The second member Cond is expected to hold an expression (ExpressionCond) or a declaration of
// a single variable (VarDeclCond).
type ForStmt struct {
	Init interface{} // statement
	Cond interface{} // condition
	Loop Expression
	Body interface{} // statement

	ForCst        *parse.Leaf
	LParenCst     *parse.Leaf
	Semicolon2Cst *parse.Leaf // semicolon1 is a part of Init
	RParenCst     *parse.Leaf
}

// return expr;
type ReturnStmt struct {
	Expr Expression

	ReturnCst    *parse.Leaf
	SemicolonCst *parse.Leaf
}

// continue;
type ContinueStmt struct {
	ContinueCst  *parse.Leaf
	SemicolonCst *parse.Leaf
}

// break;
type BreakStmt struct {
	BreakCst     *parse.Leaf
	SemicolonCst *parse.Leaf
}

// discard;
type DiscardStmt struct {
	DiscardCst   *parse.Leaf
	SemicolonCst *parse.Leaf
}

///////////////////////////////// DECLARATIONS ////////////////////////////////

// Function is an interface for function declarations. In addition to ValueSymbol methods, it
// provides methods to retrieve function return type and function parameters.
type Function interface {
	ValueSymbol
	ReturnType() Type
	Parameters() []*FuncParameterSym
}

func makeFunctionType(ret Type, args []*FuncParameterSym) Type {
	fun := &FunctionType{RetType: ret}
	for _, p := range args {
		fun.ArgTypes = append(fun.ArgTypes, p.Type())
	}
	return fun
}

// FunctionDecl is a function declaration in a GLSL program. If field Stsms is not nil, then it
// represents a function
// definition. FunctionDecl implements ValueSymbol and Function.
//
// If the function is declared with no arguments the VoidPresent field is set to indicate the
// usage of the f(void) call syntax.
type FunctionDecl struct {
	RetType Type
	SymName string
	Params  []*FuncParameterSym
	Stmts   *CompoundStmt

	NameCst      *parse.Leaf
	LParenCst    *parse.Leaf
	CommaCst     []*parse.Leaf
	RParenCst    *parse.Leaf
	SemicolonCst *parse.Leaf
	VoidPresent  bool
	VoidCst      *parse.Leaf
}

func (f *FunctionDecl) Name() string   { return f.SymName }
func (f *FunctionDecl) Type() Type     { return makeFunctionType(f.RetType, f.Params) }
func (f *FunctionDecl) LValue() bool   { return false }
func (f *FunctionDecl) Constant() bool { return false }

func (f *FunctionDecl) ReturnType() Type                { return f.RetType }
func (f *FunctionDecl) Parameters() []*FuncParameterSym { return f.Params }

// BuiltinFunction is the definition of a GLSL builtin function. It implements the Function
// interface. The definition of the function is provided as an opaque go function.
type BuiltinFunction struct {
	RetType Type
	SymName string
	Params  []*FuncParameterSym
	Impl    func(v []Value) Value
}

func (f *BuiltinFunction) Name() string   { return f.SymName }
func (f *BuiltinFunction) Type() Type     { return makeFunctionType(f.RetType, f.Params) }
func (f *BuiltinFunction) LValue() bool   { return false }
func (f *BuiltinFunction) Constant() bool { return true }

func (f *BuiltinFunction) ReturnType() Type                { return f.RetType }
func (f *BuiltinFunction) Parameters() []*FuncParameterSym { return f.Params }

// precision lowp int;
type PrecisionDecl struct {
	Type *BuiltinType

	PrecisionCst *parse.Leaf
	SemicolonCst *parse.Leaf
}

// MultiVarDecl is a declaration of a (possibly more than one) variable. If no qualifiers are
// present, the Quals field can be null.
type MultiVarDecl struct {
	Vars  []*VariableSym
	Quals *TypeQualifiers
	Type  Type

	FunnyComma    bool // for declarations like: `int ,a;`
	FunnyCommaCst *parse.Leaf
	CommaCst      []*parse.Leaf
	SemicolonCst  *parse.Leaf
}

// layout(...) uniform;
type LayoutDecl struct {
	Layout *LayoutQualifier

	UniformCst   *parse.Leaf
	SemicolonCst *parse.Leaf
}

// invariant foo, bar, baz;
type InvariantDecl struct {
	Vars []*VarRefExpr

	InvariantCst *parse.Leaf
	CommaCst     []*parse.Leaf
	SemicolonCst *parse.Leaf
}

// UniformDecl represents a whole uniform interface declaration.
//   layout(...) uniform BlockName { int Var; ... } InstanceName;
//   ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
//                  UniformBlock
//   ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
//                          UniformDecl
// Implements Symbol with Name = InstanceName.
type UniformDecl struct {
	Block   *UniformBlock
	SymName string
	Size    Expression

	NameCst      *parse.Leaf
	LBracketCst  *parse.Leaf
	RBracketCst  *parse.Leaf
	SemicolonCst *parse.Leaf
}

func (u *UniformDecl) Name() string { return u.SymName }

///////////////////////////////// SYMBOLS ////////////////////////////////

// Symbol is an interface for all declared symbols (anything that has a name and is scoped) in
// the language. It does not always coincide with the "declaration" concept. E.g.: 'int a,b;' is a
// single declaration, but declares two symbols; the declaration 'struct A { ... } a;' declares
// one symbol for the struct and another for the variable, etc.
type Symbol interface {
	Name() string
}

// ValueSymbol represents a Symbol (a named object in the language), which can be used in a value
// context in the language (basically, referenced in an expression through VarRefExpr). After a
// semantic analysis pass the three functions provide information about the nature of the symbol.
type ValueSymbol interface {
	Symbol
	Type() Type
	LValue() bool
	Constant() bool
}

// VariableSym is a declaration of a single variable, possibly with an initialization expression.
// It is normally a part of a MultiVarDecl (for normal declaration) or a VarDeclCond (for
// declarations which are a part of while and for statements).
type VariableSym struct {
	SymType Type
	SymName string
	Init    Expression

	Quals *TypeQualifiers

	NameCst  *parse.Leaf
	EqualCst *parse.Leaf

	SymLValue bool // Found by semantic analysis.
}

func (v *VariableSym) Name() string { return v.SymName }
func (v *VariableSym) Type() Type   { return v.SymType }

func (v *VariableSym) LValue() bool   { return v.SymLValue }
func (v *VariableSym) Constant() bool { return !v.LValue() }

// The direction of a parameter in a function declaration.
type ParamDirection uint8

const (
	DirNone ParamDirection = iota
	DirIn
	DirOut
	DirInout
)

func (d ParamDirection) String() string {
	switch d {
	case DirIn:
		return "in"
	case DirOut:
		return "out"
	case DirInout:
		return "inout"
	default:
		return ""
	}
}

// FuncParameter is a formal function parameter in a function declaration.
type FuncParameterSym struct {
	Const     bool
	Direction ParamDirection
	SymType   Type
	SymName   string

	ConstCst      *parse.Leaf
	DirectionCst  *parse.Leaf
	NameCst       *parse.Leaf
	ArrayAfterVar bool // Whether the variable was declared as int foo[].
}

func (v *FuncParameterSym) Name() string   { return v.SymName }
func (v *FuncParameterSym) Type() Type     { return v.SymType }
func (v *FuncParameterSym) LValue() bool   { return !v.Const }
func (v *FuncParameterSym) Constant() bool { return false }

// StructSym is the struct declaration. It has a name and a list of variable declarations.
type StructSym struct {
	SymName string
	Vars    []*MultiVarDecl

	StructCst *parse.Leaf
	NameCst   *parse.Leaf
	LBraceCst *parse.Leaf
	RBraceCst *parse.Leaf
}

func (s *StructSym) Name() string { return s.SymName }

// UniformBlock represents the "block" part of the uniform interface declarations.
//   layout(...) uniform BlockName { int Var; ... } InstanceName;
//   ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
//                  UniformBlock
//   ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
//                          UniformDecl
// Implements Symbol with Name = BlockName.
type UniformBlock struct {
	Layout  *LayoutQualifier
	SymName string
	Vars    []*MultiVarDecl

	UniformCst *parse.Leaf
	NameCst    *parse.Leaf
	LBraceCst  *parse.Leaf
	RBraceCst  *parse.Leaf
}

func (u *UniformBlock) Name() string { return u.SymName }

///////////////////////////////// CONDITIONS //////////////////////////////////

// ExpressionCond is a condition holding an expression. Conditions are used in ForStmt and
// WhileStmt.
type ExpressionCond struct {
	Expr Expression
}

// VarDeclCond is a condition holding a declaration of a single variable. Conditions are used in
// ForStmt and WhileStmt.
type VarDeclCond struct {
	Sym *VariableSym
}

// Ast represents an abstract syntax tree of a parsed program. It is just a holder for a list of
// declarations.
type Ast struct {
	Decls []interface{}
}
