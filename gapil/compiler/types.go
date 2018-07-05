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
	Ctx             *codegen.Struct                     // context_t
	CtxPtr          codegen.Type                        // context_t*
	Pool            codegen.Type                        // pool_t
	PoolPtr         codegen.Type                        // pool_t*
	Sli             codegen.Type                        // slice_t
	Str             *codegen.Struct                     // string_t
	StrPtr          codegen.Type                        // string_t*
	Arena           *codegen.Struct                     // arena_t
	ArenaPtr        codegen.Type                        // arena_t*
	Uint8Ptr        codegen.Type                        // uint8_t*
	VoidPtr         codegen.Type                        // void* (aliased of uint8_t*)
	Globals         *codegen.Struct                     // API global variables structure.
	GlobalsPtr      codegen.Type                        // Pointer to Globals.
	Buf             codegen.Type                        // buffer_t
	BufPtr          codegen.Type                        // buffer_t*
	CmdParams       map[*semantic.Function]codegen.Type // struct holding all command parameters and return value.
	DataAccess      codegen.Type
	Maps            map[*semantic.Map]*MapInfo
	mapImpls        []mapImpl
	customCtxFields []ContextField
	target          map[semantic.Type]codegen.Type
	capture         map[semantic.Type]codegen.Type
	targetToCapture map[semantic.Type]*codegen.Function
	captureToTarget map[semantic.Type]*codegen.Function
	mangled         map[codegen.Type]mangling.Type
	targetABI       *device.ABI
	captureABI      *device.ABI
}

func (c *C) declareCaptureTypes() {
	for _, t := range c.API.Classes {
		if semantic.IsStorageType(t) {
			if c.T.captureABI == c.T.targetABI {
				c.T.capture[t] = c.T.target[t]
			} else {
				c.T.capture[t] = c.T.DeclarePackedStruct("S_" + t.Name())
			}
		}
	}
}

func (c *C) buildCaptureTypes() {
	if c.T.captureABI == c.T.targetABI {
		return
	}
	for _, t := range c.API.Classes {
		if semantic.IsStorageType(t) {
			offset := int32(0)
			fields := make([]codegen.Field, 0, len(t.Fields))
			dummyFields := 0
			for _, f := range t.Fields {
				size := c.T.CaptureAllocaSize(f.Type)
				alignment := c.T.CaptureABIAlignment(f.Type)
				newOffset := (offset + (alignment - 1)) & ^(alignment - 1)
				if newOffset != offset {
					nm := fmt.Sprintf("__dummy%d", dummyFields)
					dummyFields++
					fields = append(fields, codegen.Field{Name: nm, Type: c.T.Array(c.T.Capture(semantic.Uint8Type), int(newOffset-offset))})
				}
				offset = newOffset + size
				fields = append(fields, codegen.Field{Name: f.Name(), Type: c.T.Capture(f.Type)})
			}
			totalSize := c.T.CaptureAllocaSize(t)
			if totalSize != offset {
				nm := fmt.Sprintf("__dummy%d", dummyFields)
				fields = append(fields, codegen.Field{Name: nm, Type: c.T.Array(c.T.Capture(semantic.Uint8Type), int(totalSize-offset))})
			}

			c.T.capture[t].(*codegen.Struct).SetBody(true, fields...)
		}
	}
}

