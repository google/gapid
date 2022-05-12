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
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapil/compiler/mangling"
	"github.com/google/gapid/gapil/semantic"
)

//#include "gapil/runtime/cc/runtime.h"
import "C"

// Types holds all the codegen Types for semantic types and the runtime.
// Types augments the codegen.Types structure.
type Types struct {
	codegen.Types
	Ctx       *codegen.Struct // context_t
	CtxPtr    codegen.Type    // context_t*
	Sli       codegen.Type    // slice_t
	Str       *codegen.Struct // string_t
	StrPtr    codegen.Type    // string_t*
	Arena     *codegen.Struct // arena_t
	ArenaPtr  codegen.Type    // arena_t*
	VoidPtr   codegen.Type    // void* (aliased of uint8_t*)
	Globals   *codegen.Struct // API global variables structure.
	Buf       codegen.Type    // buffer_t
	BufPtr    codegen.Type    // buffer_t*
	Maps      map[*semantic.Map]*MapInfo
	mapImpls  []mapImpl
	target    map[semantic.Type]codegen.Type
	targetABI *device.ABI
	mangled   map[codegen.Type]mangling.Type
}

type memLayoutKey string

func (c *C) declareTypes() {
	c.T.targetABI = c.Settings.TargetABI

	c.T.Types = c.M.Types
	c.T.Ctx = c.T.DeclareStruct("context")
	c.T.CtxPtr = c.T.Pointer(c.T.Ctx)
	c.T.Globals = c.T.DeclareStruct("globals")
	c.T.Sli = c.T.TypeOf(C.slice{})
	c.T.Str = c.T.TypeOf(C.string{}).(*codegen.Struct)
	c.T.StrPtr = c.T.Pointer(c.T.Str)
	c.T.Arena = c.T.DeclareStruct("arena")
	c.T.ArenaPtr = c.T.Pointer(c.T.Arena)
	c.T.VoidPtr = c.T.Pointer(c.T.Void)
	c.T.Buf = c.T.TypeOf(C.buffer{})
	c.T.BufPtr = c.T.Pointer(c.T.Buf)
	c.T.Maps = map[*semantic.Map]*MapInfo{}
	c.T.target = map[semantic.Type]codegen.Type{}
	c.T.mangled = map[codegen.Type]mangling.Type{}

	for _, api := range c.APIs {
		// Forward-declare all the class types.
		for _, t := range api.Classes {
			cgTy := c.T.DeclareStruct("T_" + t.Name())
			c.T.target[t] = cgTy
		}

		// Forward-declare all the reference types.
		for _, t := range api.References {
			cgTy := c.T.DeclareStruct(t.Name())
			c.T.target[t] = c.T.Pointer(cgTy)
		}

		// Forward-declare all the map types.
		for _, t := range api.Maps {
			cgTy := c.T.DeclareStruct(t.Name())
			mapStrTy := cgTy
			mapPtrTy := c.T.Pointer(mapStrTy)
			c.T.target[t] = mapPtrTy
		}

		// Forward-declare all the slice types.
		for _, t := range api.Slices {
			c.T.target[t] = c.T.Sli
		}
	}
}

func (c *C) declareMangling() {
	// Declare the mangled types
	for _, api := range c.APIs {
		for _, t := range api.Classes {
			c.T.mangled[c.T.Target(t)] = &mangling.Class{
				Parent: c.Root,
				Name:   t.Name(),
			}
		}
		for _, t := range api.References {
			refTy := c.T.Target(t).(codegen.Pointer).Element
			c.T.mangled[refTy] = &mangling.Class{
				Parent: c.Root,
				Name:   "Ref",
			}
		}
		for _, t := range api.Maps {
			mapTy := c.T.Target(t).(codegen.Pointer).Element
			c.T.mangled[mapTy] = &mangling.Class{
				Parent: c.Root,
				Name:   "Map",
			}
		}

		// Add template parameters
		for _, t := range api.References {
			refTy := c.T.Target(t).(codegen.Pointer).Element
			c.T.mangled[refTy].(*mangling.Class).TemplateArgs = []mangling.Type{
				c.Mangle(c.T.Target(t.To)),
			}
		}
		for _, t := range api.Maps {
			mapTy := c.T.Target(t).(codegen.Pointer).Element
			c.T.mangled[mapTy].(*mangling.Class).TemplateArgs = []mangling.Type{
				c.Mangle(c.T.Target(t.KeyType)),
				c.Mangle(c.T.Target(t.ValueType)),
			}
		}
	}
}

