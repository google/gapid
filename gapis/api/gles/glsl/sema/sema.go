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

// Package sema performs semantic checking of OpenGL ES Shading Language programs.
package sema

import (
	"fmt"
	"strings"

	"github.com/google/gapid/gapis/api/gles/glsl/ast"
	"github.com/google/gapid/gapis/api/gles/glsl/parser"
)

func isNumeric(t *ast.BuiltinType) bool { return isFundNumeric(ast.GetFundamentalType(t.Type)) }
func isFundNumeric(t ast.BareType) bool { return t == ast.TInt || t == ast.TUint || t == ast.TFloat }

type semaError struct {
	msg string
}

func (e semaError) Error() string { return e.msg }

// Evaluator is the type of the function used for evaluating constant expressions.
type Evaluator func(expr ast.Expression, resolver func(symbol ast.ValueSymbol) ast.Value, lang ast.Language) (val ast.Value, err []error)

type sema struct {
	symbolValues map[ast.ValueSymbol]ast.Value
	evaluator    Evaluator
}

func makeError(msg string, args ...interface{}) semaError { return semaError{fmt.Sprintf(msg, args...)} }

func (s *sema) visit(n interface{}) {
	ast.VisitChildren(n, s.visit)

	switch n := n.(type) {
	case *ast.VariableSym:
		declType, ok := n.Type().(*ast.ArrayType)
		if ok {
			if _, ok := declType.Base.(*ast.ArrayType); ok {
				panic(makeError("Variable '%v' declared as an array of arrays.",
					parser.Formatter(n)))
			}
			if declType.Size != nil {
				s.visit(declType.Size)
				s.computeArraySize(declType)
			}
		}
		if n.Init != nil {
			if init, ok := n.Init.Type().(*ast.ArrayType); ok && declType != nil &&
				declType.Size == nil {

				declType.ComputedSize = init.ComputedSize
			}
			if !n.Init.Type().Equal(n.Type()) {
				panic(makeError("Variable '%v' initialized with an expression of incompatible type '%v'.",
					parser.Formatter(n), parser.Formatter(n.Init.Type())))
			}
		}
		if n.Quals != nil && n.Quals.Storage == ast.StorConst {
			if n.Init == nil {
				panic(makeError("Constant variable '%v' must be initialized.",
					parser.Formatter(n)))
			}
			if !n.Init.Constant() {
				panic(makeError(
					"Constant variable '%v' initialized with a non-constant expression.",
					parser.Formatter(n)))
			}
			value, err := s.evaluator(n.Init, s.symbolResolver, ast.LangVertexShader)
			if len(err) > 0 {
				panic(makeError("Error evaluating constant expression '%v': %s",
					parser.Formatter(n.Init), err[0]))
			}
			if !value.Type().Equal(n.Type()) {
				panic(fmt.Errorf("Initializer of a constant variable "+
					"'%v' evaluated to an incorrect type.", parser.Formatter(n)))
			}
			s.symbolValues[n] = value
		} else {
			n.SymLValue = true
		}
		return

	case *ast.BinaryExpr:
		s.visitBinaryExpr(n)
	case *ast.UnaryExpr:
		s.visitUnaryExpr(n)
	case *ast.ParenExpr: // do nothing
	case *ast.ConditionalExpr:
		if !n.Cond.Type().Equal(&ast.BuiltinType{Type: ast.TBool}) {
			panic(incompatibleArgError("condition ", n.Cond, "?:"))
		}
		if !n.TrueExpr.Type().Equal(n.FalseExpr.Type()) {
			panic(makeError(
				"Expression arguments of operator '?:' have different types: '%v' vs. '%v'.",
				parser.Formatter(n.TrueExpr), parser.Formatter(n.FalseExpr)))
		}
	case *ast.IndexExpr:
		if !n.Index.Type().Equal(&ast.BuiltinType{Type: ast.TInt}) &&
			!n.Index.Type().Equal(&ast.BuiltinType{Type: ast.TUint}) {

			panic(makeError("Cannot index with an expression of type '%v'.",
				parser.Formatter(n.Index.Type())))
		}
		switch base := n.Base.Type().(type) {
		case *ast.ArrayType:
			n.ExprType = base.Base
			return
		case *ast.BuiltinType:
			bare := base.Type
			col, row := ast.TypeDimensions(bare)
			if col > 1 {
				n.ExprType = &ast.BuiltinType{Type: ast.GetVectorType(ast.TFloat, row)}
				return
			}
			if row > 1 {
				n.ExprType = &ast.BuiltinType{
					Type: ast.GetVectorType(ast.GetFundamentalType(bare), 1),
				}
				return
			}
		}
		panic(makeError("Base argument '%v' (type '%v') is not indexable.",
			parser.Formatter(n.Base), parser.Formatter(n.Base.Type())))
	case *ast.DotExpr:
		s.visitDotExpr(n)
	case *ast.CallExpr:
		s.visitCallExpr(n)
	case *ast.TypeConversionExpr:
		if array, ok := n.RetType.(*ast.ArrayType); ok {
			if array.Size != nil {
				s.visit(array.Size)
				s.computeArraySize(array)
			}
		}
		// XXX support other ast node kinds
	}
}

