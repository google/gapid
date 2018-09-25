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

func (c *C) expression(s *S, e semantic.Expression) (out *codegen.Value) {
	c.pushExpression(s, e)
	defer c.popExpression(s)

	if debugExpressions {
		msg := fmt.Sprintf("%T %+v", e, e)
		c.LogI(s, msg)
		defer c.LogI(s, msg+" -- done")
	}

	defer func() {
		if out != nil {
			got, expect := out.Type(), c.T.Target(e.ExpressionType())
			if got != expect {
				fail("expression(%T %+v) returned value of unexpected type. Got: %v, expect: %v", e, e, got, expect)
			}
		}
	}()

	switch e := e.(type) {
	case *semantic.ArrayIndex:
		return c.arrayIndex(s, e)
	case *semantic.ArrayInitializer:
		return c.arrayInitializer(s, e)
	case *semantic.BinaryOp:
		return c.binaryOp(s, e)
	case *semantic.BitTest:
		return c.bitTest(s, e)
	case semantic.BoolValue:
		return c.boolValue(s, e)
	case *semantic.Call:
		return c.call(s, e)
	case *semantic.Cast:
		return c.cast(s, e)
	case *semantic.ClassInitializer:
		return c.classInitializer(s, e)
	case *semantic.Create:
		return c.create(s, e)
	case *semantic.Clone:
		return c.clone(s, e)
	case *semantic.EnumEntry:
		return c.enumEntry(s, e)
	case semantic.Float32Value:
		return c.float32Value(s, e)
	case semantic.Float64Value:
		return c.float64Value(s, e)
	case *semantic.Global:
		return c.global(s, e)
	case *semantic.Ignore:
		return c.ignore(s, e)
	case semantic.Int16Value:
		return c.int16Value(s, e)
	case semantic.Int32Value:
		return c.int32Value(s, e)
	case semantic.Int64Value:
		return c.int64Value(s, e)
	case semantic.Int8Value:
		return c.int8Value(s, e)
	case *semantic.Length:
		return c.length(s, e)
	case *semantic.Local:
		return c.local(s, e)
	case *semantic.Make:
		return c.make(s, e)
	case *semantic.MapContains:
		return c.mapContains(s, e)
	case *semantic.MapIndex:
		return c.mapIndex(s, e)
	case *semantic.Member:
		return c.member(s, e)
	case *semantic.MessageValue:
		return c.message(s, e)
	case semantic.Null:
		return c.null(s, e)
	case *semantic.Observed:
		return c.observed(s, e)
	case *semantic.Parameter:
		return c.Parameter(s, e)
	case *semantic.PointerRange:
		return c.pointerRange(s, e)
	case *semantic.Select:
		return c.select_(s, e)
	case *semantic.SliceIndex:
		return c.sliceIndex(s, e)
	case *semantic.SliceRange:
		return c.sliceRange(s, e)
	case semantic.StringValue:
		return c.stringValue(s, e)
	case semantic.Uint16Value:
		return c.uint16Value(s, e)
	case semantic.Uint32Value:
		return c.uint32Value(s, e)
	case semantic.Uint64Value:
		return c.uint64Value(s, e)
	case semantic.Uint8Value:
		return c.uint8Value(s, e)
	case *semantic.UnaryOp:
		return c.unaryOp(s, e)
	case *semantic.Unknown:
		return c.unknown(s, e)
	default:
		panic(fmt.Errorf("Unexpected expression type %T", e))
	}
}

func (c *C) arrayIndex(s *S, e *semantic.ArrayIndex) *codegen.Value {
	return c.expressionAddr(s, e).Load()
}

func (c *C) arrayInitializer(s *S, e *semantic.ArrayInitializer) *codegen.Value {
	arr := s.Zero(c.T.Target(e.ExpressionType()))
	for i, e := range e.Values {
		arr = arr.Insert(i, c.expression(s, e))
	}
	return arr
}

