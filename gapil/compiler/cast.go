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

func (c *compiler) doCast(s *scope, dstTy, srcTy semantic.Type, v *codegen.Value) *codegen.Value {
	srcPtrTy, srcIsPtr := srcTy.(*semantic.Pointer)
	// dstPtrTy, dstIsPtr := dstTy.(*semantic.Pointer)
	srcSliceTy, srcIsSlice := srcTy.(*semantic.Slice)
	dstSliceTy, dstIsSlice := dstTy.(*semantic.Slice)
	srcIsString := srcTy == semantic.StringType
	dstIsString := dstTy == semantic.StringType

	switch {
	case srcIsPtr && srcPtrTy.To == semantic.CharType && dstIsString:
		// char* -> string
		str := s.Call(c.callbacks.pointerToString, s.ctx, v)
		c.deferRelease(s, str, semantic.StringType)
		return str
	case srcIsSlice && srcSliceTy.To == semantic.CharType && dstIsString:
		// char[] -> string
		addr := v.Extract(sliceBase)
		size := v.Extract(sliceSize)
		pool := v.Extract(slicePool)
		return s.Call(c.callbacks.sliceToString, s.ctx, addr, size, pool)
	case srcIsString && dstIsSlice && dstSliceTy.To == semantic.CharType:
		// string -> char[]
		return s.Call(c.callbacks.stringToSlice, s.ctx, v)
	case srcIsSlice && dstIsSlice:
		// T[] -> T[]
		root := v.Extract(sliceRoot)
		base := v.Extract(sliceBase)
		size := v.Extract(sliceSize)
		pool := v.Extract(slicePool)
		count := s.Div(size, s.SizeOf(c.storageType(srcSliceTy.To)))
		size = s.Mul(count, s.SizeOf(c.storageType(dstSliceTy.To)))
		return c.buildSlice(s, root, base, size, pool)
	default:
		return v.Cast(c.targetType(dstTy)) // TODO: storage vs memory.
	}
}

func (c *compiler) castTargetToStorage(s *scope, ty semantic.Type, v *codegen.Value) *codegen.Value {
	ty = semantic.Underlying(ty)
	dstTy, srcTy := c.storageType(ty), c.targetType(ty)
	if srcTy != v.Type() {
		fail("castTargetToStorage called with a value that is not of the target type")
	}
	if dstTy == srcTy {
		return v
	}

	_, isPtr := ty.(*semantic.Pointer)
	switch {
	case isPtr: // pointer -> uint64
		return v.Cast(dstTy)
	case ty == semantic.IntType:
		return v.Cast(dstTy)
	default:
		fail("castTargetToStorage() cannot handle type %v (%v -> %v)", ty.Name(), srcTy.TypeName(), dstTy.TypeName())
		return nil
	}
}

func (c *compiler) castStorageToTarget(s *scope, ty semantic.Type, v *codegen.Value) *codegen.Value {
	ty = semantic.Underlying(ty)
	dstTy, srcTy := c.targetType(ty), c.storageType(ty)
	if srcTy != v.Type() {
		fail("castStorageToTarget called with a value that is not of the memory type")
	}
	if dstTy == srcTy {
		return v
	}

	_, isPtr := ty.(*semantic.Pointer)

	switch {
	case isPtr: // uint64 -> pointer
		return v.Cast(dstTy)
	case ty == semantic.IntType:
		return v.Cast(dstTy)
	default:
		fail("castStorageToTarget() cannot handle type %v (%v -> %v)", ty.Name(), srcTy.TypeName(), dstTy.TypeName())
		return nil
	}
}
