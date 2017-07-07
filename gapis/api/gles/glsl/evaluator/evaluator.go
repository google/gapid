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

// Package evaluator is responsible for evaluating expressions in OpenGL ES Shading Language programs.
package evaluator

import (
	"fmt"

	"github.com/google/gapid/gapis/api/gles/glsl/ast"
	"github.com/google/gapid/gapis/api/gles/glsl/parser"
)

type evalError string

func (e evalError) Error() string { return string(e) }

func makeError(msg string, args ...interface{}) evalError { return evalError(fmt.Sprintf(msg, args...)) }

// Given a scalar value v and a vector or matrix type t, construct a new value of type t by
// by replicating v. It is assumed types of v and t have equal fundamental types.
func expandScalar(v ast.Value, t ast.BareType) ast.Value {
	col, row := ast.TypeDimensions(t)
	if col > 1 {
		return ast.NewMatrixValue(t, func(uint8, uint8) ast.Value { return v })
	}
	if row > 1 {
		return ast.NewVectorValue(t, func(uint8) ast.Value { return v })
	}
	panic(fmt.Errorf("Unexpected type '%s'.", t))
}

func getType(v ast.Value) ast.BareType { return v.Type().(*ast.BuiltinType).Type.Canonicalize() }

// If one of the values is scalar, this function expands it to match the type of the other value.
// Both values must have the same fundamental types.
func expandScalars(left ast.Value, right ast.Value) (ast.Value, ast.Value) {
	lt := getType(left)
	rt := getType(right)

	if ast.HasVectorExpansions(lt) && !ast.HasVectorExpansions(rt) {
		left = expandScalar(left, rt)
	} else if ast.HasVectorExpansions(rt) && !ast.HasVectorExpansions(lt) {
		right = expandScalar(right, lt)
	}
	return left, right
}

// Performs the correct type of scalar addition, depending on the value type (float, int, uint).
// Both values must have the same type.
func scalarAdd(left, right ast.Value) ast.Value {
	switch left := left.(type) {
	case ast.IntValue:
		return left + right.(ast.IntValue)
	case ast.UintValue:
		return left + right.(ast.UintValue)
	case ast.FloatValue:
		return left + right.(ast.FloatValue)
	}
	panic(fmt.Errorf("Unknown type '%s'.", getType(left)))
}

// Performs the correct type of scalar binary and, depending on the value type (int, uint).
// Both values must have the same type.
func scalarBand(left, right ast.Value) ast.Value {
	switch left := left.(type) {
	case ast.IntValue:
		return left & right.(ast.IntValue)
	case ast.UintValue:
		return left & right.(ast.UintValue)
	}
	panic(fmt.Errorf("Unknown type '%s'.", getType(left)))
}

// Performs the correct type of scalar binary or, depending on the value type (int, uint).
// Both values must have the same type.
func scalarBor(left, right ast.Value) ast.Value {
	switch left := left.(type) {
	case ast.IntValue:
		return left | right.(ast.IntValue)
	case ast.UintValue:
		return left | right.(ast.UintValue)
	}
	panic(fmt.Errorf("Unknown type '%s'.", getType(left)))
}

// Performs the correct type of scalar binary xor, depending on the value type (int, uint).
// Both values must have the same type.
func scalarBxor(left, right ast.Value) ast.Value {
	switch left := left.(type) {
	case ast.IntValue:
		return left ^ right.(ast.IntValue)
	case ast.UintValue:
		return left ^ right.(ast.UintValue)
	}
	panic(fmt.Errorf("Unknown type '%s'.", getType(left)))
}

// Performs the correct type of scalar division, depending on the value type (float, int, uint).
// Both values must have the same type.
func scalarDiv(left, right ast.Value) ast.Value {
	switch left := left.(type) {
	case ast.IntValue:
		return left / right.(ast.IntValue)
	case ast.UintValue:
		return left / right.(ast.UintValue)
	case ast.FloatValue:
		return left / right.(ast.FloatValue)
	}
	panic(fmt.Errorf("Unknown type '%s'.", getType(left)))
}

