// Copyright (C) 2018 Google Inc.
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

	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/gapil/semantic"
)

func rttiKind(ty semantic.Type) uint32 {
	switch ty := semantic.Underlying(ty).(type) {
	case *semantic.Builtin:
		switch ty {
		case semantic.BoolType:
			return KindBool
		case semantic.IntType:
			return KindInt
		case semantic.UintType:
			return KindUint
		case semantic.SizeType:
			return KindSize
		case semantic.CharType:
			return KindChar
		case semantic.Int8Type:
			return KindS8
		case semantic.Uint8Type:
			return KindU8
		case semantic.Int16Type:
			return KindS16
		case semantic.Uint16Type:
			return KindU16
		case semantic.Int32Type:
			return KindS32
		case semantic.Uint32Type:
			return KindU32
		case semantic.Int64Type:
			return KindS64
		case semantic.Uint64Type:
			return KindU64
		case semantic.Float32Type:
			return KindF32
		case semantic.Float64Type:
			return KindF64
		case semantic.StringType:
			return KindString
		default:
			fail("Unhandled builtin type %v", ty)
		}
	case *semantic.Enum:
		return KindEnum
	case *semantic.StaticArray:
		return KindArray
	case *semantic.Pointer:
		return KindPointer
	case *semantic.Class:
		return KindClass
	case *semantic.Map:
		return KindMap
	default:
		fail("Unhandled type %v", ty)
	}
	return 0
}

func (c *C) rtti(ty semantic.Type) codegen.Global {
	if existing, ok := c.T.rttis[ty]; ok {
		return existing
	}

	typename := fmt.Sprint(ty)

	fields := map[string]interface{}{
		RTTITypeName: c.M.Scalar(typename),
		RTTIKind:     rttiKind(ty),
	}

	if t, ok := ty.(semantic.Owned); ok {
		if api, ok := t.Owner().(*semantic.API); ok {
			fields[RTTIAPIIndex] = c.M.Scalar(uint32(api.Index))
			switch ty := ty.(type) {
			case *semantic.Class:
				fields[RTTITypeIndex] = c.M.Scalar(api.ClassIndex(ty))
			case *semantic.Enum:
				fields[RTTITypeIndex] = c.M.Scalar(api.EnumIndex(ty))
			case *semantic.Slice:
				fields[RTTITypeIndex] = c.M.Scalar(api.SliceIndex(ty))
			case *semantic.Map:
				fields[RTTITypeIndex] = c.M.Scalar(api.MapIndex(ty))
			}
		}
	}

	if refRel, ok := c.refRels.tys[ty]; ok {
		if _, isSlice := ty.(*semantic.Slice); isSlice {
			// TODO: The refrels of a slice pass by value, yet we'll pass by
			// pointer!
			fail("Slices cannot currently be boxed in anys")
		}
		fields[RTTIReference] = refRel.reference
		fields[RTTIRelease] = refRel.release
	}

	rtti := c.M.Global(typename+"-rtti", c.M.ConstStruct(c.T.RTTI, fields))
	c.T.rttis[ty] = rtti
	return rtti
}

func (c *C) packAny(s *S, ty semantic.Type, v *codegen.Value) *codegen.Value {
	c.reference(s, v, ty)

	var any *codegen.Value
	if codegen.IsPointer(c.T.Target(ty)) {
		any = c.Alloc(s, s.Scalar(1), c.T.Any)
	} else {
		alloc := c.Alloc(s, s.Scalar(1), c.T.Struct("",
			codegen.Field{Name: "any", Type: c.T.Any},
			codegen.Field{Name: "val", Type: c.T.Target(ty)},
		))

		val := alloc.Index(0, "val")
		val.Store(v)
		v = val

		any = alloc.Index(0, "any")
	}

	any.Index(0, AnyRefCount).Store(s.Scalar(uint32(1)))
	any.Index(0, AnyArena).Store(s.Arena)
	any.Index(0, AnyRTTI).Store(c.rtti(ty).Value(s.Builder))
	any.Index(0, AnyValue).Store(v.Cast(c.T.VoidPtr))

	c.deferRelease(s, any, semantic.AnyType)

	return any
}

func (c *C) unpackAny(s *S, ty semantic.Type, any *codegen.Value) *codegen.Value {
	dstKind := s.Scalar(rttiKind(ty))

	srcRtti := any.Index(0, AnyRTTI).Load()
	srcKind := srcRtti.Index(0, RTTIKind).Load()

	s.If(s.NotEqual(dstKind, srcKind), func(s *S) {
		c.LogF(s, "Cannot cast boxed value of type %s to %s",
			srcRtti.Index(0, RTTITypeName).Load(),
			s.Scalar(fmt.Sprint(ty)),
		)
	})

	var out *codegen.Value

	target := c.T.Target(ty)
	if codegen.IsPointer(target) {
		out = any.Index(0, AnyValue).Load().Cast(target)
	} else {
		out = any.Index(0, AnyValue).Load().Cast(c.T.Pointer(target)).Load()
	}

	c.reference(s, out, ty)
	c.deferRelease(s, out, ty)

	return out
}