func incompatibleArgError(argPos string, expr ast.Expression, op string) semaError {
	return makeError("Incompatible %sargument '%v' of type '%v' for operation '%s'.", argPos,
		parser.Formatter(expr), parser.Formatter(expr.Type()), op)
}

func incompatibleArgPairError(e *ast.BinaryExpr) semaError {
	return makeError("Arguments of incompatible types in expression '%v': %v and %v.", parser.Formatter(e),
		parser.Formatter(e.Left.Type()), parser.Formatter(e.Right.Type()))
}

func (s *sema) getTypes(e *ast.BinaryExpr) (ast.BareType, ast.BareType) {
	lt, ok := e.Left.Type().(*ast.BuiltinType)
	if !ok || !isNumeric(lt) {
		panic(incompatibleArgError("left ", e.Left, e.Op.String()))
	}

	rt, ok := e.Right.Type().(*ast.BuiltinType)
	if !ok || !isNumeric(rt) {
		panic(incompatibleArgError("right ", e.Right, e.Op.String()))
	}

	return lt.Type, rt.Type
}

func (s *sema) checkSameFundamentals(e *ast.BinaryExpr, lb, rb ast.BareType) {
	if ast.GetFundamentalType(lb) != ast.GetFundamentalType(rb) {
		panic(makeError("Incompatible types for operation '%s': '%v' and '%v'.", e.Op, lb, rb))
	}
}

func (s *sema) checkIntFundamentals(e *ast.BinaryExpr, lb, rb ast.BareType) {
	if ast.GetFundamentalType(lb) != ast.GetFundamentalType(rb) &&
		(ast.GetFundamentalType(lb) == ast.TInt || ast.GetFundamentalType(lb) == ast.TUint) {

		panic(makeError("Incompatible fundamental types for operation '%s': '%v' and '%v'.",
			e.Op, lb, rb))
	}
}

func (s *sema) setBinaryExprType(e *ast.BinaryExpr, t ast.Type) {
	if !ast.IsAssignmentOp(e.Op) {
		e.ExprType = t
		return
	}
	if !e.Left.Type().Equal(t) {
		panic(makeError("Assignment from an incompatible type: '%v' vs. '%v'.",
			parser.Formatter(e.Left.Type()), parser.Formatter(t)))
	}
	e.ExprType = e.Left.Type()
}