func (c *C) declareTypes() {
	c.T.captureABI = c.Settings.CaptureABI
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
	c.T.Arena = c.T.DeclareStruct("arena")
	c.T.ArenaPtr = c.T.Pointer(c.T.Arena)
	c.T.VoidPtr = c.T.Pointer(c.T.Void)
	c.T.Buf = c.T.TypeOf(C.buffer{})
	c.T.BufPtr = c.T.Pointer(c.T.Buf)
	c.T.Maps = map[*semantic.Map]*MapInfo{}
	c.T.CmdParams = map[*semantic.Function]codegen.Type{}
	c.T.DataAccess = c.T.Enum("gapil_data_access")
	c.T.target = map[semantic.Type]codegen.Type{}
	c.T.capture = map[semantic.Type]codegen.Type{}
	c.T.targetToCapture = map[semantic.Type]*codegen.Function{}
	c.T.captureToTarget = map[semantic.Type]*codegen.Function{}
	c.T.mangled = map[codegen.Type]mangling.Type{}

	// Forward-declare all the class types.
	for _, t := range c.API.Classes {
		cgTy := c.T.DeclareStruct("T_" + t.Name())
		c.T.target[t] = cgTy
	}

	// Forward-declare all the reference types.
	for _, t := range c.API.References {
		cgTy := c.T.DeclareStruct(t.Name())
		c.T.target[t] = c.T.Pointer(cgTy)
	}

	// Forward-declare all the map types.
	for _, t := range c.API.Maps {
		cgTy := c.T.DeclareStruct(t.Name())
		mapStrTy := cgTy
		mapPtrTy := c.T.Pointer(mapStrTy)
		c.T.target[t] = mapPtrTy
	}

	// Declare all the slice types.
	for _, t := range c.API.Slices {
		c.T.target[t] = c.T.Sli
	}

	// Declare all the command parameter structs.
	for _, f := range c.API.Functions {
		fields := make([]codegen.Field, 0, len(f.FullParameters)+1)
		fields = append(fields, codegen.Field{Name: semantic.BuiltinThreadGlobal.Name(), Type: c.T.Uint64})
		for _, p := range f.FullParameters {
			fields = append(fields, codegen.Field{Name: p.Name(), Type: c.T.Target(p.Type)})
		}
		c.T.CmdParams[f] = c.T.Struct(f.Name()+"Params", fields...)
	}

	c.declareCaptureTypes()

	c.declareMangling()

	c.declareRefRels()

	c.declareContextType()
}

func (c *C) declareMangling() {
	// Declare the mangled types
	for _, t := range c.API.Classes {
		c.T.mangled[c.T.Target(t)] = &mangling.Class{
			Parent: c.Root,
			Name:   t.Name(),
		}
	}
	for _, t := range c.API.References {
		refTy := c.T.Target(t).(codegen.Pointer).Element
		c.T.mangled[refTy] = &mangling.Class{
			Parent: c.Root,
			Name:   "Ref",
		}
	}
	for _, t := range c.API.Maps {
		mapTy := c.T.Target(t).(codegen.Pointer).Element
		c.T.mangled[mapTy] = &mangling.Class{
			Parent: c.Root,
			Name:   "Map",
		}
	}

	// Add template parameters
	for _, t := range c.API.References {
		refTy := c.T.Target(t).(codegen.Pointer).Element
		c.T.mangled[refTy].(*mangling.Class).TemplateArgs = []mangling.Type{
			c.Mangle(c.T.Target(t.To)),
		}
	}
	for _, t := range c.API.Maps {
		mapTy := c.T.Target(t).(codegen.Pointer).Element
		c.T.mangled[mapTy].(*mangling.Class).TemplateArgs = []mangling.Type{
			c.Mangle(c.T.Target(t.KeyType)),
			c.Mangle(c.T.Target(t.ValueType)),
		}
	}
}