// Performs the correct type of scalar modulo, depending on the value type (int, uint).
// Both values must have the same type.
func scalarMod(left, right ast.Value) ast.Value {
	switch left := left.(type) {
	case ast.IntValue:
		return left % right.(ast.IntValue)
	case ast.UintValue:
		return left % right.(ast.UintValue)
	}
	panic(fmt.Errorf("Unknown type '%s'.", getType(left)))
}

// Performs the correct type of scalar multiplication, depending on the value type (float, int, uint).
// Both values must have the same type.
func scalarMul(left, right ast.Value) ast.Value {
	switch left := left.(type) {
	case ast.IntValue:
		return left * right.(ast.IntValue)
	case ast.UintValue:
		return left * right.(ast.UintValue)
	case ast.FloatValue:
		return left * right.(ast.FloatValue)
	}
	panic(fmt.Errorf("Unknown type '%s'.", getType(left)))
}

// Performs the correct type of scalar shift left, depending on the value type (int, uint).
// The type of the second value must be uint.
func scalarShl(left, right ast.Value) ast.Value {
	switch left := left.(type) {
	case ast.IntValue:
		return left << right.(ast.UintValue)
	case ast.UintValue:
		return left << right.(ast.UintValue)
	}
	panic(fmt.Errorf("Unknown type '%s'.", getType(left)))
}

// Performs the correct type of scalar shift right, depending on the value type (int, uint).
// The type of the second value must be uint.
func scalarShr(left, right ast.Value) ast.Value {
	switch left := left.(type) {
	case ast.IntValue:
		return left >> right.(ast.UintValue)
	case ast.UintValue:
		return left >> right.(ast.UintValue)
	}
	panic(fmt.Errorf("Unknown type '%s'.", getType(left)))
}

// Performs the correct type of scalar comparison, depending on the value type (float, int, uint).
// Both values must have the same type.
func (v *evaluator) scalarLess(left, right ast.Value) ast.BoolValue {
	left = v.intFixup(left)
	right = v.intFixup(right)
	switch left := left.(type) {
	case ast.IntValue:
		return ast.BoolValue(left < right.(ast.IntValue))
	case ast.UintValue:
		return ast.BoolValue(left < right.(ast.UintValue))
	case ast.FloatValue:
		return ast.BoolValue(left < right.(ast.FloatValue))
	}
	panic(fmt.Errorf("Unknown type '%s'.", getType(left)))
}

// Performs the correct type of scalar subtraction, depending on the value type (float, int, uint).
// Both values must have the same type.
func scalarSub(left, right ast.Value) ast.Value {
	switch left := left.(type) {
	case ast.IntValue:
		return left - right.(ast.IntValue)
	case ast.UintValue:
		return left - right.(ast.UintValue)
	case ast.FloatValue:
		return left - right.(ast.FloatValue)
	}
	panic(fmt.Errorf("Unknown type '%s'.", getType(left)))
}

// Performs the correct type of scalar binary negation, depending on the value type (int, uint).
func scalarBnot(v ast.Value) ast.Value {
	switch v := v.(type) {
	case ast.IntValue:
		return ^v
	case ast.UintValue:
		return ^v
	}
	panic(fmt.Errorf("Unknown type '%s'.", getType(v)))
}

// Performs the correct type of scalar arithmetic negation, depending on the value type (float, int, uint).
func scalarMinus(v ast.Value) ast.Value {
	switch v := v.(type) {
	case ast.IntValue:
		return -v
	case ast.UintValue:
		return -v
	case ast.FloatValue:
		return -v
	}
	panic(fmt.Errorf("Unknown type '%s'.", getType(v)))
}