func (s *sema) inheritTypeFromNonScalar(e *ast.BinaryExpr, lb, rb ast.BareType, considerLeft bool) bool {
	_, lrow := ast.TypeDimensions(lb)
	_, rrow := ast.TypeDimensions(rb)
	switch {
	case ast.HasVectorExpansions(rb):
		s.setBinaryExprType(e, e.Left.Type())
		return true
	case ast.HasVectorExpansions(lb) && considerLeft:
		s.setBinaryExprType(e, e.Right.Type())
		return true
	case ast.IsVectorType(lb) && ast.IsVectorType(rb) && lrow == rrow:
		s.setBinaryExprType(e, e.Left.Type())
		return true
	default:
		return false
	}
}

func (s *sema) handleAddSubMulDiv(e *ast.BinaryExpr) {
	lb, rb := s.getTypes(e)
	s.checkSameFundamentals(e, lb, rb)

	if s.inheritTypeFromNonScalar(e, lb, rb, true) {
		return
	}
	if e.Op == ast.BoMul || e.Op == ast.BoMulAssign {
		lcol, lrow := ast.TypeDimensions(lb)
		if ast.IsVectorType(lb) {
			lcol, lrow = lrow, lcol
		}

		rcol, rrow := ast.TypeDimensions(rb)

		if lcol > 0 && rcol > 0 {
			if lcol != rrow {
				panic(makeError("Invalid operand sizes for matrix multiplication: "+
					"%dx%d and %dx%d.", lcol, lrow, rcol, rrow))
			} else {
				s.setBinaryExprType(e, &ast.BuiltinType{Type: ast.GetMatrixType(rcol, lrow)})
			}
			return
		}
	} else {
		if ast.IsMatrixType(lb) && lb == rb {
			e.ExprType = e.Left.Type()
			return
		}
	}
	panic(incompatibleArgPairError(e))
}

func (s *sema) visitBinaryExpr(e *ast.BinaryExpr) {
	if ast.IsAssignmentOp(e.Op) && !e.Left.LValue() {
		panic(makeError("Expression '%v' is not a lvalue.", parser.Formatter(e.Left)))
	}
	switch e.Op {
	case ast.BoAddAssign, ast.BoAdd, ast.BoSubAssign, ast.BoSub, ast.BoMulAssign, ast.BoMul, ast.BoDivAssign, ast.BoDiv:
		s.handleAddSubMulDiv(e)
	case ast.BoMod, ast.BoModAssign:
		lb, rb := s.getTypes(e)
		s.checkIntFundamentals(e, lb, rb)
		if !s.inheritTypeFromNonScalar(e, lb, rb, true) {
			panic(makeError("Operation '%s' cannot operate on vectors of different size: "+
				"'%v' vs. '%v'.", e.Op, lb, rb))
		}
	case ast.BoLess, ast.BoLessEq, ast.BoMore, ast.BoMoreEq:
		lb, rb := s.getTypes(e)
		s.checkSameFundamentals(e, lb, rb)
		if !ast.HasVectorExpansions(lb) {
			panic(makeError("Non-scalar left operand type '%v' for operation '%s'.",
				lb, e.Op))
		}
		if !ast.HasVectorExpansions(rb) {
			panic(makeError("Non-scalar right operand type '%v' for operation '%s'.",
				rb, e.Op))
		}
		s.setBinaryExprType(e, &ast.BuiltinType{Type: ast.TBool})
	case ast.BoEq, ast.BoNotEq:
		if !e.Left.Type().Equal(e.Right.Type()) {
			panic(incompatibleArgPairError(e))
		}
		s.setBinaryExprType(e, &ast.BuiltinType{Type: ast.TBool})
	case ast.BoLand, ast.BoLor, ast.BoLxor:
		boolType := &ast.BuiltinType{Type: ast.TBool}
		if !e.Left.Type().Equal(boolType) {
			panic(incompatibleArgError("non-boolean left ", e.Left, e.Op.String()))
		}
		if !e.Right.Type().Equal(boolType) {
			panic(incompatibleArgError("non-boolean right ", e.Right, e.Op.String()))
		}
		s.setBinaryExprType(e, boolType)
	case ast.BoComma:
		s.setBinaryExprType(e, e.Right.Type())
	case ast.BoShl, ast.BoShlAssign, ast.BoShr, ast.BoShrAssign:
		lb, rb := s.getTypes(e)
		if ast.GetFundamentalType(rb) != ast.TUint {
			panic(makeError("Right operand type of operation '%s' is not unsigned integer, but '%v'.",
				e.Op, parser.Formatter(e.Right.Type())))
		}
		switch ast.GetFundamentalType(lb) {
		case ast.TInt, ast.TUint: // ok
		default:
			panic(makeError("Non-integral left operand of operation '%s': %v.",
				e.Op, parser.Formatter(e.Left.Type())))
		}
		if !s.inheritTypeFromNonScalar(e, lb, rb, false) {
			panic(incompatibleArgPairError(e))
		}
	case ast.BoBand, ast.BoBor, ast.BoBxor:
		lb, rb := s.getTypes(e)
		s.checkIntFundamentals(e, lb, rb)
		if !s.inheritTypeFromNonScalar(e, lb, rb, true) {
			panic(incompatibleArgPairError(e))
		}
	case ast.BoAssign:
		s.setBinaryExprType(e, e.Right.Type())
	}
}

