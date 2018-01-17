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

package compiler

import (
	"fmt"
	"strings"

	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/semantic"
)

func (c *compiler) expression(s *scope, e semantic.Expression) *codegen.Value {
	old := c.setCurrentExpression(s, e)
	var out *codegen.Value
	switch e := e.(type) {
	case *semantic.ArrayIndex:
		out = c.arrayIndex(s, e)
	case *semantic.ArrayInitializer:
		out = c.arrayInitializer(s, e)
	case *semantic.BinaryOp:
		out = c.binaryOp(s, e)
	case *semantic.BitTest:
		out = c.bitTest(s, e)
	case semantic.BoolValue:
		out = c.boolValue(s, e)
	case *semantic.Call:
		out = c.call(s, e)
	case *semantic.Cast:
		out = c.cast(s, e)
	case *semantic.ClassInitializer:
		out = c.classInitializer(s, e)
	case *semantic.Create:
		out = c.create(s, e)
	case *semantic.Clone:
		out = c.clone(s, e)
	case *semantic.EnumEntry:
		out = c.enumEntry(s, e)
	case semantic.Float32Value:
		out = c.float32Value(s, e)
	case semantic.Float64Value:
		out = c.float64Value(s, e)
	case *semantic.Global:
		out = c.global(s, e)
	case *semantic.Ignore:
		out = c.ignore(s, e)
	case semantic.Int16Value:
		out = c.int16Value(s, e)
	case semantic.Int32Value:
		out = c.int32Value(s, e)
	case semantic.Int64Value:
		out = c.int64Value(s, e)
	case semantic.Int8Value:
		out = c.int8Value(s, e)
	case *semantic.Label:
		out = c.label(s, e)
	case *semantic.Length:
		out = c.length(s, e)
	case *semantic.Local:
		out = c.local(s, e)
	case *semantic.Make:
		out = c.make(s, e)
	case *semantic.MapContains:
		out = c.mapContains(s, e)
	case *semantic.MapIndex:
		out = c.mapIndex(s, e)
	case *semantic.Member:
		out = c.member(s, e)
	case *semantic.MessageValue:
		out = c.message(s, e)
	case semantic.Null:
		out = c.null(s, e)
	case *semantic.Observed:
		out = c.observed(s, e)
	case *semantic.Parameter:
		out = c.parameter(s, e)
	case *semantic.PointerRange:
		out = c.pointerRange(s, e)
	case *semantic.Select:
		out = c.select_(s, e)
	case *semantic.SliceIndex:
		out = c.sliceIndex(s, e)
	case *semantic.SliceRange:
		out = c.sliceRange(s, e)
	case semantic.StringValue:
		out = c.stringValue(s, e)
	case semantic.Uint16Value:
		out = c.uint16Value(s, e)
	case semantic.Uint32Value:
		out = c.uint32Value(s, e)
	case semantic.Uint64Value:
		out = c.uint64Value(s, e)
	case semantic.Uint8Value:
		out = c.uint8Value(s, e)
	case *semantic.UnaryOp:
		out = c.unaryOp(s, e)
	case *semantic.Unknown:
		out = c.unknown(s, e)
	default:
		panic(fmt.Errorf("Unexpected expression type %T", e))
	}
	c.setCurrentExpression(s, old)
	return out
}

func (c *compiler) arrayIndex(s *scope, e *semantic.ArrayIndex) *codegen.Value {
	return c.expressionAddr(s, e).Load()
}

func (c *compiler) arrayInitializer(s *scope, e *semantic.ArrayInitializer) *codegen.Value {
	arr := s.Zero(c.targetType(e.ExpressionType()))
	for i, e := range e.Values {
		arr = arr.Insert(i, c.expression(s, e))
	}
	return arr
}

