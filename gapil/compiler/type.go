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
	ctx     codegen.Type
	ctxPtr  codegen.Type
	globals *codegen.Struct
	pool    codegen.Type
	sli     codegen.Type
	str     codegen.Type
	strPtr  codegen.Type
	u8Ptr   codegen.Type
	voidPtr codegen.Type
	target  map[semantic.Type]codegen.Type
	storage map[semantic.Type]codegen.Type
	maps    map[*semantic.Map]*MapInfo
}

// isStorageType returns true if ty can be used as a storage type.
func isStorageType(ty semantic.Type) bool {
	switch ty := ty.(type) {
	case *semantic.Builtin:
		return true
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
	c.ty.maps = map[*semantic.Map]*MapInfo{}

	usages := []struct {
		prefix  string
		get     func(t semantic.Type) codegen.Type
		out     map[semantic.Type]codegen.Type
		storage bool
	}{
		{"T_", c.targetType, c.ty.target, false},
		{"S_", c.storageType, c.ty.storage, true},
	}

	for _, usage := range usages {
		// Forward-declare all the class types.
		for _, t := range api.Classes {
			if !usage.storage || isStorageType(t) {
				usage.out[t] = c.ty.DeclareStruct(usage.prefix + t.Name())
			}
		}
		if !usage.storage { // references and maps cannot be put in storage
			// Forward-declare all the reference types.
			for _, t := range api.References {
				usage.out[t] = c.ty.Pointer(c.ty.DeclareStruct(t.Name()))
			}
			// Forward-declare all the map types.
			for _, t := range api.Maps {
				usage.out[t] = c.ty.Pointer(c.ty.DeclareStruct(t.Name()))
			}
			// Declare all the slice types.
			for _, t := range api.Slices {
				usage.out[t] = c.ty.sli
			}
		}
	}

	c.declareRefRels()
}

func (c *compiler) buildTypes(api *semantic.API) {
	usages := []struct {
		prefix  string
		get     func(t semantic.Type) codegen.Type
		out     map[semantic.Type]codegen.Type
		storage bool
	}{
		{"T_", c.targetType, c.ty.target, false},
		{"S_", c.storageType, c.ty.storage, true},
	}

	for _, usage := range usages {
		// Build all the class types.
		for _, t := range api.Classes {
			if !usage.storage || isStorageType(t) {
				fields := make([]codegen.Field, len(t.Fields))
				for i, f := range t.Fields {
					fields[i] = codegen.Field{Name: f.Name(), Type: usage.get(f.Type)}
				}
				usage.out[t].(*codegen.Struct).SetBody(false, fields...)
			}
		}

		if !usage.storage { // references and maps cannot be put in storage
			// Build all the reference types.
			for _, t := range api.References {
				// struct ref!T {
				//      uint32_t ref_count;
				//      T        value;
				// }
				ptr := usage.out[t].(codegen.Pointer)
				str := ptr.Element.(*codegen.Struct)
				str.SetBody(false,
					codegen.Field{Name: refRefCount, Type: c.ty.Uint32},
					codegen.Field{Name: refValue, Type: usage.get(t.To)},
				)
			}

			// Build all the map types.
			for _, t := range api.Maps {
				mapPtrTy := usage.out[t].(codegen.Pointer)
				mapStrTy := mapPtrTy.Element.(*codegen.Struct)
				keyTy := usage.get(t.KeyType)
				valTy := usage.get(t.ValueType)
				elTy := c.ty.Struct(fmt.Sprintf("%v…%v", keyTy.TypeName(), valTy.TypeName()),
					codegen.Field{Name: "k", Type: keyTy},
					codegen.Field{Name: "v", Type: valTy},
				)
				mapStrTy.SetBody(false,
					codegen.Field{Name: mapRefCount, Type: c.ty.Uint32},
					codegen.Field{Name: mapCount, Type: c.ty.Uint64},
					codegen.Field{Name: mapCapacity, Type: c.ty.Uint64},
					codegen.Field{Name: mapElements, Type: c.ty.Pointer(elTy)},
				)
				c.ty.maps[t] = &MapInfo{Type: mapStrTy, Elements: elTy, Key: keyTy, Val: valTy}

				c.buildMapType(t)
			}
		}
	}

	c.buildRefRels()

	globalsFields := make([]codegen.Field, len(api.Globals))
	for i, g := range api.Globals {
		globalsFields[i] = codegen.Field{Name: g.Name(), Type: c.targetType(g.Type)}
	}
	c.ty.globals.SetBody(false, globalsFields...)
}

// targetType returns the codegen type used to represent t in the
// target-preferred form.
func (c *compiler) targetType(t semantic.Type) codegen.Type {
	layout := c.settings.StorageABI.MemoryLayout
	switch t := semantic.Underlying(t).(type) {
	case *semantic.Builtin:
		switch t {
		case semantic.IntType:
			return c.basicType(c.intType(layout.Integer.Size))
		case semantic.SizeType:
			return c.basicType(c.intType(layout.Size.Size))
		}
	case *semantic.StaticArray:
		return c.ty.Array(c.targetType(t.ValueType), int(t.Size))
	case *semantic.Slice:
		return c.ty.sli
	case *semantic.Pseudonym:
		return c.targetType(t.To)
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
	switch t := semantic.Underlying(t).(type) {
	case *semantic.Builtin:
		switch t {
		case semantic.IntType:
			return c.basicType(c.intType(layout.Integer.Size))
		case semantic.SizeType:
			return c.basicType(c.intType(layout.Size.Size))
		}
	case *semantic.StaticArray:
		return c.ty.Array(c.storageType(t.ValueType), int(t.Size))
	case *semantic.Pseudonym:
		return c.storageType(t.To)
	case *semantic.Pointer:
		return c.ty.Pointer(c.storageType(t.To))
	case *semantic.Class:
		if out, ok := c.ty.target[t]; ok {
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

func (c *compiler) buildMapType(t *semantic.Map) {
	info, ok := c.ty.maps[t]
	if !ok {
		fail("Unknown map")
	}

	mapPtrTy := c.targetType(t)
	elTy, keyTy, valTy := info.Elements, info.Key, info.Val
	valPtrTy := c.ty.Pointer(valTy)

	contains := c.module.Function(c.ty.Bool, t.Name()+"•contains", c.ty.ctxPtr, mapPtrTy, keyTy)
	err(contains.Build(func(jb *codegen.Builder) {
		s := c.scope(jb)
		m := s.Parameter(1).SetName("map")
		k := s.Parameter(2).SetName("key")
		count := m.Index(0, mapCount).Load()
		elements := m.Index(0, mapElements).Load()
		s.ForN(count, func(it *codegen.Value) *codegen.Value {
			key := elements.Index(it, "k")
			found := c.equal(s, key.Load(), k)
			s.If(found, func() { s.Return(s.Scalar(true)) })
			return s.Not(found)
		})
		s.Return(s.Scalar(false))
	}))

	index := c.module.Function(valPtrTy, t.Name()+"•index", c.ty.ctxPtr, mapPtrTy, keyTy, c.ty.Bool)
	err(index.Build(func(jb *codegen.Builder) {
		s := c.scope(jb)
		m := s.Parameter(1).SetName("map")
		k := s.Parameter(2).SetName("key")
		addIfNotFound := s.Parameter(3).SetName("addIfNotFound")

		countPtr := m.Index(0, mapCount)
		capacityPtr := m.Index(0, mapCapacity)
		elementsPtr := m.Index(0, mapElements)
		count := countPtr.Load()
		capacity := capacityPtr.Load()
		elements := elementsPtr.Load()

		// Search for existing
		s.ForN(count, func(it *codegen.Value) *codegen.Value {
			found := c.equal(s, elements.Index(it, "k").Load(), k)
			s.If(found, func() { s.Return(elements.Index(it, "v")) })
			return nil
		})

		s.If(addIfNotFound, func() {
			space := s.Sub(capacity, count).SetName("space")
			s.If(s.Equal(space, s.Zero(space.Type())), func() {
				// Grow
				capacity := s.AddS(capacity, uint64(mapGrowBy))
				capacityPtr.Store(capacity)
				s.IfElse(elements.IsNull(), func() {
					elementsPtr.Store(c.alloc(s, capacity, elTy))
				}, /* else */ func() {
					elementsPtr.Store(c.realloc(s, elements, capacity, elTy))
				})
			})

			count := countPtr.Load()
			elements := elementsPtr.Load()
			elements.Index(count, "k").Store(k)
			valPtr := elements.Index(count, "v")
			v := c.initialValue(s, t.ValueType)
			valPtr.Store(v)
			countPtr.Store(s.AddS(count, uint64(1)))

			c.reference(s, v, t.ValueType)
			c.reference(s, k, t.KeyType)

			s.Return(valPtr)
		})
	}))

	lookup := c.module.Function(valTy, t.Name()+"•lookup", c.ty.ctxPtr, mapPtrTy, keyTy)
	err(lookup.Build(func(jb *codegen.Builder) {
		s := c.scope(jb)
		m := s.Parameter(1).SetName("map")
		k := s.Parameter(2).SetName("key")
		ptr := s.Call(index, s.ctx, m, k, s.Scalar(false))
		s.If(ptr.IsNull(), func() {
			s.Return(c.initialValue(s, t.ValueType))
		})
		v := ptr.Load()
		c.reference(s, v, t.ValueType)
		s.Return(v)
	}))

	remove := c.module.Function(c.ty.Void, t.Name()+"•remove", c.ty.ctxPtr, mapPtrTy, keyTy)
	err(remove.Build(func(jb *codegen.Builder) {
		s := c.scope(jb)
		m := s.Parameter(1).SetName("map")
		k := s.Parameter(2).SetName("key")

		countPtr := m.Index(0, mapCount)
		elementsPtr := m.Index(0, mapElements)
		count := countPtr.Load()
		elements := elementsPtr.Load()

		// Search for element
		s.ForN(count, func(it *codegen.Value) *codegen.Value {
			found := c.equal(s, elements.Index(it, "k").Load(), k)
			s.If(found, func() {
				// Release references to el
				elPtr := elements.Index(it)
				if c.isRefCounted(t.KeyType) {
					c.release(s, elPtr.Index(0, "k").Load(), t.KeyType)
				}
				if c.isRefCounted(t.ValueType) {
					c.release(s, elPtr.Index(0, "v").Load(), t.ValueType)
				}
				// Replace element with last
				countM1 := s.SubS(count, uint64(1)).SetName("count-1")
				last := elements.Index(countM1).SetName("last").Load()
				elPtr.Store(last)
				// Decrement count
				countPtr.Store(countM1)
			})
			return s.Not(found)
		})
	}))

	clear := c.module.Function(nil, t.Name()+"•clear", c.ty.ctxPtr, mapPtrTy)
	err(clear.Build(func(jb *codegen.Builder) {
		s := c.scope(jb)
		m := s.Parameter(1).SetName("map")
		count := m.Index(0, mapCount).Load()
		elements := m.Index(0, mapElements).Load()
		if c.isRefCounted(t.KeyType) || c.isRefCounted(t.ValueType) {
			s.ForN(count, func(it *codegen.Value) *codegen.Value {
				if c.isRefCounted(t.KeyType) {
					c.release(s, elements.Index(it, "k").Load(), t.KeyType)
				}
				if c.isRefCounted(t.ValueType) {
					c.release(s, elements.Index(it, "v").Load(), t.ValueType)
				}
				return nil
			})
		}
		c.free(s, elements)
		m.Index(0, mapCount).Store(s.Scalar(uint64(0)))
		m.Index(0, mapCapacity).Store(s.Scalar(uint64(0)))
	}))

	mi := c.ty.maps[t]
	mi.Contains = contains
	mi.Index = index
	mi.Lookup = lookup
	mi.Remove = remove
	mi.Clear = clear
	c.ty.maps[t] = mi
}

const mapGrowBy = 16

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
