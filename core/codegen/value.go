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

package codegen

import (
	"fmt"
	"strings"

	"llvm/bindings/go/llvm"
)

func (b *Builder) val(ty Type, val llvm.Value) *Value {
	// if a, b := val.Type().String(), ty.llvmTy().String(); a != b {
	// 	fail("Value type mismatch: value: %v, type: %v", a, b)
	// }
	return &Value{ty, val, val.Name(), b}
}

// Value represents a value.
type Value struct {
	ty   Type
	llvm llvm.Value
	name string
	b    *Builder
}

// Type returns the type of the value.
func (v *Value) Type() Type { return v.ty }

// Name returns the value's name.
func (v *Value) Name() string { return v.name }

// SetName assigns a name to the value.
func (v *Value) SetName(name string) *Value {
	v.name = name
	if IsPointer(v.ty) {
		name = "→" + name
	}
	v.llvm.SetName(name)
	return v
}

// EmitDebug emits debug info for the value.
func (v *Value) EmitDebug(name string) *Value {
	v.b.dbgEmitValue(v, name)
	return v
}

// Load loads the element from the pointer value.
func (v *Value) Load() *Value {
	if !IsPointer(v.ty) {
		fail("Load must be from a pointer. Got %v", v.Type())
	}
	elTy := v.Type().(Pointer).Element
	return v.b.val(elTy, v.b.llvm.CreateLoad(v.llvm, v.name))
}

// LoadUnaligned loads the element from the pointer value, it allows the loaded
// value to be unaligned
func (v *Value) LoadUnaligned() *Value {
	if !IsPointer(v.ty) {
		fail("Load must be from a pointer. Got %v", v.Type)
	}
	elTy := v.Type().(Pointer).Element
	load := v.b.llvm.CreateLoad(v.llvm, v.name)
	load.SetAlignment(1)
	return v.b.val(elTy, load)
}

// StoreUnaligned stores val to the pointer ptr. It assumes that the
// destination address may be unaligned
func (v *Value) StoreUnaligned(val *Value) {
	if !IsPointer(v.ty) {
		fail("Store must be to a pointer. Got %v", v.Type)
	}
	elTy := v.Type().(Pointer).Element
	if val.ty.String() != elTy.String() {
		fail("Attempted to store value of type %v to pointer element type %v",
			val.ty.TypeName(), elTy.TypeName())
	}
	store := v.b.llvm.CreateStore(val.llvm, v.llvm)
	store.SetAlignment(1)
}

// Store stores val to the pointer ptr.
func (v *Value) Store(val *Value) {
	if !IsPointer(v.ty) {
		fail("Store must be to a pointer. Got %v", v.Type)
	}
	elTy := v.Type().(Pointer).Element
	if val.ty.String() != elTy.String() {
		fail("Attempted to store value of type %v to pointer element type %v",
			val.ty.TypeName(), elTy.TypeName())
	}
	v.b.llvm.CreateStore(val.llvm, v.llvm)
}

func field(s *Struct, f IndexOrName) (Field, int) {
	var i int
	switch f := f.(type) {
	case int:
		i = f
	case string:
		var ok bool
		if i, ok = s.fieldIndices[f]; !ok {
			fail("%v does not contain a field with name '%v':\n%+v", s.TypeName(), f, s)
		}
	default:
		fail("Attempted to index field of struct '%v' with %T. Must be int or string", s.TypeName(), f)
	}
	return s.fields[i], i
}

func (b *Builder) pathName(rootName string, path []ValueIndexOrName) string {
	name := rootName
	for _, p := range path {
		switch p := p.(type) {
		case int:
			name = fmt.Sprintf("%v[%v]", name, p)
		case string:
			name = fmt.Sprintf("%v.%v", name, p)
		case *Value:
			name = fmt.Sprintf("%v[%v]", name, p.Name())
		}
	}
	return name
}