// Performs the binary scalar operation given by the functor argument on the two values provided.
// In case the operands are vectors or matrices, the scalar operation is applied for each
// component separately. If one of the operands is scalar, it is expanded to the size of the
// other operand.
func (v *evaluator) evalBinaryPiecewiseExpr(left, right ast.Value, scalarOp func(ast.Value, ast.Value) ast.Value) ast.Value {
	left, right = expandScalars(left, right)

	t := getType(left)
	col, row := ast.TypeDimensions(t)
	if col > 1 {
		left, right := left.(ast.MatrixValue), right.(ast.MatrixValue)
		return ast.NewMatrixValue(t, func(i, j uint8) ast.Value {
			return scalarOp(left.Members[i][j], right.Members[i][j])
		})
	}
	if row > 1 {
		left, right := left.(ast.VectorValue), right.(ast.VectorValue)
		return ast.NewVectorValue(t,
			func(i uint8) ast.Value { return scalarOp(left.Members[i], right.Members[i]) })
	}
	return scalarOp(v.intFixup(left), v.intFixup(right))
}

// Perform a multiplication operation.
func (v *evaluator) evalMul(left, right ast.Value) ast.Value {
	lt, rt := getType(left), getType(right)

	// These cases are handled the same as other binary operators.
	if ast.HasVectorExpansions(lt) || ast.HasVectorExpansions(rt) ||
		(ast.IsVectorType(lt) && ast.IsVectorType(rt)) {

		return v.evalBinaryPiecewiseExpr(left, right, scalarMul)
	}

	// Here we do matrix multiplication.

	// Convert the left operand to a matrix.
	var lmat ast.MatrixValue
	if v, ok := left.(ast.VectorValue); ok {
		lmat = ast.NewMatrixValueCR(uint8(len(v.Members)), 1,
			func(i, j uint8) ast.Value { return v.Members[i] })
	} else {
		lmat = left.(ast.MatrixValue)
	}

	// Convert the right operand to a matrix.
	var rmat ast.MatrixValue
	if v, ok := right.(ast.VectorValue); ok {
		rmat.Members = [][]ast.Value{v.Members}
	} else {
		rmat = right.(ast.MatrixValue)
	}

	if len(lmat.Members) != len(rmat.Members[0]) {
		panic(fmt.Errorf("Incompatible matrix dimensions (%dx%d) vs. (%dx%d).", len(lmat.Members),
			len(lmat.Members[0]), len(rmat.Members), len(rmat.Members[0])))
	}

	// Do the actual multiplication.
	mat := ast.NewMatrixValueCR(uint8(len(rmat.Members)), uint8(len(lmat.Members[0])),
		func(i, j uint8) ast.Value {
			acc := ast.FloatValue(0.0)
			for k := range lmat.Members {
				acc += lmat.Members[k][j].(ast.FloatValue) * rmat.Members[i][k].(ast.FloatValue)
			}
			return acc
		})

	// Return a matrix with one column as a vector.
	if len(mat.Members) == 1 {
		return ast.VectorValue{
			Members: mat.Members[0],
			ValType: ast.BuiltinType{Type: ast.GetVectorType(ast.TFloat, uint8(len(mat.Members[0])))},
		}
	}

	// Return a 1x1 matrix as a scalar.
	if len(mat.Members[0]) == 1 {
		return ast.NewVectorValue(ast.GetVectorType(ast.TFloat, uint8(len(mat.Members))),
			func(i uint8) ast.Value { return mat.Members[i][0] })
	}
	return mat
}

type evaluator struct {
	resolver func(sym ast.ValueSymbol) ast.Value
	language ast.Language
}

// If evaluating preprocessor expressions, values are coerced to a boolean in boolean contexts.
// In other cases, the values should already have the correct type.
func (v *evaluator) boolFixup(val ast.Value) ast.Value {
	if v.language == ast.LangPreprocessor {
		return convertBool(val)
	}
	return val
}