func (c *C) buildTypes() {
	for _, api := range c.APIs {
		// Build all the class types.
		for _, t := range api.Classes {
			fields := make([]codegen.Field, len(t.Fields))
			for i, f := range t.Fields {
				fields[i] = codegen.Field{Name: f.Name(), Type: c.T.Target(f.Type)}
			}
			c.T.target[t].(*codegen.Struct).SetBody(false, fields...)
		}

		// Build all the reference types.
		for _, t := range api.References {
			// struct ref!T {
			//      uint32_t ref_count;
			//      arena*   arena;
			//      T        value;
			// }
			ptr := c.T.target[t].(codegen.Pointer)
			str := ptr.Element.(*codegen.Struct)
			str.SetBody(false,
				codegen.Field{Name: RefRefCount, Type: c.T.Uint32},
				codegen.Field{Name: RefArena, Type: c.T.ArenaPtr},
				codegen.Field{Name: RefValue, Type: c.T.Target(t.To)},
			)
		}
	}

	// Build all the map types.
	c.defineMapTypes()

	apiGlobals := make([]codegen.Field, len(c.APIs))
	for i, api := range c.APIs {
		fields := make([]codegen.Field, len(api.Globals))
		for i, g := range api.Globals {
			fields[i] = codegen.Field{Name: g.Name(), Type: c.T.Target(g.Type)}
		}
		apiGlobals[i] = codegen.Field{
			Name: api.Name(),
			Type: c.T.Struct(api.Name(), fields...),
		}
	}
	c.T.Globals.SetBody(false, apiGlobals...)
}

// Target returns the codegen type used to represent ty in the target-preferred
// form.
func (t *Types) Target(ty semantic.Type) codegen.Type {
	ty = semantic.Underlying(ty)
	switch ty := ty.(type) {
	case *semantic.Builtin:
		switch ty {
		case semantic.IntType:
			return t.Int
		case semantic.SizeType:
			return t.Size
		}
	case *semantic.StaticArray:
		return t.Array(t.Target(ty.ValueType), int(ty.Size))
	case *semantic.Slice:
		return t.Sli
	case *semantic.Pointer:
		return t.Uintptr
	case *semantic.Class, *semantic.Reference, *semantic.Map:
		if out, ok := t.target[ty]; ok {
			return out
		}
		fail("Target type not registered: '%v' (%T)", ty.Name(), t)
	}
	return t.basic(ty)
}

// AlignOf returns the alignment of this type in bytes for the given memory
// layout.
func (t *Types) AlignOf(layout *device.MemoryLayout, ty semantic.Type) uint64 {
	switch ty := semantic.Underlying(ty).(type) {
	case *semantic.Builtin:
		switch ty {
		case semantic.BoolType:
			return uint64(layout.I8.Alignment)
		case semantic.IntType:
			return uint64(layout.Integer.Alignment)
		case semantic.UintType:
			return uint64(layout.Integer.Alignment)
		case semantic.SizeType:
			return uint64(layout.Size.Alignment)
		case semantic.CharType:
			return uint64(layout.Char.Alignment)
		case semantic.Int8Type:
			return uint64(layout.I8.Alignment)
		case semantic.Uint8Type:
			return uint64(layout.I8.Alignment)
		case semantic.Int16Type:
			return uint64(layout.I16.Alignment)
		case semantic.Uint16Type:
			return uint64(layout.I16.Alignment)
		case semantic.Int32Type:
			return uint64(layout.I32.Alignment)
		case semantic.Uint32Type:
			return uint64(layout.I32.Alignment)
		case semantic.Int64Type:
			return uint64(layout.I64.Alignment)
		case semantic.Uint64Type:
			return uint64(layout.I64.Alignment)
		case semantic.Float32Type:
			return uint64(layout.F32.Alignment)
		case semantic.Float64Type:
			return uint64(layout.F64.Alignment)
		}
	case *semantic.Enum:
		return t.AlignOf(layout, ty.NumberType)
	case *semantic.StaticArray:
		return t.AlignOf(layout, ty.ValueType)
	case *semantic.Pointer:
		return uint64(layout.Pointer.Alignment)
	case *semantic.Class:
		alignment := uint64(1)
		for _, f := range ty.Fields {
			a := t.AlignOf(layout, f.Type)
			if alignment < a {
				alignment = a
			}
		}
		return alignment
	}
	fail("Cannot determine the alignment for %T %v", ty, ty)
	return 1
}

