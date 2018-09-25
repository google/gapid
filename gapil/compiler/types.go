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
	Ctx             *codegen.Struct                        // context_t
	CtxPtr          codegen.Type                           // context_t*
	Pool            codegen.Type                           // pool_t
	PoolPtr         codegen.Type                           // pool_t*
	Sli             codegen.Type                           // slice_t
	Str             *codegen.Struct                        // string_t
	StrPtr          codegen.Type                           // string_t*
	Arena           *codegen.Struct                        // arena_t
	ArenaPtr        codegen.Type                           // arena_t*
	Any             *codegen.Struct                        // gapil_any_t
	AnyPtr          codegen.Type                           // gapil_any_t*
	Msg             *codegen.Struct                        // gapil_msg_t
	MsgPtr          codegen.Type                           // gapil_msg_t*
	MsgArg          *codegen.Struct                        // gapil_msg_arg_t
	MsgArgPtr       codegen.Type                           // gapil_msg_arg_t*
	RTTI            *codegen.Struct                        // gapil_rtti
	Uint8Ptr        codegen.Type                           // uint8_t*
	VoidPtr         codegen.Type                           // void* (aliased of uint8_t*)
	VoidPtrPtr      codegen.Type                           // void** (aliased of uint8_t**)
	Globals         *codegen.Struct                        // API global variables structure.
	GlobalsPtr      codegen.Type                           // Pointer to Globals.
	Buf             codegen.Type                           // buffer_t
	BufPtr          codegen.Type                           // buffer_t*
	CmdParams       map[*semantic.Function]*codegen.Struct // struct holding all command parameters and return value.
	DataAccess      codegen.Type
	Maps            map[*semantic.Map]*MapInfo
	mapImpls        []mapImpl
	customCtxFields []ContextField
	target          map[semantic.Type]codegen.Type
	targetABI       *device.ABI
	storage         map[memLayoutKey]*StorageTypes
	CaptureTypes    *StorageTypes
	captureToTarget map[semantic.Type]*codegen.Function
	targetToCapture map[semantic.Type]*codegen.Function
	mangled         map[codegen.Type]mangling.Type
	rttis           map[semantic.Type]codegen.Global
}

type memLayoutKey string