// If evaluating preprocessor expressions, values are coerced to an integer in int contexts.
// In other cases, the values should already have the correct type.
func (v *evaluator) intFixup(val ast.Value) ast.Value {
	if v.language == ast.LangPreprocessor {
		return convertInt(val)
	}
	return val
}

// Evaluate a binary expression.
func (v *evaluator) evalBinaryExpr(e *ast.BinaryExpr) ast.Value {
	// TODO: Assignment operators ???
	left := v.eval(e.Left)

	switch e.Op {
	case ast.BoLand:
		if !v.boolFixup(left).(ast.BoolValue) {
			return left
		}
	case ast.BoLor:
		if v.boolFixup(left).(ast.BoolValue) {
			return left
		}
	}

	right := v.eval(e.Right)

	switch e.Op {
	case ast.BoAdd:
		return v.evalBinaryPiecewiseExpr(left, right, scalarAdd)
	case ast.BoBand:
		return v.evalBinaryPiecewiseExpr(left, right, scalarBand)
	case ast.BoBor:
		return v.evalBinaryPiecewiseExpr(left, right, scalarBor)
	case ast.BoBxor:
		return v.evalBinaryPiecewiseExpr(left, right, scalarBxor)
	case ast.BoDiv:
		return v.evalBinaryPiecewiseExpr(left, right, scalarDiv)
	case ast.BoMod:
		return v.evalBinaryPiecewiseExpr(left, right, scalarMod)
	case ast.BoShl:
		return v.evalBinaryPiecewiseExpr(left, right, scalarShl)
	case ast.BoShr:
		return v.evalBinaryPiecewiseExpr(left, right, scalarShr)
	case ast.BoSub:
		return v.evalBinaryPiecewiseExpr(left, right, scalarSub)
	case ast.BoMul:
		return v.evalMul(left, right)

	case ast.BoLand:
		return right
	case ast.BoLor:
		return right
	case ast.BoLxor:
		return ast.BoolValue(v.boolFixup(left).(ast.BoolValue) != v.boolFixup(right).(ast.BoolValue))

	case ast.BoEq:
		return ast.BoolValue(ast.ValueEquals(left, right))
	case ast.BoNotEq:
		return ast.BoolValue(!ast.ValueEquals(left, right))
	case ast.BoLess:
		return v.scalarLess(left, right)
	case ast.BoLessEq:
		return !v.scalarLess(right, left)
	case ast.BoMore:
		return v.scalarLess(right, left)
	case ast.BoMoreEq:
		return !v.scalarLess(left, right)

	case ast.BoComma:
		return right
	}
	return nil
}

// Performs the unary scalar operation given by the functor argument on the provided value.
func (v *evaluator) evalUnaryPiecewiseExpr(val ast.Value, scalarOp func(ast.Value) ast.Value) ast.Value {
	t := getType(val)
	col, row := ast.TypeDimensions(t)
	if col > 1 {
		val := val.(ast.MatrixValue)
		return ast.NewMatrixValue(t, func(i, j uint8) ast.Value { return scalarOp(val.Members[i][j]) })
	}
	if row > 1 {
		val := val.(ast.VectorValue)
		return ast.NewVectorValue(t, func(i uint8) ast.Value { return scalarOp(val.Members[i]) })
	}
	return scalarOp(v.intFixup(val))
}

// Checks that the value is within the range 0..size-1. Returns the value as a uint64.
func checkIndexRange(v ast.Value, size uint64) (index uint64, ok bool) {
	switch v := v.(type) {
	case ast.IntValue:
		if uint64(v) >= size || v < 0 {
			return 0, false
		}
		return uint64(v), true
	case ast.UintValue:
		if uint64(v) >= size {
			return 0, false
		}
		return uint64(v), true
	}
	panic(fmt.Errorf("Unexpected type '%v' as index.", parser.Formatter(v.Type())))
}

// Helper type for doing type conversions.
type convertBuiltin ast.BareType