func (s *sema) visitUnaryExpr(e *ast.UnaryExpr) {
	switch e.Op {
	case ast.UoPreinc, ast.UoPredec, ast.UoPostinc, ast.UoPostdec:
		if !e.Expr.LValue() {
			panic(makeError("Expression '%v' is not a lvalue.", parser.Formatter(e.Expr)))
		}
	}
	t, ok := e.Expr.Type().(*ast.BuiltinType)
	if !ok {
		panic(incompatibleArgError("", e.Expr, e.Op.String()))
	}
	bare := t.Type
	fund := ast.GetFundamentalType(bare)
	switch e.Op {
	case ast.UoPreinc, ast.UoPredec, ast.UoPostinc, ast.UoPostdec, ast.UoMinus:
		if fund != ast.TInt && fund != ast.TUint && fund != ast.TFloat {
			panic(incompatibleArgError("", e.Expr, e.Op.String()))
		}
	case ast.UoLnot:
		if bare != ast.TBool {
			panic(incompatibleArgError("", e.Expr, e.Op.String()))
		}
	case ast.UoBnot:
		if fund != ast.TInt && fund != ast.TUint {
			panic(incompatibleArgError("", e.Expr, e.Op.String()))
		}
	}
}

func (s *sema) visitDotExpr(e *ast.DotExpr) {
	switch t := e.Expr.Type().(type) {
	case *ast.ArrayType:
		if e.Field != "length" {
			panic(makeError("Expression '%v' of type '%v' has no member '%s'.",
				parser.Formatter(e.Expr), parser.Formatter(t), e.Field))
		}
		e.ExprType = &ast.FunctionType{RetType: &ast.BuiltinType{Type: ast.TUint}, ArgTypes: nil}
		e.ExprLValue = false
		e.ExprConstant = true
		return
	case *ast.StructType:
		for _, mvd := range t.Sym.Vars {
			for _, sym := range mvd.Vars {
				if sym.Name() == e.Field {
					e.ExprType = sym.Type()
					e.ExprLValue = e.Expr.LValue()
					e.ExprConstant = e.Expr.Constant()
					return
				}
			}
		}
		panic(makeError("Expression '%v' of type '%v' has no member '%s'.",
			parser.Formatter(e.Expr), parser.Formatter(t), e.Field))
	case *ast.BuiltinType:
		if !ast.IsVectorType(t.Type) {
			break
		}
		_, row := ast.TypeDimensions(t.Type)
		kind := ast.ComponentKindNone
		lvalue := e.Expr.LValue()
		for _, r := range e.Field {
			if rpos, rkind := ast.GetVectorComponentInfo(r); rkind != ast.ComponentKindNone {
				if kind != ast.ComponentKindNone && kind != rkind {
					panic(makeError("Swizzle sequence '%s' contains characters from "+
						"multiple coordinate sets.", e.Field))
				}
				kind = rkind
				if rpos >= row {
					panic(makeError("Cannot access component '%s' of a vector of size %d.",
						string(r), row))
				}
				if strings.Count(e.Field, string(r)) >= 2 {
					lvalue = false
				}
			} else {
				panic(makeError("Invalid character '%s' for vector swizzle.", string(r)))
			}
		}
		if len(e.Field) > 4 || len(e.Field) == 0 {
			panic(makeError("Swizzle result must have between 1 and 4 elements."))
		}
		e.ExprType = &ast.BuiltinType{
			Type: ast.GetVectorType(ast.GetFundamentalType(t.Type), uint8(len(e.Field))),
		}
		e.ExprLValue = lvalue
		e.ExprConstant = e.Expr.Constant()
		return
	}
	panic(makeError("Illegal type '%v' for the dot operator.", parser.Formatter(e.Expr.Type())))
}