func (c *C) declareTypes() {
	c.T.targetABI = c.Settings.TargetABI

	c.T.Types = c.M.Types
	c.T.Ctx = c.T.DeclareStruct("context")
	c.T.CtxPtr = c.T.Pointer(c.T.Ctx)
	c.T.Globals = c.T.DeclareStruct("globals")
	c.T.GlobalsPtr = c.T.Pointer(c.T.Globals)
	c.T.Pool = c.T.TypeOf(C.pool{})
	c.T.PoolPtr = c.T.Pointer(c.T.Pool)
	c.T.Sli = c.T.TypeOf(C.slice{})
	c.T.Str = c.T.TypeOf(C.string{}).(*codegen.Struct)
	c.T.StrPtr = c.T.Pointer(c.T.Str)
	c.T.Uint8Ptr = c.T.Pointer(c.T.Uint8)
	c.T.Any = c.T.TypeOf(C.gapil_any{}).(*codegen.Struct)
	c.T.AnyPtr = c.T.Pointer(c.T.Any)
	c.T.Msg = c.T.TypeOf(C.gapil_msg{}).(*codegen.Struct)
	c.T.MsgPtr = c.T.Pointer(c.T.Msg)
	c.T.MsgArg = c.T.TypeOf(C.gapil_msg_arg{}).(*codegen.Struct)
	c.T.MsgArgPtr = c.T.Pointer(c.T.MsgArg)
	c.T.RTTI = c.T.TypeOf(C.gapil_rtti{}).(*codegen.Struct)
	c.T.Arena = c.T.DeclareStruct("arena")
	c.T.ArenaPtr = c.T.Pointer(c.T.Arena)
	c.T.VoidPtr = c.T.Pointer(c.T.Void)
	c.T.VoidPtrPtr = c.T.Pointer(c.T.VoidPtr)
	c.T.Buf = c.T.TypeOf(C.buffer{})
	c.T.BufPtr = c.T.Pointer(c.T.Buf)
	c.T.Maps = map[*semantic.Map]*MapInfo{}
	c.T.CmdParams = map[*semantic.Function]*codegen.Struct{}
	c.T.DataAccess = c.T.Enum("gapil_data_access")
	c.T.target = map[semantic.Type]codegen.Type{}
	c.T.storage = map[memLayoutKey]*StorageTypes{}
	c.T.captureToTarget = map[semantic.Type]*codegen.Function{}
	c.T.targetToCapture = map[semantic.Type]*codegen.Function{}
	c.T.mangled = map[codegen.Type]mangling.Type{}
	c.T.rttis = map[semantic.Type]codegen.Global{}

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

		// Forward-declare all the command parameter structs.
		for _, f := range api.Functions {
			c.T.CmdParams[f] = c.T.DeclareStruct(f.Name() + "Params")
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

		// Build all the command parameter types.
		for _, f := range api.Functions {
			fields := make([]codegen.Field, 0, len(f.FullParameters)+1)
			fields = append(fields, codegen.Field{Name: semantic.BuiltinThreadGlobal.Name(), Type: c.T.Uint64})
			for _, p := range f.FullParameters {
				fields = append(fields, codegen.Field{Name: p.Name(), Type: c.T.Target(p.Type)})
			}
			c.T.CmdParams[f].SetBody(false, fields...)
		}
	}

	// Build all the map types.
	c.defineMapTypes()

	c.buildRefRels()

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

	// Build storage types for the capture's memory layout.
	c.T.CaptureTypes = c.StorageTypes(c.Settings.CaptureABI.MemoryLayout, "S_")

	if !c.Settings.CaptureABI.SameAs(c.T.targetABI) {
		// Declare conversion functions between target and capture types.
		for _, api := range c.APIs {
			for _, t := range api.Classes {
				if !semantic.IsStorageType(t) {
					continue
				}
				captureTypePtr := c.T.Pointer(c.T.Capture(t))
				targetTypePtr := c.T.Pointer(c.T.Target(t))

				copyToTarget := c.M.Function(c.T.Void, "S_"+t.Name()+"_copy_to_target", c.T.CtxPtr, captureTypePtr, targetTypePtr).
					LinkOnceODR().
					Inline()

				c.T.captureToTarget[t] = copyToTarget

				copyToCapture := c.M.Function(c.T.Void, "T_"+t.Name()+"_copy_to_capture", c.T.CtxPtr, targetTypePtr, captureTypePtr).
					LinkOnceODR().
					Inline()

				c.T.targetToCapture[t] = copyToCapture
			}
		}

		// Build conversion functions between target and capture types.
		for _, api := range c.APIs {
			for _, t := range api.Classes {
				if !semantic.IsStorageType(t) {
					continue
				}

				c.Build(c.T.captureToTarget[t], func(s *S) {
					src := s.Parameter(1).SetName("src")
					dst := s.Parameter(2).SetName("dst")
					for _, f := range t.Fields {
						firstElem := src.Index(0, f.Name()).LoadUnaligned()
						dst.Index(0, f.Name()).Store(c.castCaptureToTarget(s, f.Type, firstElem))
					}
				})

				c.Build(c.T.targetToCapture[t], func(s *S) {
					src := s.Parameter(1).SetName("src")
					dst := s.Parameter(2).SetName("dst")
					for _, f := range t.Fields {
						firstElem := src.Index(0, f.Name()).Load()
						dst.Index(0, f.Name()).StoreUnaligned(c.castTargetToCapture(s, f.Type, firstElem))
					}
				})
			}
		}
	}

	// Build all the map types.
	c.buildMapTypes()
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

// Capture returns the codegen type used to represent ty when stored in a
// capture's buffer.
func (t *Types) Capture(ty semantic.Type) codegen.Type {
	return t.CaptureTypes.Get(ty)
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
		case semantic.AnyType:
			return t.AnyPtr
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
		case semantic.MessageType:
			return t.MsgPtr
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

func (c *C) initialValue(s *S, t semantic.Type) *codegen.Value {
	t = semantic.Underlying(t)
	switch t {
	case semantic.StringType:
		return c.emptyString.Value(s.Builder)
	}
	switch t := t.(type) {
	case *semantic.Class:
		class := s.Undef(c.T.Target(t))
		for i, f := range t.Fields {
			var val *codegen.Value
			if f.Default != nil {
				val = c.expression(s, f.Default)
			} else {
				val = c.initialValue(s, f.Type)
			}
			c.reference(s, val, f.Type)
			class = class.Insert(i, val)
		}
		c.deferRelease(s, class, t)
		return class
	case *semantic.Map:
		mapInfo := c.T.Maps[t]
		m := c.Alloc(s, s.Scalar(uint64(1)), mapInfo.Type).SetName(t.String())
		m.Index(0, MapRefCount).Store(s.Scalar(uint32(1)))
		m.Index(0, MapArena).Store(s.Arena)
		m.Index(0, MapCount).Store(s.Scalar(uint64(0)))
		m.Index(0, MapCapacity).Store(s.Scalar(uint64(0)))
		m.Index(0, MapElements).Store(s.Zero(c.T.Pointer(mapInfo.Elements)))
		c.deferRelease(s, m, t)
		return m
	default:
		return s.Zero(c.T.Target(t))
	}
}

func (c *C) buildSlice(s *S, root, base, size, count, pool *codegen.Value) *codegen.Value {
	return s.Undef(c.T.Sli).
		Insert(SliceRoot, root).
		Insert(SliceBase, base).
		Insert(SliceSize, size).
		Insert(SliceCount, count).
		Insert(SlicePool, pool)
}