// Converts a matrix in the argument to the type given by c, according to the GLSL rules. The
// slice v should contain a single value.
func (c convertBuiltin) matrixToMatrix(v []ast.Value) ast.Value {
	members := v[0].(ast.MatrixValue).Members
	return ast.NewMatrixValue(ast.BareType(c), func(i, j uint8) ast.Value {
		switch {
		case int(i) < len(members) && int(j) < len(members[i]):
			return members[i][j]
		case i == j:
			return ast.FloatValue(1.0)
		default:
			return ast.FloatValue(0.0)
		}
	})
}

// convert a value to a boolean
func convertBool(v ast.Value) ast.Value {
	switch v := v.(type) {
	case ast.BoolValue:
		return v
	case ast.UintValue:
		return ast.BoolValue(v != 0)
	case ast.IntValue:
		return ast.BoolValue(v != 0)
	case ast.FloatValue:
		return ast.BoolValue(v != 0.0)
	}
	panic(fmt.Errorf("Unknown type '%v'.", v.Type()))
}

// convert a value to an unsigned integer
func convertUint(v ast.Value) ast.Value {
	switch v := v.(type) {
	case ast.BoolValue:
		if v {
			return ast.UintValue(1)
		}
		return ast.UintValue(0)
	case ast.UintValue:
		return v
	case ast.IntValue:
		return ast.UintValue(v)
	case ast.FloatValue:
		return ast.UintValue(v)
	}
	panic(fmt.Errorf("Unknown type '%v'.", v.Type()))
}

// convert a value to a signed integer
func convertInt(v ast.Value) ast.Value {
	switch v := v.(type) {
	case ast.BoolValue:
		if v {
			return ast.IntValue(1)
		}
		return ast.IntValue(0)
	case ast.UintValue:
		return ast.IntValue(v)
	case ast.IntValue:
		return v
	case ast.FloatValue:
		return ast.IntValue(v)
	}
	panic(fmt.Errorf("Unknown type '%v'.", v.Type()))
}

// convert a value to a float
func convertFloat(v ast.Value) ast.Value {
	switch v := v.(type) {
	case ast.BoolValue:
		if v {
			return ast.FloatValue(1.0)
		}
		return ast.FloatValue(0.0)
	case ast.UintValue:
		return ast.FloatValue(v)
	case ast.IntValue:
		return ast.FloatValue(v)
	case ast.FloatValue:
		return v
	}
	panic(fmt.Errorf("Unknown type '%v'.", v.Type()))
}

var convertMap = map[ast.BareType]func(ast.Value) ast.Value{
	ast.TBool:  convertBool,
	ast.TUint:  convertUint,
	ast.TInt:   convertInt,
	ast.TFloat: convertFloat,
}

// Construct a value of type c from a single scalar present in the slice v, as per GLSL spec.
func (c convertBuiltin) fromScalar(v []ast.Value) ast.Value {
	t := ast.BareType(c)
	value := convertMap[ast.GetFundamentalType(t)](v[0])

	col, row := ast.TypeDimensions(t)
	switch {
	case col > 1:
		return ast.NewMatrixValue(t,
			func(i, j uint8) ast.Value {
				if i == j {
					return value
				} else {
					return ast.FloatValue(0.0)
				}
			})
	case row > 1:
		return ast.NewVectorValue(t, func(uint8) ast.Value { return value })
	default:
		return value
	}
}

// The general conversion function of GLSL. Constructs a sequence of components from the given
// slice and uses these components to initialize a new value of type c. There should always be
// enough components in v to fully initialize the value.
func (c convertBuiltin) generalConvert(v []ast.Value) ast.Value {
	t := ast.BareType(c)
	convertFun := convertMap[ast.GetFundamentalType(t)]

	var components []ast.Value
	for _, v := range v {
		switch v := v.(type) {
		case ast.MatrixValue:
			for _, v := range v.Members {
				components = append(components, v...)
			}
		case ast.VectorValue:
			components = append(components, v.Members...)
		default:
			components = append(components, v)
		}
	}

	i := 0
	col, row := ast.TypeDimensions(t)
	switch {
	case col > 1:
		return ast.NewMatrixValue(t,
			func(uint8, uint8) ast.Value { i++; return convertFun(components[i-1]) })
	case row > 1:
		return ast.NewVectorValue(t,
			func(uint8) ast.Value { i++; return convertFun(components[i-1]) })
	default:
		return convertFun(components[0])
	}
}

