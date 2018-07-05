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
	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/gapil/semantic"
)

func (c *C) doCast(s *S, dstTy, srcTy semantic.Type, v *codegen.Value) *codegen.Value {
	srcPtrTy, srcIsPtr := srcTy.(*semantic.Pointer)
	// dstPtrTy, dstIsPtr := dstTy.(*semantic.Pointer)
	srcSliceTy, srcIsSlice := srcTy.(*semantic.Slice)
	dstSliceTy, dstIsSlice := dstTy.(*semantic.Slice)
	srcIsString := srcTy == semantic.StringType
	dstIsString := dstTy == semantic.StringType

	switch {
	case srcIsPtr && srcPtrTy.To == semantic.CharType && dstIsString:
		// char* -> string
		slicePtr := s.Local("slice", c.T.Sli)
		s.Call(c.callbacks.cstringToSlice, s.Ctx, v, slicePtr)
		slice := slicePtr.Load()
		c.plugins.foreach(func(p OnReadListener) { p.OnRead(s, slice, srcPtrTy.Slice) })
		str := s.Call(c.callbacks.sliceToString, s.Ctx, slicePtr)
		c.release(s, slice, slicePrototype)
		c.deferRelease(s, str, semantic.StringType)
		return str
	case srcIsSlice && srcSliceTy.To == semantic.CharType && dstIsString:
		// char[] -> string
		slicePtr := s.LocalInit("slice", v)
		c.plugins.foreach(func(p OnReadListener) { p.OnRead(s, v, srcSliceTy) })
		return s.Call(c.callbacks.sliceToString, s.Ctx, slicePtr)
	case srcIsString && dstIsSlice && dstSliceTy.To == semantic.CharType:
		// string -> char[]
		slicePtr := s.Local("slice", c.T.Sli)
		s.Call(c.callbacks.stringToSlice, s.Ctx, v, slicePtr)
		return slicePtr.Load()
	case srcIsSlice && dstIsSlice:
		// T[] -> T[]
		root := v.Extract(SliceRoot)
		base := v.Extract(SliceBase)
		size := v.Extract(SliceSize)
		pool := v.Extract(SlicePool)
		count := s.Div(size, s.SizeOf(c.T.Capture(srcSliceTy.To)))
		size = s.Mul(count, s.SizeOf(c.T.Capture(dstSliceTy.To)))
		return c.buildSlice(s, root, base, size, count, pool)
	default:
		return v.Cast(c.T.Target(dstTy)) // TODO: capture vs memory.
	}
}

func (c *C) castTargetToCapture(s *S, ty semantic.Type, v *codegen.Value) *codegen.Value {
	ty = semantic.Underlying(ty)
	dstTy, srcTy := c.T.Capture(ty), c.T.Target(ty)
	if srcTy != v.Type() {
		fail("castTargetToCapture called with a value that is not of the target type")
	}
	if dstTy == srcTy {
		return v
	}

	_, isPtr := ty.(*semantic.Pointer)
	_, isClass := ty.(*semantic.Class)
	switch {
	case isPtr: // pointer -> uint64
		return v.Cast(dstTy)
	case isClass:
		if fn, ok := c.T.targetToCapture[ty]; ok {
			tmpTarget := s.Local("cast_target_"+ty.Name(), dstTy)
			tmpSource := s.LocalInit("cast_source_"+ty.Name(), v)
			s.Call(fn, s.Ctx, tmpSource, tmpTarget)
			return tmpTarget.Load()
		}
		fail("castTargetToCapture() cannot handle type %v (%v -> %v)", ty.Name(), srcTy.TypeName(), dstTy.TypeName())
		return nil
	case ty == semantic.IntType, ty == semantic.SizeType:
		return v.Cast(dstTy)
	default:
		fail("castTargetToCapture() cannot handle type %v (%v -> %v)", ty.Name(), srcTy.TypeName(), dstTy.TypeName())
		return nil
	}
}

func (c *C) castCaptureToTarget(s *S, ty semantic.Type, v *codegen.Value) *codegen.Value {
	ty = semantic.Underlying(ty)
	dstTy, srcTy := c.T.Target(ty), c.T.Capture(ty)
	if srcTy != v.Type() {
		fail("castCaptureToTarget called with a value that is not of the capture type %+v, %+v", srcTy, v.Type())
	}
	if dstTy == srcTy {
		return v
	}

	_, isPtr := ty.(*semantic.Pointer)
	_, isClass := ty.(*semantic.Class)
	switch {
	case isPtr: // uint64 -> pointer
		return v.Cast(dstTy)
	case isClass:
		if fn, ok := c.T.captureToTarget[ty]; ok {
			tmpTarget := s.Local("cast_target_"+ty.Name(), dstTy)
			tmpSource := s.LocalInit("cast_source_"+ty.Name(), v)
			s.Call(fn, s.Ctx, tmpSource, tmpTarget)
			return tmpTarget.Load()
		}
		fail("castCaptureToTarget() cannot handle type %v (%v -> %v)", ty.Name(), srcTy.TypeName(), dstTy.TypeName())
		return nil
	case ty == semantic.IntType, ty == semantic.SizeType:
		return v.Cast(dstTy)
	default:
		fail("castCaptureToTarget() cannot handle type %v (%v -> %v)", ty.Name(), srcTy.TypeName(), dstTy.TypeName())
		return nil
	}
}