func (c *C) buildTypes() {
	// Build all the class types.
	for _, t := range c.API.Classes {
		fields := make([]codegen.Field, len(t.Fields))
		for i, f := range t.Fields {
			fields[i] = codegen.Field{Name: f.Name(), Type: c.T.Target(f.Type)}
		}
		c.T.target[t].(*codegen.Struct).SetBody(false, fields...)
	}

	c.buildCaptureTypes()

	// Build all the reference types.
	for _, t := range c.API.References {
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

	// Build all the map types.
	c.defineMapTypes()

	c.buildRefRels()

	globalsFields := make([]codegen.Field, len(c.API.Globals))
	for i, g := range c.API.Globals {
		globalsFields[i] = codegen.Field{Name: g.Name(), Type: c.T.Target(g.Type)}
	}
	c.T.Globals.SetBody(false, globalsFields...)
	if c.Settings.CaptureABI != c.T.targetABI {
		for _, t := range c.API.Classes {
			if semantic.IsStorageType(t) {
				captureTypePtr := c.T.Pointer(c.T.Capture(t))
				targetTypePtr := c.T.Pointer(c.T.Target(t))

				copyToTarget := c.M.Function(c.T.Void, "S_"+t.Name()+"_copy_to_target", c.T.CtxPtr, captureTypePtr, targetTypePtr).
					LinkOnceODR().
					Inline()

				c.T.captureToTarget[t] = copyToTarget
				c.Build(copyToTarget, func(s *S) {
					src := s.Parameter(1).SetName("src")
					dst := s.Parameter(2).SetName("dst")
					for _, f := range t.Fields {
						firstElem := src.Index(0, f.Name()).LoadUnaligned()
						dst.Index(0, f.Name()).Store(c.castCaptureToTarget(s, f.Type, firstElem))
					}
				})

				copyToCapture := c.M.Function(c.T.Void, "T_"+t.Name()+"_copy_to_capture", c.T.CtxPtr, targetTypePtr, captureTypePtr).
					LinkOnceODR().
					Inline()

				c.T.targetToCapture[t] = copyToCapture
				c.Build(copyToCapture, func(s *S) {
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

// Capture returns the codegen type used to store ty in a buffer.
func (t *Types) Capture(ty semantic.Type) codegen.Type {
	layout := t.captureABI.MemoryLayout
	ty = semantic.Underlying(ty)
	switch ty := ty.(type) {
	case *semantic.Builtin:
		switch ty {
		case semantic.IntType:
			return t.basic(semantic.Integer(layout.Integer.Size))
		case semantic.SizeType:
			return t.basic(semantic.UnsignedInteger(layout.Size.Size))
		}
	case *semantic.StaticArray:
		return t.Array(t.Capture(ty.ValueType), int(ty.Size))
	case *semantic.Pointer:
		return t.basic(semantic.UnsignedInteger(layout.Pointer.Size))
	case *semantic.Class:
		if out, ok := t.capture[ty]; ok {
			return out
		}
		fail("Capture class not registered: '%v'", ty.Name())
	case *semantic.Slice, *semantic.Reference, *semantic.Map:
		fail("Cannot store type '%v' (%T) in buffers", ty.Name(), t)
	}
	return t.basic(ty)
}

func (t *Types) basic(ty semantic.Type) codegen.Type {
	switch ty := ty.(type) {
	case *semantic.Builtin:
		switch ty {
		case semantic.AnyType:
			return t.Uint8Ptr // TODO
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
			return t.Uint8 // TODO: Messages
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

// CaptureABIAlignment returns the alignment of this type in bytes when stored.
func (t *Types) CaptureABIAlignment(ty semantic.Type) int32 {
	layout := t.captureABI.MemoryLayout
	switch ty := semantic.Underlying(ty).(type) {
	case *semantic.Builtin:
		switch ty {
		case semantic.BoolType:
			return int32(layout.I8.Alignment)
		case semantic.IntType:
			return int32(layout.Integer.Alignment)
		case semantic.UintType:
			return int32(layout.Integer.Alignment)
		case semantic.SizeType:
			return int32(layout.Size.Alignment)
		case semantic.CharType:
			return int32(layout.Char.Alignment)
		case semantic.Int8Type:
			return int32(layout.I8.Alignment)
		case semantic.Uint8Type:
			return int32(layout.I8.Alignment)
		case semantic.Int16Type:
			return int32(layout.I16.Alignment)
		case semantic.Uint16Type:
			return int32(layout.I16.Alignment)
		case semantic.Int32Type:
			return int32(layout.I32.Alignment)
		case semantic.Uint32Type:
			return int32(layout.I32.Alignment)
		case semantic.Int64Type:
			return int32(layout.I64.Alignment)
		case semantic.Uint64Type:
			return int32(layout.I64.Alignment)
		case semantic.Float32Type:
			return int32(layout.F32.Alignment)
		case semantic.Float64Type:
			return int32(layout.F64.Alignment)
		}
	case *semantic.StaticArray:
		return t.CaptureABIAlignment(ty.ValueType)
	case *semantic.Pointer:
		return layout.Pointer.Alignment
	case *semantic.Class:
		alignment := int32(1)
		for _, f := range ty.Fields {
			a := t.CaptureABIAlignment(f.Type)
			if alignment < a {
				alignment = a
			}
		}
		return alignment
	}
	fail("Cannot determine the capture alignemnt for %T %v", ty, ty)
	return 1
}

// CaptureSize returns the number of bytes needed to store this type.
func (t *Types) CaptureSize(ty semantic.Type) int32 {
	layout := t.captureABI.MemoryLayout
	switch ty := semantic.Underlying(ty).(type) {
	case *semantic.Builtin:
		switch ty {
		case semantic.BoolType:
			return int32(layout.I8.Size)
		case semantic.IntType:
			return int32(layout.Integer.Size)
		case semantic.UintType:
			return int32(layout.Integer.Size)
		case semantic.SizeType:
			return int32(layout.Size.Size)
		case semantic.CharType:
			return int32(layout.Char.Size)
		case semantic.Int8Type:
			return int32(layout.I8.Size)
		case semantic.Uint8Type:
			return int32(layout.I8.Size)
		case semantic.Int16Type:
			return int32(layout.I16.Size)
		case semantic.Uint16Type:
			return int32(layout.I16.Size)
		case semantic.Int32Type:
			return int32(layout.I32.Size)
		case semantic.Uint32Type:
			return int32(layout.I32.Size)
		case semantic.Int64Type:
			return int32(layout.I64.Size)
		case semantic.Uint64Type:
			return int32(layout.I64.Size)
		case semantic.Float32Type:
			return int32(layout.F32.Size)
		case semantic.Float64Type:
			return int32(layout.F64.Size)
		}
	case *semantic.StaticArray:
		return int32(ty.Size) * t.CaptureAllocaSize(ty.ValueType)
	case *semantic.Pointer:
		return layout.Pointer.Size
	case *semantic.Class:
		size := int32(0)
		for _, f := range ty.Fields {
			fieldSize := t.CaptureAllocaSize(f.Type)
			fieldAlignment := t.CaptureABIAlignment(f.Type)
			size = (size + fieldAlignment - 1) & ^(fieldAlignment - 1)
			size += fieldSize
		}
		return size
	}

	fail("Cannot determine the capture size for %T %v", ty, ty)
	return 1
}

// CaptureAllocaSize returns the number of bytes per object if you were to
// store two next to each other in memory.
func (t *Types) CaptureAllocaSize(ty semantic.Type) int32 {
	alignment := t.CaptureABIAlignment(ty)
	size := t.CaptureSize(ty)
	return (size + alignment - 1) & ^(alignment - 1)
}

func (c *C) initialValue(s *S, t semantic.Type) *codegen.Value {
	switch t {
	case semantic.StringType:
		return c.emptyString.Value(s.Builder)
	}
	switch t := t.(type) {
	case *semantic.Class:
		class := s.Undef(c.T.Target(t))
		for i, f := range t.Fields {
			if f.Default != nil {
				class = class.Insert(i, c.expression(s, f.Default))
			} else {
				class = class.Insert(i, c.initialValue(s, f.Type))
			}
		}
		return class
	case *semantic.Map:
		mapInfo := c.T.Maps[t]
		mapPtr := c.Alloc(s, s.Scalar(uint64(1)), mapInfo.Type)
		mapPtr.Index(0, MapRefCount).Store(s.Scalar(uint32(1)))
		mapPtr.Index(0, MapArena).Store(s.Arena)
		mapPtr.Index(0, MapCount).Store(s.Scalar(uint64(0)))
		mapPtr.Index(0, MapCapacity).Store(s.Scalar(uint64(0)))
		mapPtr.Index(0, MapElements).Store(s.Zero(c.T.Pointer(mapInfo.Elements)))
		c.deferRelease(s, mapPtr, t)
		return mapPtr
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