func (c *compiler) binaryOp(s *scope, e *semantic.BinaryOp) *codegen.Value {
	op, lhs, rhs := e.Operator, c.expression(s, e.LHS), c.expression(s, e.RHS)
	switch op {
	case ast.OpBitShiftLeft:
		// RHS is always unsigned. JIT requires LHS and RHS type to be the same.
		rhs = c.doCast(s, e.LHS.ExpressionType(), e.RHS.ExpressionType(), rhs)
	case ast.OpBitShiftRight:
		// RHS is always unsigned. JIT requires LHS and RHS type to be the same.
		rhs = c.doCast(s, e.LHS.ExpressionType(), e.RHS.ExpressionType(), rhs)
	}
	return c.doBinaryOp(s, op, lhs, rhs)
}

func (c *compiler) equal(s *scope, lhs, rhs *codegen.Value) *codegen.Value {
	return c.doBinaryOp(s, "==", lhs, rhs)
}

func (c *compiler) doBinaryOp(s *scope, op string, lhs, rhs *codegen.Value) *codegen.Value {
	if lhs.Type() == c.ty.strPtr {
		switch op {
		case ast.OpEQ, ast.OpGT, ast.OpLT, ast.OpGE, ast.OpLE, ast.OpNE:
			cmp := s.Call(c.callbacks.stringCompare, s.ctx, lhs, rhs)
			lhs, rhs = cmp, s.Zero(cmp.Type())
		}
	}

	switch op {
	case ast.OpEQ:
		return s.Equal(lhs, rhs)
	case ast.OpGT:
		return s.GreaterThan(lhs, rhs)
	case ast.OpLT:
		return s.LessThan(lhs, rhs)
	case ast.OpGE:
		return s.GreaterOrEqualTo(lhs, rhs)
	case ast.OpLE:
		return s.LessOrEqualTo(lhs, rhs)
	case ast.OpNE:
		return s.NotEqual(lhs, rhs)
	case ast.OpOr, ast.OpBitwiseOr:
		return s.Or(lhs, rhs)
	case ast.OpAnd, ast.OpBitwiseAnd:
		return s.And(lhs, rhs)
	case ast.OpPlus:
		if lhs.Type() == c.ty.strPtr {
			str := s.Call(c.callbacks.stringConcat, s.ctx, lhs, rhs)
			c.deferRelease(s, str, semantic.StringType)
			return str
		}
		return s.Add(lhs, rhs)
	case ast.OpMinus:
		return s.Sub(lhs, rhs)
	case ast.OpMultiply:
		return s.Mul(lhs, rhs)
	case ast.OpDivide:
		return s.Div(lhs, rhs)
	case ast.OpBitShiftLeft:
		return s.ShiftLeft(lhs, rhs)
	case ast.OpBitShiftRight:
		return s.ShiftRight(lhs, rhs)
	case ast.OpRange:
	case ast.OpIn:
	}
	fail("binary operator '%v' not implemented", op)
	return nil
}

func (c *compiler) bitTest(s *scope, e *semantic.BitTest) *codegen.Value {
	bits := c.expression(s, e.Bits)
	bitfield := c.expression(s, e.Bitfield)
	mask := s.And(bits, bitfield)
	return s.NotEqual(mask, s.Zero(mask.Type()))
}

func (c *compiler) boolValue(s *scope, e semantic.BoolValue) *codegen.Value {
	return s.Scalar(bool(e))
}

func (c *compiler) call(s *scope, e *semantic.Call) *codegen.Value {
	tf := e.Target.Function
	if extern, ok := c.externs[tf]; ok {
		return c.callExtern(s, extern, e)
	}
	args := make([]*codegen.Value, len(e.Arguments)+1)
	args[0] = s.ctx
	for i, a := range e.Arguments {
		args[i+1] = c.expression(s, a).SetName(tf.FullParameters[i].Name())
	}
	f, ok := c.functions[tf]
	if !ok && tf.Subroutine {
		// Likely a subroutine calling another subrotine that hasn't been compiled yet.
		// Compile it now.
		c.subroutine(tf)
		f, ok = c.functions[tf]
	}
	if !ok {
		panic(fmt.Errorf("Couldn't resolve call target %v", tf.Name()))
	}
	res := s.Call(f, args...)

	if tf.Subroutine {
		// Subroutines return a <error, value> pair.
		// Check the error.
		err := res.Extract(retError)
		s.If(s.NotEqual(err, s.Scalar(ErrSuccess)), func() {
			retTy := c.returnType(c.currentFunc)
			s.Return(s.Zero(retTy).Insert(retError, err))
		})
		if tf.Return.Type == semantic.VoidType {
			return nil
		}
		// Return the value.
		return res.Extract(retValue)
	}

	return res
}