type makeStruct ast.StructType

// Constructs a new struct using the values in the slice v. The input values should have the
// correct types.
func (t *makeStruct) construct(v []ast.Value) ast.Value {
	i := 0
	return ast.NewStructValue((*ast.StructType)(t), func(string) ast.Value { i++; return v[i-1] })
}

// Returns a function value which will perform the correct type conversion on its arguments when
// invoked.
func (v *evaluator) evalTypeConversion(e *ast.TypeConversionExpr) ast.Value {
	returnFunc := func(f func(v []ast.Value) ast.Value) ast.FunctionValue {
		return ast.FunctionValue{
			Func:    f,
			ValType: e.Type().(*ast.FunctionType),
		}
	}

	switch retType := e.RetType.(type) {
	case *ast.BuiltinType:
		if len(e.ArgTypes) == 1 {
			argBare := e.ArgTypes[0].(*ast.BuiltinType).Type

			switch {
			case ast.HasVectorExpansions(argBare):
				return returnFunc(convertBuiltin(retType.Type).fromScalar)

			case ast.IsMatrixType(retType.Type) &&
				ast.IsMatrixType(e.ArgTypes[0].(*ast.BuiltinType).Type):

				return returnFunc(convertBuiltin(retType.Type).matrixToMatrix)
			}
		}

		return returnFunc(convertBuiltin(retType.Type).generalConvert)

	case *ast.StructType:
		return returnFunc((*makeStruct)(retType).construct)

	case *ast.ArrayType:
		return returnFunc(func(v []ast.Value) ast.Value { return ast.ArrayValue(v) })

	default:
		panic(fmt.Errorf("Unknown type: %T", e.RetType))
	}
}

// Evaluates an index expression base[index], for all supported types of base (matrix, vector,
// array). Returns an error if the index is out of range.
func (v *evaluator) evalIndexExpr(e *ast.IndexExpr) ast.Value {
	base := v.eval(e.Base)
	index := v.eval(e.Index)
	switch base := base.(type) {
	case ast.MatrixValue:
		if index, ok := checkIndexRange(index, uint64(len(base.Members))); ok {
			return ast.VectorValue{
				Members: base.Members[index],
				ValType: ast.BuiltinType{
					Type: ast.GetVectorType(ast.TFloat, uint8(len(base.Members[index]))),
				},
			}
		}

	case ast.VectorValue:
		if index, ok := checkIndexRange(index, uint64(len(base.Members))); ok {
			return base.Members[uint64(index)]
		}

	case ast.ArrayValue:
		if index, ok := checkIndexRange(index, uint64(len(base))); ok {
			return base[uint64(index)]
		}
	}
	panic(makeError("Index %d is out of range for a value of type '%v'.", index,
		parser.Formatter(base.Type())))
}