func (c *C) binaryOp(s *S, e *semantic.BinaryOp) *codegen.Value {
	op, lhs := e.Operator, c.expression(s, e.LHS)
	switch op {
	case ast.OpBitShiftLeft:
		// RHS is always unsigned. JIT requires LHS and RHS type to be the same.
		rhs := c.doCast(s, e.LHS.ExpressionType(), e.RHS.ExpressionType(), c.expression(s, e.RHS))
		return c.doBinaryOp(s, op, lhs, rhs)
	case ast.OpBitShiftRight:
		// RHS is always unsigned. JIT requires LHS and RHS type to be the same.
		rhs := c.doCast(s, e.LHS.ExpressionType(), e.RHS.ExpressionType(), c.expression(s, e.RHS))
		return c.doBinaryOp(s, op, lhs, rhs)
	case ast.OpAnd:
		// Handle short-circuits.
		res := s.LocalInit("and-sc", s.Scalar(false))
		s.If(lhs, func(s *S) { res.Store(c.expression(s, e.RHS)) })
		return res.Load()
	case ast.OpOr:
		// Handle short-circuits.
		res := s.LocalInit("or-sc", s.Scalar(true))
		s.If(s.Not(lhs), func(s *S) { res.Store(c.expression(s, e.RHS)) })
		return res.Load()
	default:
		rhs := c.expression(s, e.RHS)
		return c.doBinaryOp(s, op, lhs, rhs)
	}
}

func (c *C) equal(s *S, lhs, rhs *codegen.Value) *codegen.Value {
	return c.doBinaryOp(s, "==", lhs, rhs)
}