func (c *compiler) cast(s *scope, e *semantic.Cast) *codegen.Value {
	dstTy := semantic.Underlying(e.Type)
	srcTy := semantic.Underlying(e.Object.ExpressionType())
	v := c.expression(s, e.Object)
	return c.doCast(s, dstTy, srcTy, v)
}

func (c *compiler) classInitializer(s *scope, e *semantic.ClassInitializer) *codegen.Value {
	class := c.classInitializerNoRelease(s, e)
	c.deferRelease(s, class, e.Class)
	return class
}

func (c *compiler) classInitializerNoRelease(s *scope, e *semantic.ClassInitializer) *codegen.Value {
	class := s.Undef(c.targetType(e.ExpressionType()))
	for i, iv := range e.InitialValues() {
		if iv != nil {
			class = class.Insert(i, c.expression(s, iv))
		} else {
			class = class.Insert(i, c.initialValue(s, e.Class.Fields[i].Type))
		}
	}
	c.reference(s, class, e.Class) // references all referencable fields.
	return class
}

func (c *compiler) create(s *scope, e *semantic.Create) *codegen.Value {
	refPtrTy := c.targetType(e.Type).(codegen.Pointer)
	refTy := refPtrTy.Element
	ptr := c.alloc(s, s.Scalar(uint64(1)), refTy)
	ptr.Index(0, refRefCount).Store(s.Scalar(uint32(1)))
	ptr.Index(0, refValue).Store(c.classInitializerNoRelease(s, e.Initializer))
	c.deferRelease(s, ptr, e.Type)
	return ptr
}

func (c *compiler) clone(s *scope, e *semantic.Clone) *codegen.Value {
	src := c.expression(s, e.Slice)
	size := src.Extract(sliceSize)
	dstPtr := s.Local("dstPtr", c.ty.sli)
	s.Call(c.callbacks.makeSlice, s.ctx, size, dstPtr)
	dst := dstPtr.Load()
	c.doCopy(s, dst, src, e.Type.To)
	c.deferRelease(s, dst, e.Type)
	return dst
}

func (c *compiler) enumEntry(s *scope, e *semantic.EnumEntry) *codegen.Value {
	return s.Scalar(e.Value)
}

func (c *compiler) float32Value(s *scope, e semantic.Float32Value) *codegen.Value {
	return s.Scalar(float32(e))
}

func (c *compiler) float64Value(s *scope, e semantic.Float64Value) *codegen.Value {
	return s.Scalar(float64(e))
}

func (c *compiler) global(s *scope, e *semantic.Global) *codegen.Value {
	return s.globals.Index(0, e.Name()).Load()
}

func (c *compiler) ignore(s *scope, e *semantic.Ignore) *codegen.Value {
	panic("Unreachable")
}

func (c *compiler) int8Value(s *scope, e semantic.Int8Value) *codegen.Value {
	return s.Scalar(int8(e))
}

func (c *compiler) int16Value(s *scope, e semantic.Int16Value) *codegen.Value {
	return s.Scalar(int16(e))
}

func (c *compiler) int32Value(s *scope, e semantic.Int32Value) *codegen.Value {
	return s.Scalar(int32(e))
}

func (c *compiler) int64Value(s *scope, e semantic.Int64Value) *codegen.Value {
	return s.Scalar(int64(e))
}

func (c *compiler) label(s *scope, e *semantic.Label) *codegen.Value {
	return c.expression(s, e.Value)
}

func (c *compiler) length(s *scope, e *semantic.Length) *codegen.Value {
	o := c.expression(s, e.Object)
	var l *codegen.Value
	switch ty := semantic.Underlying(e.Object.ExpressionType()).(type) {
	case *semantic.Slice:
		size := o.Extract(sliceSize)
		l = s.Div(size, s.SizeOf(c.storageType(ty.To)))
	case *semantic.Map:
		l = o.Index(0, mapCount).Load()
	case *semantic.Builtin:
		switch ty {
		case semantic.StringType:
			l = o.Index(0, stringLength).Load()
		}
	}
	if l == nil {
		fail("Unhandled length expression type %v", e.Object.ExpressionType().Name())
	}
	return c.doCast(s, e.Type, semantic.Uint32Type, l)
}