// SizeOf returns size of the type for the given memory layout.
func (t *Types) SizeOf(layout *device.MemoryLayout, ty semantic.Type) uint64 {
	switch ty := semantic.Underlying(ty).(type) {
	case *semantic.Builtin:
		switch ty {
		case semantic.BoolType:
			return uint64(layout.I8.Size)
		case semantic.IntType:
			return uint64(layout.Integer.Size)
		case semantic.UintType:
			return uint64(layout.Integer.Size)
		case semantic.SizeType:
			return uint64(layout.Size.Size)
		case semantic.CharType:
			return uint64(layout.Char.Size)
		case semantic.Int8Type:
			return uint64(layout.I8.Size)
		case semantic.Uint8Type:
			return uint64(layout.I8.Size)
		case semantic.Int16Type:
			return uint64(layout.I16.Size)
		case semantic.Uint16Type:
			return uint64(layout.I16.Size)
		case semantic.Int32Type:
			return uint64(layout.I32.Size)
		case semantic.Uint32Type:
			return uint64(layout.I32.Size)
		case semantic.Int64Type:
			return uint64(layout.I64.Size)
		case semantic.Uint64Type:
			return uint64(layout.I64.Size)
		case semantic.Float32Type:
			return uint64(layout.F32.Size)
		case semantic.Float64Type:
			return uint64(layout.F64.Size)
		}
	case *semantic.Enum:
		return t.SizeOf(layout, ty.NumberType)
	case *semantic.StaticArray:
		return uint64(ty.Size) * t.StrideOf(layout, ty.ValueType)
	case *semantic.Pointer:
		return uint64(layout.Pointer.Size)
	case *semantic.Class:
		size := uint64(0)
		for _, f := range ty.Fields {
			fieldSize := t.StrideOf(layout, f.Type)
			fieldAlignment := t.AlignOf(layout, f.Type)
			size = (size + fieldAlignment - 1) & ^(fieldAlignment - 1)
			size += fieldSize
		}
		return size
	}

	fail("Cannot determine the size for %T %v", ty, ty)
	return 1
}

// StrideOf returns the number of bytes per element when held in an array
// for the given memory layout.
func (t *Types) StrideOf(layout *device.MemoryLayout, ty semantic.Type) uint64 {
	alignment := t.AlignOf(layout, ty)
	size := t.SizeOf(layout, ty)
	return (size + alignment - 1) & ^(alignment - 1)
}

func (t *Types) basic(ty semantic.Type) codegen.Type {
	switch ty := ty.(type) {
	case *semantic.Builtin:
		switch ty {
		case semantic.VoidType:
			return t.Void
		case semantic.BoolType:
			return t.Bool
		case semantic.Int8Type:
			return t.Int8
		case semantic.Int16Type:
			return t.Int16
		case semantic.Int32Type:
			return t.Int32
		case semantic.Int64Type:
			return t.Int64
		case semantic.Uint8Type:
			return t.Uint8
		case semantic.Uint16Type:
			return t.Uint16
		case semantic.Uint32Type:
			return t.Uint32
		case semantic.Uint64Type:
			return t.Uint64
		case semantic.Float32Type:
			return t.Float32
		case semantic.Float64Type:
			return t.Float64
		case semantic.StringType:
			return t.StrPtr
		case semantic.CharType:
			return t.Uint8 // TODO: dynamic length
		default:
			fail("Unsupported builtin type %v", ty.Name())
			return nil
		}
	case *semantic.Enum:
		return t.basic(ty.NumberType)
	case *semantic.Slice:
		return t.Sli
	default:
		fail("Unsupported basic type %v (%T)", ty.Name(), ty)
		return nil
	}
}