func (b *Builder) path(rootTy Type, rootName string, path ...ValueIndexOrName) (indices []llvm.Value, name string, target Type) {
	err := func(i int) string {
		full := b.pathName(rootName, path)
		okay := b.pathName(rootName, path[:i])
		fail := b.pathName(rootName, path[:i+1])
		pad := strings.Repeat(" ", len(okay))
		highlight := strings.Repeat("^", len(fail)-len(okay))
		return fmt.Sprintf("\n%v\n%v%v", full, pad, highlight)
	}
	target = rootTy
	indices = make([]llvm.Value, len(path))
	for i, p := range path {
		switch t := target.(type) {
		case Pointer:
			if i == 0 {
				switch p := p.(type) {
				case int:
					indices[i] = b.Scalar(uint32(p)).llvm
				case *Value:
					if !IsInteger(p.Type()) {
						fail("Tried to index pointer with non-integer %v.%v", p.Type().TypeName(), err(i))
					}
					indices[i] = p.llvm
				default:
					fail("Tried to index pointer with %T (%v).%v", p, err(i))
				}
				target = t.Element
			} else {
				fail("Tried to index %v. Only the root pointer can be indexed.%v", target.TypeName(), err(i))
			}
		case *Struct:
			field, idx := field(t, p)
			target = field.Type
			indices[i] = b.Scalar(uint32(idx)).llvm
		case *Array:
			switch p := p.(type) {
			case int:
				indices[i] = b.Scalar(uint32(p)).llvm
			case *Value:
				indices[i] = p.llvm
			default:
				fail("Tried to index array with %T.%v", p, err(i))
			}
			target = t.Element
		default:
			fail("Cannot index type %v.%v", target, err(i))
		}
	}
	return indices, b.pathName(rootName, path), target
}

// Index returns a new pointer to the array or field element found by following
// the list of indices as specified by path.
func (v *Value) Index(path ...ValueIndexOrName) *Value {
	if !IsPointer(v.ty) {
		fail("Index only works with pointer value types. Got %v", v.Type().TypeName())
	}
	indices, name, target := v.b.path(v.Type(), v.Name(), path...)
	return v.b.val(v.b.m.Types.Pointer(target), v.b.llvm.CreateGEP(v.llvm, indices, "")).SetName(name)
}

// Insert creates a copy of the struct or array v with the field/element at
// changed to val.
func (v *Value) Insert(at ValueIndexOrName, val *Value) *Value {
	switch ty := v.ty.(type) {
	case *Struct:
		f, idx := field(ty, at)
		assertTypesEqual(f.Type, val.Type())
		return v.b.val(ty, v.b.llvm.CreateInsertValue(v.llvm, val.llvm, idx, "")).SetName(v.Name())
	case *Array:
		idx, ok := at.(int)
		if !ok {
			fail("Insert parameter at must be int for arrays values. Got %T", at)
		}
		assertTypesEqual(ty.Element, val.Type())
		return v.b.val(ty, v.b.llvm.CreateInsertValue(v.llvm, val.llvm, idx, "")).SetName(v.Name())
	default:
		fail("Attempted to insert on non-struct and non-array type %v", v.ty.TypeName())
		return nil
	}
}

// Extract returns the field at extracted from the struct or array v.
func (v *Value) Extract(at IndexOrName) *Value {
	switch ty := v.ty.(type) {
	case *Struct:
		f, idx := field(ty, at)
		return v.b.val(f.Type, v.b.llvm.CreateExtractValue(v.llvm, idx, f.Name))
	case *Array:
		idx, ok := at.(int)
		if !ok {
			fail("Extract parameter at must be int for arrays values. Got %T", at)
		}
		return v.b.val(ty.Element, v.b.llvm.CreateExtractValue(v.llvm, idx, fmt.Sprintf("%v[%d]", v.name, idx)))
	default:
		fail("Attempted to extract on non-struct and non-array type %v", v.ty.TypeName())
		return nil
	}
}

// IsNull returns true if the pointer value v is null.
func (v *Value) IsNull() *Value {
	if !IsPointer(v.ty) {
		fail("IsNull only works with pointer value types. Got %v", v.Type().TypeName())
	}
	return v.b.val(v.b.m.Types.Bool, v.b.llvm.CreateIsNull(v.llvm, ""))
}