func (c *compiler) local(s *scope, e *semantic.Local) *codegen.Value {
	l, ok := s.locals[e]
	if !ok {
		locals := make([]string, 0, len(s.locals))
		for l := range s.locals {
			locals = append(locals, fmt.Sprintf(" • %v", l.Name()))
		}
		fail("Couldn't locate local '%v'. Have locals:\n%v", e.Name(), strings.Join(locals, "\n"))
	}
	return l.Load()
}

func (c *compiler) make(s *scope, e *semantic.Make) *codegen.Value {
	elTy := c.storageType(e.Type.To)
	count := c.expression(s, e.Size).Cast(c.ty.Uint64)
	size := s.Mul(count, s.SizeOf(elTy))
	slicePtr := s.Local("slicePtr", c.ty.sli)
	s.Call(c.callbacks.makeSlice, s.ctx, size, slicePtr)
	slice := slicePtr.Load()
	c.deferRelease(s, slice, e.Type)
	return slice
}

func (c *compiler) mapContains(s *scope, e *semantic.MapContains) *codegen.Value {
	m := c.expression(s, e.Map)
	k := c.expression(s, e.Key)
	return s.Call(c.ty.maps[e.Type].Contains, s.ctx, m, k).SetName("map_contains")
}

func (c *compiler) mapIndex(s *scope, e *semantic.MapIndex) *codegen.Value {
	m := c.expression(s, e.Map)
	k := c.expression(s, e.Index)
	res := s.Call(c.ty.maps[e.Type].Lookup, s.ctx, m, k).SetName("map_lookup")
	c.deferRelease(s, res, e.Type.ValueType)
	return res
}

func (c *compiler) member(s *scope, e *semantic.Member) *codegen.Value {
	obj := c.expression(s, e.Object)
	switch ty := semantic.Underlying(e.Object.ExpressionType()).(type) {
	case *semantic.Class:
		return obj.Extract(e.Field.Name())
	case *semantic.Reference:
		return obj.Index(0, refValue, e.Field.Name()).Load()
	default:
		fail("Unexpected type for member: '%v'", ty)
		return nil
	}
}

func (c *compiler) message(s *scope, e *semantic.MessageValue) *codegen.Value {
	return s.Zero(c.targetType(e.ExpressionType())).SetName("TODO:message") // TODO
}

func (c *compiler) null(s *scope, e semantic.Null) *codegen.Value {
	return s.Zero(c.targetType(e.Type))
}

func (c *compiler) observed(s *scope, e *semantic.Observed) *codegen.Value {
	return c.parameter(s, e.Parameter)
}

func (c *compiler) parameter(s *scope, e *semantic.Parameter) *codegen.Value {
	p, ok := s.parameters[e]
	if !ok {
		params := make([]string, 0, len(s.parameters))
		for p := range s.parameters {
			params = append(params, fmt.Sprintf(" • %v", p.Name()))
		}
		panic(fmt.Errorf("Couldn't locate parameter '%v'. Have parameters:\n%v",
			e.Name(), strings.Join(params, "\n")))
	}
	return p
}

func (c *compiler) pointerRange(s *scope, e *semantic.PointerRange) *codegen.Value {
	p := c.expression(s, e.Pointer)
	elTy := c.storageType(e.Type.To)
	u64 := c.ty.Uint64
	address := p.Cast(u64).SetName("address")
	start := c.expression(s, e.Range.LHS).Cast(c.ty.Uint64).SetName("start")
	end := c.expression(s, e.Range.RHS).Cast(c.ty.Uint64).SetName("end")
	offset := s.Mul(start, s.SizeOf(elTy)).Cast(c.ty.Uint64).SetName("offset")
	count := s.Sub(end, start).SetName("count")
	size := s.Mul(count, s.SizeOf(elTy)).Cast(c.ty.Uint64).SetName("size")
	slicePtr := s.Local("slicePtr", c.ty.sli)
	s.Call(c.callbacks.pointerToSlice, s.ctx, address, offset, size, slicePtr)
	slice := slicePtr.Load()
	c.deferRelease(s, slice, e.Type)
	return slice
}