func componentCount(t ast.BareType) int {
	col, row := ast.TypeDimensions(t)
	return int(col * row)
}

func (s *sema) computeArraySize(t *ast.ArrayType) {
	if !t.Size.Constant() {
		panic(makeError("Array size expression '%v' is not constant.", parser.Formatter(t.Size)))
	}
	sizeType, ok := t.Size.Type().(*ast.BuiltinType)
	if !ok || (sizeType.Type != ast.TInt && sizeType.Type != ast.TUint) {
		panic(makeError("Invalid type '%v' of array size expression '%v'.",
			parser.Formatter(t.Size.Type()), parser.Formatter(t.Size)))
	}
	size, err := s.evaluator(t.Size, s.symbolResolver, ast.LangVertexShader)
	if len(err) > 0 {
		panic(makeError("Error evaluating constant expression '%v': %s",
			parser.Formatter(t.Size), err[0]))
	}
	switch size := size.(type) {
	case ast.UintValue:
		if size == 0 {
			panic(makeError("Cannot construct an array of size zero."))
		}
		t.ComputedSize = size
	case ast.IntValue:
		if size <= 0 {
			panic(makeError("Array size must be positive."))
		}
		t.ComputedSize = ast.UintValue(size)
	default:
		panic(fmt.Errorf("Unexpected value type '%v'.", size.Type()))
	}
}