// Computes the value of an expression by dispatching based on the expression type.
func (v *evaluator) eval(e ast.Expression) ast.Value {
	switch e := e.(type) {
	case *ast.BinaryExpr:
		return v.evalBinaryExpr(e)
	case *ast.IndexExpr:
		return v.evalIndexExpr(e)

	case *ast.UnaryExpr:
		expr := v.eval(e.Expr)
		// TODO post/pre-inc/dec
		switch e.Op {
		case ast.UoBnot:
			return v.evalUnaryPiecewiseExpr(expr, scalarBnot)
		case ast.UoLnot:
			return !v.boolFixup(expr).(ast.BoolValue)
		case ast.UoMinus:
			return v.evalUnaryPiecewiseExpr(expr, scalarMinus)
		case ast.UoPlus:
			return expr
		}

	case *ast.ConditionalExpr:
		cond := v.eval(e.Cond)
		if cond.(ast.BoolValue) {
			return v.eval(e.TrueExpr)
		} else {
			return v.eval(e.FalseExpr)
		}

	case *ast.DotExpr:
		if t, ok := e.Expr.Type().(*ast.ArrayType); ok {
			if e.Field != "length" {
				panic(fmt.Errorf("Unexpected array field '%s'.", e.Field))
			}
			length := ast.UintValue(t.ComputedSize)
			return ast.FunctionValue{
				Func:    func([]ast.Value) ast.Value { return length },
				ValType: e.ExprType.(*ast.FunctionType),
			}
		}

		expr := v.eval(e.Expr)
		switch expr := expr.(type) {
		case ast.StructValue:
			return expr.Members[e.Field]
		case ast.VectorValue:
			getField := func(i uint8) ast.Value {
				pos, _ := ast.GetVectorComponentInfo(rune(e.Field[i]))
				return expr.Members[pos]
			}
			if len(e.Field) == 1 {
				return getField(0)
			}
			return ast.NewVectorValue(e.ExprType.(*ast.BuiltinType).Type, getField)
		}

	case *ast.ConstantExpr:
		return e.Value

	case *ast.VarRefExpr:
		return v.resolver(e.Sym)

	case *ast.TypeConversionExpr:
		return v.evalTypeConversion(e)

	case *ast.ParenExpr:
		return v.eval(e.Expr)

	case *ast.CallExpr:
		callee := v.eval(e.Callee)
		args := make([]ast.Value, len(e.Args))
		for i := range e.Args {
			args[i] = v.eval(e.Args[i])
		}
		return callee.(ast.FunctionValue).Func(args)

	}
	panic(fmt.Errorf("Unknown expression kind %T.", e))
}

// Evaluate evaluates an expression in a GLES Shading Language program. It assumes the expression
// has already been typed and statically checked for correctness by the sema package. This
// function is suitable to be passed as the evaluator argument to the sema.Analyze function. Its
// arguments are:
//
// - expr the expression to evaluate
//
// - res a function resolving Symbols occuring in the expressions to values
//
// - lang the language whose semantics to employ during evaluation
//
// The function returns the value obtained by evaluating the expression. If the evaluation fails,
// the second result shall contain a list of errors.
func Evaluate(expr ast.Expression, res func(sym ast.ValueSymbol) ast.Value,
	lang ast.Language) (val ast.Value, err []error) {

	// Any evalErrors thrown during the evaluation indicate a malformed expression, and are
	// returned as an error. Any thrown error is let through as it is either an assertion
	// (should not happen in production), or foreign exception.
	defer func() {
		if r := recover(); r != nil {
			if er, ok := r.(evalError); ok {
				err = []error{er}
			} else {
				panic(r)
			}
		}
	}()

	return (&evaluator{resolver: res, language: lang}).eval(expr), nil
}

// EvaluatePreprocessorExpression is a wrapper around Evaluate, which adapts it to processing
// preprocessor expressions. This function is suitable to be passed as the evaluator argument to
// parse.Parse function.
func EvaluatePreprocessorExpression(e ast.Expression) (val ast.IntValue, err []error) {
	var ppErr ast.ErrorCollector
	resolver := func(sym ast.ValueSymbol) ast.Value {
		ppErr.Errorf("Unknown symbol: '%s'.", sym.Name())
		return ast.IntValue(0)
	}

	result, evalErr := Evaluate(e, resolver, ast.LangPreprocessor)
	if len(err) == 0 {
		val = convertInt(result).(ast.IntValue)
	}
	err = append(ppErr.GetErrors(), evalErr...)
	return
}