func (c *compiler) select_(s *scope, e *semantic.Select) *codegen.Value {
	cases := make([]codegen.SwitchCase, len(e.Choices))
	res := s.Local("selectval", c.targetType(e.Type))
	if e.Default != nil {
		res.Store(c.expression(s, e.Default))
	}
	for i, choice := range e.Choices {
		i, choice := i, choice
		cases[i] = codegen.SwitchCase{
			Conditions: func() []*codegen.Value {
				conds := make([]*codegen.Value, len(choice.Conditions))
				for i, cond := range choice.Conditions {
					conds[i] = c.expression(s, cond)
				}
				return conds
			},
			Block: func() { res.Store(c.expression(s, choice.Expression)) },
		}
	}
	s.Switch(c.expression(s, e.Value), cases, nil)
	return res.Load()
}

func (c *compiler) sliceIndex(s *scope, e *semantic.SliceIndex) *codegen.Value {
	index := c.expression(s, e.Index).Cast(c.ty.Uint64).SetName("index")
	slice := c.expression(s, e.Slice)

	read := func(elType codegen.Type) *codegen.Value {
		base := slice.Extract(sliceBase).Cast(c.ty.Pointer(elType))
		return base.Index(index).Load()
	}

	elTy := e.Type.To
	targetTy := c.targetType(e.Type.To)
	storageTy := c.storageType(e.Type.To)
	if targetTy == storageTy {
		return read(targetTy)
	}
	return c.castStorageToTarget(s, elTy, read(storageTy))
}

func (c *compiler) sliceRange(s *scope, e *semantic.SliceRange) *codegen.Value {
	slice := c.expression(s, e.Slice)
	elTy := c.storageType(e.Type.To)
	elPtrTy := c.ty.Pointer(elTy)
	base := slice.Extract(sliceBase).Cast(elPtrTy) // T*
	from := c.expression(s, e.Range.LHS).SetName("slice_from")
	to := c.expression(s, e.Range.RHS).SetName("slice_to")
	start := base.Index(from).SetName("slice_start")                                    // T*
	end := base.Index(to).SetName("slice_end")                                          // T*
	size := s.Sub(end.Cast(c.ty.Uint64), start.Cast(c.ty.Uint64)).SetName("slice_size") // u64

	slice = slice.Insert(sliceSize, size)
	slice = slice.Insert(sliceBase, start.Cast(c.ty.u8Ptr))
	// TODO: Check sub-slice is within original slice bounds.
	return slice
}

func (c *compiler) stringValue(s *scope, e semantic.StringValue) *codegen.Value {
	str := s.Call(c.callbacks.makeString, s.ctx, s.Scalar(uint64(len(e))), s.GlobalString(string(e)))
	c.deferRelease(s, str, semantic.StringType)
	return str
}

func (c *compiler) uint8Value(s *scope, e semantic.Uint8Value) *codegen.Value {
	return s.Scalar(uint8(e))
}

func (c *compiler) uint16Value(s *scope, e semantic.Uint16Value) *codegen.Value {
	return s.Scalar(uint16(e))
}

func (c *compiler) uint32Value(s *scope, e semantic.Uint32Value) *codegen.Value {
	return s.Scalar(uint32(e))
}

func (c *compiler) uint64Value(s *scope, e semantic.Uint64Value) *codegen.Value {
	return s.Scalar(uint64(e))
}

func (c *compiler) unaryOp(s *scope, e *semantic.UnaryOp) *codegen.Value {
	switch e.Operator {
	case ast.OpNot:
		return s.Not(c.expression(s, e.Expression))
	}
	fail("unary operator '%v' not implemented", e.Operator)
	return nil
}

func (c *compiler) unknown(s *scope, e *semantic.Unknown) *codegen.Value {
	return c.expression(s, e.Inferred)
}