func (s *sema) visitTypeConversionCall(e *ast.CallExpr) {
	callee := e.Callee.(*ast.TypeConversionExpr)

	e.ExprConstant = true
	for _, arg := range e.Args {
		if !arg.Constant() {
			e.ExprConstant = false
			break
		}
	}

	switch retType := callee.RetType.(type) {
	case *ast.BuiltinType:
		retBare := retType.Type
		if ast.GetFundamentalType(retBare) == ast.TVoid {
			panic(makeError("Cannot construct an object of type '%v'.",
				parser.Formatter(retType)))
		}
		neededComponents := componentCount(retBare)

		for _, arg := range e.Args {
			argType, ok := arg.Type().(*ast.BuiltinType)
			var argBare ast.BareType
			if ok {
				argBare = argType.Type
			}
			if !ok || ast.GetFundamentalType(argBare) == ast.TVoid {
				panic(makeError("Cannot construct an object of type '%v' using "+
					"'%v' (type '%v') as argument.",
					parser.Formatter(retType), parser.Formatter(arg),
					parser.Formatter(argType)))
			}
			if neededComponents <= 0 {
				panic(makeError("Too many components for constructing an object of type '%v'.",
					parser.Formatter(retType)))
			}
			if ast.IsMatrixType(retBare) && ast.IsMatrixType(argBare) {
				if len(e.Args) > 1 {
					panic(makeError("Constructor for a matrix type '%v' can only take "+
						"one matrix argument.", parser.Formatter(retType)))
				}
				neededComponents = 0
				break
			}
			neededComponents -= componentCount(argBare)
		}

		if neededComponents > 0 && (len(e.Args) != 1 ||
			!ast.HasVectorExpansions(e.Args[0].Type().(*ast.BuiltinType).Type)) {

			panic(makeError("Not enough arguments for construction of an object of type '%v'.",
				parser.Formatter(retType)))
		}

	case *ast.StructType:
		arg := 0
		for _, mvd := range retType.Sym.Vars {
			for _, sym := range mvd.Vars {
				if arg >= len(e.Args) {
					panic(makeError("Not enough arguments to construct struct '%s'.",
						retType.Sym.Name()))
				}
				if !sym.Type().Equal(e.Args[arg].Type()) {
					panic(makeError("Cannot initialize '%s.%s' (type '%v') "+
						"with an argument of type '%v'.",
						retType.Sym.Name(), sym.Name(),
						parser.Formatter(sym.Type()),
						parser.Formatter(e.Args[arg].Type())))
				}
				arg++
			}
		}
		if arg < len(e.Args) {
			panic(makeError("Too many arguments for construction of struct '%s'.",
				retType.Sym.Name()))
		}

	case *ast.ArrayType:
		for _, arg := range e.Args {
			if !arg.Type().Equal(retType.Base) {
				panic(makeError("Cannot construct an array of type '%v' from a "+
					"member of type '%v'.", parser.Formatter(retType),
					parser.Formatter(arg.Type())))
			}
		}
		if retType.Size != nil {
			if uint64(retType.ComputedSize) != uint64(len(e.Args)) {
				panic(makeError("Array of size %d constructed with an incorrect "+
					"number of parameters (%d).", retType.ComputedSize, len(e.Args)))
			}
		} else {
			size := len(e.Args)
			if size == 0 {
				panic(makeError("Cannot construct an array of size zero."))
			}
			retType.ComputedSize = ast.UintValue(size)
		}

	}
	callee.ArgTypes = nil
	for _, arg := range e.Args {
		callee.ArgTypes = append(callee.ArgTypes, arg.Type())
	}
	e.ExprType = callee.RetType
}

func (s *sema) visitCallExpr(e *ast.CallExpr) {
	switch callee := e.Callee.(type) {
	case *ast.TypeConversionExpr:
		s.visitTypeConversionCall(e)
	case *ast.DotExpr:
		lengthType := &ast.FunctionType{RetType: &ast.BuiltinType{Type: ast.TUint}}
		if callee.Field != "length" || !callee.ExprType.Equal(lengthType) {
			panic(makeError("Cannot call method '%s' of type '%v'.", callee.Field,
				parser.Formatter(callee.ExprType)))
		}
		e.ExprType = callee.ExprType.(*ast.FunctionType).RetType
		e.ExprConstant = true
	case *ast.VarRefExpr:
		// XXX build overload set, select the correct function, determine constness...
	}
}

func (s *sema) symbolResolver(sym ast.ValueSymbol) ast.Value { return s.symbolValues[sym] }

// Analyze is the main entry point of the package. It performs a semantic check of a GLES Shading
// Language program. Its arguments are the AST representation of the program, and a constant
// expression evaluating function. This function is used for evaluating array size expressions
// and the values of constant variables. An implementation of such a function can be found in the
// evaluator package.
//
// The Analyze function is a work in progress. Currently it just computes the types of all the
// expressions and evaluates all constant expressions.
func Analyze(program interface{}, evaluator Evaluator) (err []error) {
	defer func() {
		if r := recover(); r != nil {
			if sr, ok := r.(semaError); ok {
				err = []error{sr}
			} else {
				panic(r)
			}
		}
	}()

	s := &sema{
		symbolValues: make(map[ast.ValueSymbol]ast.Value),
		evaluator:    evaluator,
	}

	s.visit(program)
	return nil
}
