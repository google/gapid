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
	"github.com/google/gapid/gapil/semantic"
)

//#include "gapil/compiler/cc/builtins.h"
import "C"

type types struct {
	codegen.Types
	ctx             codegen.Type
	ctxPtr          codegen.Type
	globals         *codegen.Struct
	pool            codegen.Type
	sli             codegen.Type
	str             codegen.Type
	strPtr          codegen.Type
	u8Ptr           codegen.Type
	voidPtr         codegen.Type
	target          map[semantic.Type]codegen.Type
	storage         map[semantic.Type]codegen.Type
	targetToStorage map[semantic.Type]codegen.Function
	storageToTarget map[semantic.Type]codegen.Function
	maps            map[*semantic.Map]*MapInfo
}

// isStorageType returns true if ty can be used as a storage type.
func isStorageType(ty semantic.Type) bool {
	switch ty := ty.(type) {
	case *semantic.Builtin:
		switch ty {
		case semantic.StringType,
			semantic.AnyType,
			semantic.MessageType:
			return false
		default:
			return true
		}
	case *semantic.Pseudonym:
		return isStorageType(ty.To)
	case *semantic.Pointer:
		return isStorageType(ty.To)
	case *semantic.Class:
		for _, f := range ty.Fields {
			if !isStorageType(f.Type) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (c *compiler) declareStorageTypes(api *semantic.API) {
	for _, t := range api.Classes {
		if isStorageType(t) {
			if c.settings.StorageABI == c.settings.TargetABI {
				c.ty.storage[t] = c.ty.target[t]
			} else {
				c.ty.storage[t] = c.ty.DeclarePackedStruct("S_" + t.Name())
			}
		}
	}
}

func (c *compiler) buildStorageTypes(api *semantic.API) {
	if c.settings.StorageABI == c.settings.TargetABI {
		return
	}
	for _, t := range api.Classes {
		if isStorageType(t) {
			offset := int32(0)
			fields := make([]codegen.Field, 0, len(t.Fields))
			dummyFields := 0
			for _, f := range t.Fields {
				size := c.storageAllocaSize(f.Type)
				alignment := c.storageABIAlignment(f.Type)
				newOffset := (offset + (alignment - 1)) & ^(alignment - 1)
				if newOffset != offset {
					nm := fmt.Sprintf("__dummy%d", dummyFields)
					dummyFields++
					fields = append(fields, codegen.Field{Name: nm, Type: c.ty.Array(c.storageType(semantic.Uint8Type), int(newOffset-offset))})
				}
				offset = newOffset + size
				fields = append(fields, codegen.Field{Name: f.Name(), Type: c.storageType(f.Type)})
			}
			totalSize := c.storageAllocaSize(t)
			if totalSize != offset {
				nm := fmt.Sprintf("__dummy%d", dummyFields)
				fields = append(fields, codegen.Field{Name: nm, Type: c.ty.Array(c.storageType(semantic.Uint8Type), int(totalSize-offset))})
			}

			c.ty.storage[t].(*codegen.Struct).SetBody(true, fields...)
		}
	}
}

func (c *compiler) declareTypes(api *semantic.API) {
	c.ty.Types = c.module.Types
	c.ty.globals = c.ty.DeclareStruct("globals")
	c.ty.pool = c.ty.TypeOf(C.pool{})
	c.ty.sli = c.ty.TypeOf(C.slice{})
	c.ty.str = c.ty.TypeOf(C.string{})
	c.ty.strPtr = c.ty.Pointer(c.ty.str)
	c.ty.u8Ptr = c.ty.Pointer(c.ty.Uint8)
	c.ty.voidPtr = c.ty.Pointer(c.ty.Void)
	c.ty.ctx = c.ty.TypeOf(C.context{})
	c.ty.ctxPtr = c.ty.Pointer(c.ty.ctx)
	c.ty.target = map[semantic.Type]codegen.Type{}
	c.ty.storage = map[semantic.Type]codegen.Type{}
	c.ty.targetToStorage = map[semantic.Type]codegen.Function{}
	c.ty.storageToTarget = map[semantic.Type]codegen.Function{}
	c.ty.maps = map[*semantic.Map]*MapInfo{}

	// Forward-declare all the class types.
	for _, t := range api.Classes {
		c.ty.target[t] = c.ty.DeclareStruct("T_" + t.Name())
	}

	// Forward-declare all the reference types.
	for _, t := range api.References {
		c.ty.target[t] = c.ty.Pointer(c.ty.DeclareStruct(t.Name()))
	}
	// Forward-declare all the map types.
	for _, t := range api.Maps {
		c.ty.target[t] = c.ty.Pointer(c.ty.DeclareStruct(t.Name()))
	}
	// Declare all the slice types.
	for _, t := range api.Slices {
		c.ty.target[t] = c.ty.sli
	}

	c.declareStorageTypes(api)

	c.declareRefRels()
}

func (c *compiler) buildTypes(api *semantic.API) {

	// Build all the class types.
	for _, t := range api.Classes {
		fields := make([]codegen.Field, len(t.Fields))
		for i, f := range t.Fields {
			fields[i] = codegen.Field{Name: f.Name(), Type: c.targetType(f.Type)}
		}
		c.ty.target[t].(*codegen.Struct).SetBody(false, fields...)
	}

	c.buildStorageTypes(api)

	// Build all the reference types.
	for _, t := range api.References {
		// struct ref!T {
		//      uint32_t ref_count;
		//      T        value;
		// }
		ptr := c.ty.target[t].(codegen.Pointer)
		str := ptr.Element.(*codegen.Struct)
		str.SetBody(false,
			codegen.Field{Name: refRefCount, Type: c.ty.Uint32},
			codegen.Field{Name: refValue, Type: c.targetType(t.To)},
		)
	}

	// Build all the map types.
	for _, t := range api.Maps {
		c.defineMapType(t)
	}

	c.buildRefRels()

	globalsFields := make([]codegen.Field, len(api.Globals))
	for i, g := range api.Globals {
		globalsFields[i] = codegen.Field{Name: g.Name(), Type: c.targetType(g.Type)}
	}
	c.ty.globals.SetBody(false, globalsFields...)
	if c.settings.StorageABI != c.settings.TargetABI {
		for _, t := range api.Classes {
			if isStorageType(t) {
				storageTypePtr := c.ty.Pointer(c.storageType(t))
				targetTypePtr := c.ty.Pointer(c.targetType(t))

				copyToTarget := c.module.Function(c.ty.Void, "S_"+t.Name()+"_copy_to_target", c.ty.ctxPtr, storageTypePtr, targetTypePtr)
				copyToTarget.MakeInline()
				c.ty.storageToTarget[t] = copyToTarget
				c.build(copyToTarget, func(s *scope) {
					src := s.Parameter(1).SetName("src")
					dst := s.Parameter(2).SetName("dst")
					for _, f := range t.Fields {
						firstElem := src.Index(0, f.Name()).LoadUnaligned()
						dst.Index(0, f.Name()).Store(c.castStorageToTarget(s, f.Type, firstElem))
					}
				})

				copyToStorage := c.module.Function(c.ty.Void, "T_"+t.Name()+"_copy_to_storage", c.ty.ctxPtr, targetTypePtr, storageTypePtr)
				copyToStorage.MakeInline()
				c.ty.targetToStorage[t] = copyToStorage
				c.build(copyToStorage, func(s *scope) {
					src := s.Parameter(1).SetName("src")
					dst := s.Parameter(2).SetName("dst")
					for _, f := range t.Fields {
						firstElem := src.Index(0, f.Name()).Load()
						dst.Index(0, f.Name()).StoreUnaligned(c.castTargetToStorage(s, f.Type, firstElem))
					}
				})
			}
		}
	}
}

// targetType returns the codegen type used to represent t in the
// target-preferred form.
func (c *compiler) targetType(t semantic.Type) codegen.Type {
	layout := c.settings.TargetABI.MemoryLayout
	t = semantic.Underlying(t)
	switch t := t.(type) {
	case *semantic.Builtin:
		switch t {
		case semantic.IntType:
			return c.basicType(c.intType(layout.Integer.Size))
		case semantic.SizeType:
			return c.basicType(c.uintType(layout.Size.Size))
		}
	case *semantic.StaticArray:
		return c.ty.Array(c.targetType(t.ValueType), int(t.Size))
	case *semantic.Slice:
		return c.ty.sli
	case *semantic.Pointer:
		return c.ty.Uint64
	case *semantic.Class, *semantic.Reference, *semantic.Map:
		if out, ok := c.ty.target[t]; ok {
			return out
		}
		fail("Target type not registered: '%v' (%T)", t.Name(), t)
	}
	return c.basicType(t)
}

// storageType returns the codegen type used to store t in a buffer.
func (c *compiler) storageType(t semantic.Type) codegen.Type {
	layout := c.settings.StorageABI.MemoryLayout
	t = semantic.Underlying(t)
	switch t := t.(type) {
	case *semantic.Builtin:
		switch t {
		case semantic.IntType:
			return c.basicType(c.intType(layout.Integer.Size))
		case semantic.SizeType:
			return c.basicType(c.uintType(layout.Size.Size))
		}
	case *semantic.StaticArray:
		return c.ty.Array(c.storageType(t.ValueType), int(t.Size))
	case *semantic.Pointer:
		return c.basicType(c.uintType(layout.Pointer.Size))
	case *semantic.Class:
		if out, ok := c.ty.storage[t]; ok {
			return out
		}
		fail("Storage class not registered: '%v'", t.Name())
	case *semantic.Slice, *semantic.Reference, *semantic.Map:
		fail("Cannot store type '%v' (%T) in buffers", t.Name(), t)
	}
	return c.basicType(t)
}

func (c *compiler) basicType(t semantic.Type) (out codegen.Type) {
	switch t := t.(type) {
	case *semantic.Builtin:
		switch t {
		case semantic.AnyType:
			return c.ty.u8Ptr // TODO
		case semantic.VoidType:
			return c.ty.Void
		case semantic.BoolType:
			return c.ty.Bool
		case semantic.Int8Type:
			return c.ty.Int8
		case semantic.Int16Type:
			return c.ty.Int16
		case semantic.Int32Type:
			return c.ty.Int32
		case semantic.Int64Type:
			return c.ty.Int64
		case semantic.Uint8Type:
			return c.ty.Uint8
		case semantic.Uint16Type:
			return c.ty.Uint16
		case semantic.Uint32Type:
			return c.ty.Uint32
		case semantic.Uint64Type:
			return c.ty.Uint64
		case semantic.Float32Type:
			return c.ty.Float32
		case semantic.Float64Type:
			return c.ty.Float64
		case semantic.StringType:
			return c.ty.strPtr
		case semantic.CharType:
			return c.ty.Uint8 // TODO: dynamic length
		case semantic.MessageType:
			return c.ty.Uint8 // TODO: Messages
		default:
			fail("Unsupported builtin type %v", t.Name())
			return nil
		}
	case *semantic.Enum:
		return c.ty.Uint32 // TODO: This right?
	case *semantic.Slice:
		return c.ty.sli
	default:
		fail("Unsupported basic type %v (%T)", t.Name(), t)
		return nil
	}
}

// storageABIAlignment is the alignment of this type when stored
func (c *compiler) storageABIAlignment(t semantic.Type) int32 {
	layout := c.settings.StorageABI.MemoryLayout
	switch t := semantic.Underlying(t).(type) {
	case *semantic.Builtin:
		switch t {
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
		default:
			fail("Cannot determine the storage alignemnt for %T", t)
			return 1
		}
	case *semantic.StaticArray:
		return c.storageABIAlignment(t.ValueType)
	case *semantic.Pointer:
		return layout.Pointer.Alignment
	case *semantic.Class:
		alignment := int32(1)
		for _, f := range t.Fields {
			a := c.storageABIAlignment(f.Type)
			if alignment < a {
				alignment = a
			}
		}
		return alignment
	default:
		fail("Cannot determine the storage alignemnt for %T", t)
		return 1
	}
}

// storageSize is the number of bytes needed to store this type
func (c *compiler) storageSize(t semantic.Type) int32 {
	layout := c.settings.StorageABI.MemoryLayout
	switch t := semantic.Underlying(t).(type) {
	case *semantic.Builtin:
		switch t {
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
		default:
			fail("Cannot determine the storage size for %T, %v", t, t)
			return 1
		}
	case *semantic.StaticArray:
		return int32(t.Size) * c.storageAllocaSize(t.ValueType)
	case *semantic.Pointer:
		return layout.Pointer.Size
	case *semantic.Class:
		size := int32(0)
		for _, f := range t.Fields {
			fieldSize := c.storageAllocaSize(f.Type)
			fieldAlignment := c.storageABIAlignment(f.Type)
			size = (size + fieldAlignment - 1) & ^(fieldAlignment - 1)
			size += fieldSize
		}
		return size
	default:
		fail("Cannot determine the storage size of %T", t)
		return 1
	}
}

// storageAllocaSize is the number of bytes per object if you were to
// store two next to each other in memory
func (c *compiler) storageAllocaSize(t semantic.Type) int32 {
	alignment := c.storageABIAlignment(t)
	size := c.storageSize(t)
	return (size + alignment - 1) & ^(alignment - 1)
}

func (c *compiler) initialValue(s *scope, t semantic.Type) *codegen.Value {
	switch t {
	case semantic.StringType:
		return s.ctx.Index(0, contextEmptyString).Load()
	}
	switch t := t.(type) {
	case *semantic.Class:
		class := s.Undef(c.targetType(t))
		for i, f := range t.Fields {
			if f.Default != nil {
				class = class.Insert(i, c.expression(s, f.Default))
			} else {
				class = class.Insert(i, c.initialValue(s, f.Type))
			}
		}
		return class
	case *semantic.Map:
		mapInfo := c.ty.maps[t]
		mapPtr := c.alloc(s, s.Scalar(uint64(1)), mapInfo.Type)
		mapPtr.Index(0, mapRefCount).Store(s.Scalar(uint32(1)))
		mapPtr.Index(0, mapCount).Store(s.Scalar(uint64(0)))
		mapPtr.Index(0, mapCapacity).Store(s.Scalar(uint64(0)))
		mapPtr.Index(0, mapElements).Store(s.Zero(c.ty.Pointer(mapInfo.Elements)))
		c.deferRelease(s, mapPtr, t)
		return mapPtr
	default:
		return s.Zero(c.targetType(t))
	}
}

func (c *compiler) buildSlice(s *scope, root, base, size, pool *codegen.Value) *codegen.Value {
	slice := s.Undef(c.ty.sli)
	slice = slice.Insert(sliceRoot, root)
	slice = slice.Insert(sliceBase, base)
	slice = slice.Insert(sliceSize, size)
	slice = slice.Insert(slicePool, pool)
	return slice
}

func (c *compiler) intType(bytes int32) (out semantic.Type) {
	switch bytes {
	case 1:
		return semantic.Int8Type
	case 2:
		return semantic.Int16Type
	case 4:
		return semantic.Int32Type
	case 8:
		return semantic.Int64Type
	default:
		fail("Unexpected target integer size %v", bytes)
		return nil
	}
}

func (c *compiler) uintType(bytes int32) (out semantic.Type) {
	switch bytes {
	case 1:
		return semantic.Uint8Type
	case 2:
		return semantic.Uint16Type
	case 4:
		return semantic.Uint32Type
	case 8:
		return semantic.Uint64Type
	default:
		fail("Unexpected target integer size %v", bytes)
		return nil
	}
}
