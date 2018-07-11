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

package resolver

import (
	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/semantic"
)

func unaryOp(rv *resolver, in *ast.UnaryOp) *semantic.UnaryOp {
	out := &semantic.UnaryOp{AST: in}
	out.Operator = in.Operator
	out.Expression = expression(rv, in.Expression)
	et := out.Expression.ExpressionType()
	switch out.Operator {
	case ast.OpNot:
		out.Type = semantic.BoolType
		if !equal(et, semantic.BoolType) {
			rv.errorf(in, "operator %s applied to non bool type %s", out.Operator, typename(et))
		}
	default:
		rv.icef(in, "unhandled unary operator %s", out.Operator)
	}
	rv.mappings.Add(in, out)
	return out
}

func binaryOp(rv *resolver, in *ast.BinaryOp) semantic.Expression {
	if in.Operator == ast.OpIn {
		return binaryOpIn(rv, in)
	}

	var lhs semantic.Expression
	var rhs semantic.Expression
	switch in.LHS.(type) {
	case *ast.Number, *ast.Null, *ast.Definition:
		// leave these to be inferred after the rhs is known
	default:
		lhs = expression(rv, in.LHS)
	}

	switch {
	case in.RHS == nil:
		if lhs == nil {
			// lhs was deferred, but no rhs is available
			lhs = expression(rv, in.LHS)
		}
	case in.Operator == ast.OpBitShiftLeft, in.Operator == ast.OpBitShiftRight:
		rv.with(semantic.Uint32Type, func() { rhs = expression(rv, in.RHS) })
		if lhs == nil {
			rv.with(rhs.ExpressionType(), func() { lhs = expression(rv, in.LHS) })
		}
	case lhs != nil:
		// allow us to infer rhs type from lhs
		rv.with(lhs.ExpressionType(), func() { rhs = expression(rv, in.RHS) })
	default:
		// lhs was deferred, do the rhs first
		rhs = expression(rv, in.RHS)
		// allow us to infer lhs type from rhs
		rv.with(rhs.ExpressionType(), func() { lhs = expression(rv, in.LHS) })
	}

	lt := lhs.ExpressionType()
	rt := semantic.Type(semantic.VoidType)
	if rhs != nil {
		rt = rhs.ExpressionType()
	}
	var out semantic.Expression
	switch in.Operator {
	case ast.OpEQ, ast.OpGT, ast.OpLT, ast.OpGE, ast.OpLE, ast.OpNE:
		if !comparable(lt, rt) {
			rv.errorf(in, "comparison %s %s %s not allowed", typename(lt), in.Operator, typename(rt))
		}
		out = &semantic.BinaryOp{AST: in, LHS: lhs, RHS: rhs, Type: semantic.BoolType, Operator: in.Operator}
	case ast.OpOr, ast.OpAnd:
		if !equal(lt, semantic.BoolType) {
			rv.errorf(in, "lhs of %s is %s not boolean", in.Operator, typename(lt))
		}
		if !equal(rt, semantic.BoolType) {
			rv.errorf(in, "rhs of %s is %s not boolean", in.Operator, typename(rt))
		}
		out = &semantic.BinaryOp{AST: in, LHS: lhs, RHS: rhs, Type: semantic.BoolType, Operator: in.Operator}
	case ast.OpBitwiseAnd, ast.OpBitwiseOr:
		_, ltEnum := rv.findType(in, typename(lt)).(*semantic.Enum)
		if !((ltEnum || isNumber(lt)) && equal(lt, rt)) {
			rv.errorf(in, "incompatible types for bitwise maths %s %s %s", typename(lt), in.Operator, typename(rt))
		}
		out = &semantic.BinaryOp{AST: in, LHS: lhs, RHS: rhs, Type: lt, Operator: in.Operator}
	case ast.OpPlus, ast.OpMinus, ast.OpMultiply, ast.OpDivide:
		if !equal(lt, rt) {
			rv.errorf(in, "incompatible types for maths %s %s %s", typename(lt), in.Operator, typename(rt))
		}
		out = &semantic.BinaryOp{AST: in, LHS: lhs, RHS: rhs, Type: lt, Operator: in.Operator}
	case ast.OpBitShiftLeft, ast.OpBitShiftRight:
		if !isInteger(semantic.Underlying(lt)) {
			rv.errorf(in, "lhs of %s must be an integer, got %s", in.Operator, typename(lt))
		}
		if !isUnsignedInteger(semantic.Underlying(rt)) {
			rv.errorf(in, "lhs of %s must be an unsigned integer, got %s", in.Operator, typename(rt))
		}
		out = &semantic.BinaryOp{AST: in, LHS: lhs, RHS: rhs, Type: lt, Operator: in.Operator}
	case ast.OpRange:
		if !equal(lt, rt) {
			rv.errorf(in, "range %s %s %s not allowed", typename(lt), in.Operator, typename(rt))
		}
		out = &semantic.BinaryOp{AST: in, LHS: lhs, RHS: rhs, Type: lt, Operator: in.Operator}
	case ast.OpSlice:
		out = &semantic.BinaryOp{AST: in, LHS: lhs, RHS: rhs, Type: lt, Operator: in.Operator}
	default:
		rv.icef(in, "unknown binary operator %s", in.Operator)
		return semantic.Invalid{}
	}
	rv.mappings.Add(in, out)
	return out
}

func binaryOpIn(rv *resolver, in *ast.BinaryOp) semantic.Expression {
	rhs := expression(rv, in.RHS)
	var lhs, out semantic.Expression

	switch rt := rhs.ExpressionType().(type) {
	case *semantic.Map:
		rv.with(rt.KeyType, func() { lhs = expression(rv, in.LHS) })
		lt := lhs.ExpressionType()
		if !comparable(lt, rt.KeyType) {
			rv.errorf(in, "%s with type %s, but map key type is %s", in.Operator, typename(lt), typename(rt.KeyType))
		}
		out = &semantic.MapContains{AST: in, Type: rt, Map: rhs, Key: lhs}
	case *semantic.Slice:
		rv.with(rt.To, func() { lhs = expression(rv, in.LHS) })
		lt := lhs.ExpressionType()
		if !comparable(lt, rt.To) {
			rv.errorf(in, "%s with type %s, but slice element type is %s", in.Operator, typename(lt), typename(rt.To))
		}
		out = &semantic.SliceContains{AST: in, Type: rt, Slice: rhs, Value: lhs}
	case *semantic.Enum:
		rv.with(rt, func() { lhs = expression(rv, in.LHS) })
		lt := lhs.ExpressionType()
		if !equal(lt, rt) {
			rv.errorf(in, "enum bittest on %s with %s is not allowed", typename(lt), typename(rt))
		}
		out = &semantic.BitTest{AST: in, Bitfield: rhs, Bits: lhs}
	default:
		rv.errorf(in, "%s only allowed on maps, not %s", in.Operator, typename(rt))
		return semantic.Invalid{}
	}
	rv.mappings.Add(in, out)
	return out
}