func (c *C) doBinaryOp(s *S, op string, lhs, rhs *codegen.Value) *codegen.Value {
	if lhs.Type() == c.T.StrPtr {
		switch op {
		case ast.OpEQ, ast.OpGT, ast.OpLT, ast.OpGE, ast.OpLE, ast.OpNE:
			cmp := s.Call(c.callbacks.stringCompare, lhs, rhs)
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
		if lhs.Type() == c.T.StrPtr {
			str := s.Call(c.callbacks.stringConcat, lhs, rhs)
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

func (c *C) bitTest(s *S, e *semantic.BitTest) *codegen.Value {
	bits := c.expression(s, e.Bits)
	bitfield := c.expression(s, e.Bitfield)
	mask := s.And(bits, bitfield)
	return s.NotEqual(mask, s.Zero(mask.Type()))
}

func (c *C) boolValue(s *S, e semantic.BoolValue) *codegen.Value {
	return s.Scalar(bool(e))
}

func (c *C) call(s *S, e *semantic.Call) *codegen.Value {
	tf := e.Target.Function
	if tf.Extern {
		return c.callExtern(s, e)
	}

	args := make([]*codegen.Value, len(e.Arguments)+1)
	args[0] = s.Ctx
	for i, a := range e.Arguments {
		args[i+1] = c.expression(s, a).SetName(tf.FullParameters[i].Name())
	}
	f, ok := c.subroutines[tf]
	if !ok && tf.Subroutine {
		// Likely a subroutine calling another subrotine that hasn't been compiled yet.
		// Compile it now.
		c.subroutine(tf)
		f, ok = c.subroutines[tf]
	}
	if !ok {
		panic(fmt.Errorf("Couldn't resolve call target %v", tf.Name()))
	}

	var res *codegen.Value

	switch {
	case tf.Subroutine:
		needCleanup := false
		for s := s; s != nil; s = s.parent {
			if len(s.pendingRefRels.list) > 0 || len(s.onExitLogic) > 0 {
				needCleanup = true
				break
			}
		}
		if needCleanup {
			cleanup := func() {
				for s := s; s != nil; s = s.parent {
					s.exit()
				}
			}
			res = s.Invoke(f, cleanup, args...)
		} else {
			res = s.Call(f, args...)
		}

	default:
		res = s.Call(f, args...)
	}

	if res != nil {
		c.deferRelease(s, res, tf.Return.Type)
	}

	return res
}

func (c *C) cast(s *S, e *semantic.Cast) *codegen.Value {
	dstTy := semantic.Underlying(e.Type)
	srcTy := semantic.Underlying(e.Object.ExpressionType())
	v := c.expression(s, e.Object)
	return c.doCast(s, dstTy, srcTy, v)
}

func (c *C) classInitializer(s *S, e *semantic.ClassInitializer) *codegen.Value {
	class := s.Undef(c.T.Target(e.ExpressionType()))
	for i, iv := range e.InitialValues() {
		f := e.Class.Fields[i]
		var val *codegen.Value
		if iv != nil {
			val = c.expression(s, iv)
		} else {
			val = c.initialValue(s, f.Type)
		}
		c.reference(s, val, f.Type)
		class = class.Insert(f.Name(), val)
	}
	c.deferRelease(s, class, e.Class)
	return class
}

func (c *C) create(s *S, e *semantic.Create) *codegen.Value {
	refPtrTy := c.T.Target(e.Type).(codegen.Pointer)
	refTy := refPtrTy.Element
	ptr := c.Alloc(s, s.Scalar(uint64(1)), refTy)
	ptr.Index(0, RefRefCount).Store(s.Scalar(uint32(1)))
	ptr.Index(0, RefArena).Store(s.Arena)
	class := c.classInitializer(s, e.Initializer)
	c.reference(s, class, e.Type.To)
	ptr.Index(0, RefValue).Store(class)
	c.deferRelease(s, ptr, e.Type)
	return ptr
}

func (c *C) clone(s *S, e *semantic.Clone) *codegen.Value {
	src := c.expression(s, e.Slice)
	size := src.Extract(SliceSize)
	count := src.Extract(SliceCount)
	dst := c.MakeSlice(s, size, count)
	c.plugins.foreach(func(p OnReadListener) { p.OnRead(s, src, e.Type) })
	c.CopySlice(s, dst, src)
	c.plugins.foreach(func(p OnWriteListener) { p.OnWrite(s, dst, e.Type) })
	c.deferRelease(s, dst, e.Type)
	return dst
}

func (c *C) enumEntry(s *S, e *semantic.EnumEntry) *codegen.Value {
	return s.Scalar(e.Value)
}

func (c *C) float32Value(s *S, e semantic.Float32Value) *codegen.Value {
	return s.Scalar(float32(e))
}

func (c *C) float64Value(s *S, e semantic.Float64Value) *codegen.Value {
	return s.Scalar(float64(e))
}

func (c *C) global(s *S, e *semantic.Global) *codegen.Value {
	if e.Name() == semantic.BuiltinThreadGlobal.Name() {
		return s.CurrentThread
	}
	return s.Globals.Index(0, c.CurrentAPI().Name(), e.Name()).Load()
}

func (c *C) ignore(s *S, e *semantic.Ignore) *codegen.Value {
	panic("Unreachable")
}

func (c *C) int8Value(s *S, e semantic.Int8Value) *codegen.Value {
	return s.Scalar(int8(e))
}

func (c *C) int16Value(s *S, e semantic.Int16Value) *codegen.Value {
	return s.Scalar(int16(e))
}

func (c *C) int32Value(s *S, e semantic.Int32Value) *codegen.Value {
	return s.Scalar(int32(e))
}

func (c *C) int64Value(s *S, e semantic.Int64Value) *codegen.Value {
	return s.Scalar(int64(e))
}

func (c *C) length(s *S, e *semantic.Length) *codegen.Value {
	o := c.expression(s, e.Object)
	var l *codegen.Value
	switch ty := semantic.Underlying(e.Object.ExpressionType()).(type) {
	case *semantic.Slice:
		l = o.Extract(SliceCount)
	case *semantic.Map:
		l = o.Index(0, MapCount).Load()
	case *semantic.Builtin:
		switch ty {
		case semantic.StringType:
			l = o.Index(0, StringLength).Load()
		}
	}
	if l == nil {
		fail("Unhandled length expression type %v", e.Object.ExpressionType().Name())
	}
	return c.doCast(s, e.Type, semantic.Uint64Type, l)
}

func (c *C) local(s *S, e *semantic.Local) *codegen.Value {
	l, ok := s.locals[e]
	if !ok {
		locals := make([]string, 0, len(s.locals))
		for l := range s.locals {
			locals = append(locals, fmt.Sprintf(" • %v", l.Name()))
		}
		fail("Couldn't locate local '%v'. Have locals:\n%v", e.Name(), strings.Join(locals, "\n"))
	}
	if l.isPtr {
		return l.val.Load()
	}
	return l.val
}

func (c *C) make(s *S, e *semantic.Make) *codegen.Value {
	elTy := c.T.Capture(e.Type.To)
	count := c.expression(s, e.Size).Cast(c.T.Uint64)
	size := s.Mul(count, s.SizeOf(elTy))
	slice := c.MakeSlice(s, size, count)
	c.deferRelease(s, slice, e.Type)
	return slice
}

func (c *C) mapContains(s *S, e *semantic.MapContains) *codegen.Value {
	m := c.expression(s, e.Map)
	k := c.expression(s, e.Key)
	return s.Call(c.T.Maps[e.Type].Contains, m, k).SetName("map_contains")
}

func (c *C) mapIndex(s *S, e *semantic.MapIndex) *codegen.Value {
	m := c.expression(s, e.Map)
	k := c.expression(s, e.Index)

	res := s.Local("map_lookup", c.T.Target(e.Type.ValueType))
	ptr := s.Call(c.T.Maps[e.Type].Index, m, k, s.Scalar(false))
	s.IfElse(ptr.IsNull(), func(s *S) {
		val := c.initialValue(s, e.Type.ValueType)
		c.reference(s, val, e.Type.ValueType)
		res.Store(val)
	}, func(s *S) {
		val := ptr.Load()
		c.reference(s, val, e.Type.ValueType)
		res.Store(val)
	})

	out := res.Load()
	c.deferRelease(s, out, e.Type.ValueType)
	return out
}

func (c *C) member(s *S, e *semantic.Member) *codegen.Value {
	obj := c.expression(s, e.Object)
	switch ty := semantic.Underlying(e.Object.ExpressionType()).(type) {
	case *semantic.Class:
		return obj.Extract(e.Field.Name())
	case *semantic.Reference:
		val := obj.Index(0, RefValue, e.Field.Name()).Load()
		c.reference(s, val, e.Field.Type)
		c.deferRelease(s, val, e.Field.Type)
		return val
	default:
		fail("Unexpected type for member: '%v'", ty)
		return nil
	}
}

func (c *C) message(s *S, e *semantic.MessageValue) *codegen.Value {
	args := c.Alloc(s, s.Scalar(len(e.Arguments)+1), c.T.MsgArg)
	for i, a := range e.Arguments {
		val := c.expression(s, a.Value)
		c.reference(s, val, a.Field.Type)
		args.Index(i, MsgArgName).Store(s.Scalar(a.Field.Name()))
		args.Index(i, MsgArgValue).Store(c.packAny(s, a.Field.Type, val))
	}
	args.Index(len(e.Arguments)).Store(s.Zero(c.T.MsgArg))

	msg := c.Alloc(s, s.Scalar(1), c.T.Msg)
	msg.Index(0, MsgRefCount).Store(s.Scalar(uint32(1)))
	msg.Index(0, MsgArena).Store(s.Arena)
	msg.Index(0, MsgIdentifier).Store(s.Scalar(e.AST.Name.Value))
	msg.Index(0, MsgArgs).Store(args)

	c.deferRelease(s, msg, semantic.MessageType)

	return msg
}

func (c *C) null(s *S, e semantic.Null) *codegen.Value {
	return c.initialValue(s, e.Type)
}

func (c *C) observed(s *S, e *semantic.Observed) *codegen.Value {
	return c.Parameter(s, e.Parameter)
}

// Parameter returns the loaded parameter value, failing if the parameter cannot
// be found.
func (c *C) Parameter(s *S, e *semantic.Parameter) *codegen.Value {
	p, ok := s.Parameters[e]
	if !ok {
		params := make([]string, 0, len(s.Parameters))
		for p := range s.Parameters {
			params = append(params, fmt.Sprintf(" • %v", p.Name()))
		}
		c.Fail("Couldn't locate parameter '%v'. Have parameters:\n%v",
			e.Name(), strings.Join(params, "\n"))
	}
	return p
}

func (c *C) pointerRange(s *S, e *semantic.PointerRange) *codegen.Value {
	elTy := c.T.Capture(e.Type.To)
	root := c.expression(s, e.Pointer).Cast(c.T.Uint64).SetName("root")
	start := c.expression(s, e.Range.LHS).Cast(c.T.Uint64).SetName("start")
	end := c.expression(s, e.Range.RHS).Cast(c.T.Uint64).SetName("end")
	offset := s.Mul(start, s.SizeOf(elTy)).Cast(c.T.Uint64).SetName("offset")
	count := s.Sub(end, start).SetName("count")
	size := s.Mul(count, s.SizeOf(elTy)).Cast(c.T.Uint64).SetName("size")
	return s.Zero(c.T.Sli).
		Insert(SliceRoot, root).
		Insert(SliceBase, s.Add(root, offset)).
		Insert(SliceSize, size).
		Insert(SliceCount, count)
}

func (c *C) select_(s *S, e *semantic.Select) *codegen.Value {
	val := c.expression(s, e.Value)

	cases := make([]SwitchCase, len(e.Choices))
	res := s.Local("select_result", c.T.Target(e.Type))
	for i, choice := range e.Choices {
		i, choice := i, choice
		cases[i] = SwitchCase{
			Conditions: func(s *S) []*codegen.Value {
				conds := make([]*codegen.Value, len(choice.Conditions))
				for i, cond := range choice.Conditions {
					conds[i] = c.equal(s, val, c.expression(s, cond))
				}
				return conds
			},
			Block: func(s *S) {
				val := c.expression(s, choice.Expression)
				c.reference(s, val, e.Type)
				res.Store(val)
			},
		}
	}

	var def func(s *S)
	if e.Default != nil {
		def = func(s *S) {
			val := c.expression(s, e.Default)
			c.reference(s, val, e.Type)
			res.Store(val)
		}
	}

	s.Switch(cases, def)

	out := res.Load()
	c.deferRelease(s, out, e.Type)
	return out
}

func (c *C) sliceIndex(s *S, e *semantic.SliceIndex) *codegen.Value {
	index := c.expression(s, e.Index).Cast(c.T.Uint64).SetName("index")
	slice := c.expression(s, e.Slice)

	elTy := e.Type.To
	targetTy := c.T.Target(e.Type.To)
	captureTy := c.T.Capture(e.Type.To)
	captureSize := s.Scalar(uint64(c.T.CaptureTypes.SizeOf(elTy)))
	captureStride := s.Scalar(uint64(c.T.CaptureTypes.StrideOf(elTy)))

	base := slice.Extract(SliceBase)
	offset := s.Mul(index, captureStride)
	subslice := slice.
		Insert(SliceBase, s.Add(base, offset)).
		Insert(SliceSize, captureSize).
		Insert(SliceCount, s.Scalar(uint64(1)))
	subslicePtr := s.LocalInit("subslice", subslice)

	read := func(elType codegen.Type) *codegen.Value {
		c.plugins.foreach(func(p OnReadListener) { p.OnRead(s, subslice, e.Type) })
		return c.SliceDataForRead(s, subslicePtr, elType).Load()
	}
	if targetTy == captureTy {
		return read(targetTy)
	}
	return c.castCaptureToTarget(s, elTy, read(captureTy))
}

func (c *C) sliceRange(s *S, e *semantic.SliceRange) *codegen.Value {
	slice := c.expression(s, e.Slice)
	elTy := c.T.Capture(e.Type.To)
	elPtrTy := c.T.Pointer(elTy)
	base := slice.Extract(SliceBase).Cast(elPtrTy) // T*
	from := c.expression(s, e.Range.LHS).SetName("slice_from")
	to := c.expression(s, e.Range.RHS).SetName("slice_to")
	start := base.Index(from).SetName("slice_start")                                  // T*
	end := base.Index(to).SetName("slice_end")                                        // T*
	size := s.Sub(end.Cast(c.T.Uint64), start.Cast(c.T.Uint64)).SetName("slice_size") // u64

	slice = slice.Insert(SliceCount, s.Sub(to, from))
	slice = slice.Insert(SliceSize, size)
	slice = slice.Insert(SliceBase, start.Cast(c.T.Uint64))
	// TODO: Check sub-slice is within original slice bounds.
	return slice
}

func (c *C) stringValue(s *S, e semantic.StringValue) *codegen.Value {
	str := c.MakeString(s, s.Scalar(uint64(len(e))), s.Scalar(string(e)))
	c.deferRelease(s, str, semantic.StringType)
	return str
}

func (c *C) uint8Value(s *S, e semantic.Uint8Value) *codegen.Value {
	return s.Scalar(uint8(e))
}

func (c *C) uint16Value(s *S, e semantic.Uint16Value) *codegen.Value {
	return s.Scalar(uint16(e))
}

func (c *C) uint32Value(s *S, e semantic.Uint32Value) *codegen.Value {
	return s.Scalar(uint32(e))
}

func (c *C) uint64Value(s *S, e semantic.Uint64Value) *codegen.Value {
	return s.Scalar(uint64(e))
}

func (c *C) unaryOp(s *S, e *semantic.UnaryOp) *codegen.Value {
	switch e.Operator {
	case ast.OpNot:
		return s.Not(c.expression(s, e.Expression))
	}
	fail("unary operator '%v' not implemented", e.Operator)
	return nil
}

func (c *C) unknown(s *S, e *semantic.Unknown) *codegen.Value {
	return c.expression(s, e.Inferred)
}
