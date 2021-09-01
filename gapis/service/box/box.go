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

package box

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/google/gapid/core/data/deep"
	"github.com/google/gapid/core/data/dictionary"
	"github.com/google/gapid/core/data/pod"
	"github.com/google/gapid/core/data/slice"
	"github.com/google/gapid/gapis/memory"
)

// NewValue attempts to box and return v into a Value.
// If v cannot be boxed into a Value then nil is returned.
func NewValue(v interface{}) *Value {
	return newBoxer().val(reflect.ValueOf(v))
}

// NewType returns the Type of value v
func NewType(t reflect.Type) *Type {
	return newBoxer().ty(t)
}

// Get returns the boxed value.
func (v *Value) Get() interface{} {
	return newUnboxer().val(v).Interface()
}

// Get returns the type as a reflect.Type.
func (t *Type) Get() reflect.Type {
	return newUnboxer().ty(t)
}

// AssignTo assigns the boxed value to the value at pointer p.
func (v *Value) AssignTo(p interface{}) error {
	unboxer := newUnboxer()
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Errorf("%v\nValue: %v", r, v))
		}
	}()
	s := unboxer.val(v).Interface()
	return deep.Copy(p, s)
}

type BoxedArray interface {
	GetArrayValues() interface{}
}

// The part of the api.Handle interface we care about here.
// TODO(pmuetschard): there are some tricky dependencies that need to be cleaned
// up so that we can refer to the actual interface from here without circular
// dependencies.
type handle interface {
	Handle() uint64
}

var (
	tyEmptyInterface = reflect.TypeOf((*interface{})(nil)).Elem()
	tyMemoryPointer  = reflect.TypeOf((*memory.Pointer)(nil)).Elem()
	tyMemorySlice    = reflect.TypeOf((*memory.Slice)(nil)).Elem()
	tyBoxedArray     = reflect.TypeOf((*BoxedArray)(nil)).Elem()
	tyHandle         = reflect.TypeOf((*handle)(nil)).Elem()
	noValue          = reflect.Value{}
)

// IsMemoryPointer returns true if t is a (or is an alias of a) memory.Pointer.
func IsMemoryPointer(t reflect.Type) bool {
	return t.Implements(tyMemoryPointer)
}

// IsMemorySlice returns true if t implements memory.Slice.
func IsMemorySlice(t reflect.Type) bool {
	return t.Implements(tyMemorySlice)
}

// IsBoxedArray returns true if t implements memory.Slice.
func IsBoxedArray(t reflect.Type) bool {
	return t.Implements(tyBoxedArray)
}

// IsHandle returns true if t implements api.Handle.
func IsHandle(t reflect.Type) bool {
	return t.Implements(tyHandle)
}

// AsMemoryPointer returns v cast to a memory.Pointer. IsMemoryPointer must
// return true for the type of v.
func AsMemoryPointer(v reflect.Value) memory.Pointer {
	return v.Interface().(memory.Pointer)
}

// AsMemorySlice returns v cast to a memory.Slice. IsMemorySlice must
// return true for the type of v.
func AsMemorySlice(v reflect.Value) memory.Slice {
	return v.Interface().(memory.Slice)
}

type boxer struct {
	values map[reflect.Value]uint32
	types  map[reflect.Type]uint32
}

func newBoxer() *boxer {
	return &boxer{map[reflect.Value]uint32{}, map[reflect.Type]uint32{}}
}

func (b *boxer) val(v reflect.Value) *Value {
	t := v.Type()

	switch {
	case IsHandle(t):
		h := v.Interface().(handle).Handle()
		return &Value{Val: &Value_Handle{&Handle{Value: h}}}
	case IsMemoryPointer(t):
		p := AsMemoryPointer(v)
		return &Value{Val: &Value_Pointer{&Pointer{Address: p.Address()}}}
	case IsMemorySlice(t):
		s := v.Interface().(memory.Slice)
		elTy, ok := pod.TypeOf(s.ElementType())
		if !ok {
			panic(fmt.Errorf("Type %T is not a POD type", s.ElementType()))
		}
		return &Value{Val: &Value_Slice{&Slice{
			Type:  elTy,
			Pool:  uint64(s.Pool()),
			Base:  &Pointer{Address: s.Base()},
			Size:  s.Size(),
			Count: s.Count(),
			Root:  s.Root(),
		}}}
	case IsBoxedArray(t):
		s := v.Interface().(BoxedArray)
		v = reflect.ValueOf(s.GetArrayValues())
		return b.val(v)
	}

	if b := pod.NewValue(v.Interface()); b != nil {
		return &Value{Val: &Value_Pod{b}}
	}

	switch t.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return &Value{Val: &Value_Reference{&Reference{Val: &Reference_Null{}}}}
		}
		return b.val(v.Elem())
	}

	id, ok := b.values[v]
	if ok {
		return &Value{ValueId: id, Val: &Value_BackReference{true}}
	}
	id = uint32(len(b.values) + 1)
	b.values[v] = id

	if d := dictionary.From(v.Interface()); d != nil {
		entries := []*MapEntry{}
		mapTy := b.ty(reflect.MapOf(d.KeyTy(), d.ValTy()))
		for _, k := range d.Keys() {
			v := d.Get(k)
			entries = append(entries, &MapEntry{
				Key:   b.val(reflect.ValueOf(k)),
				Value: b.val(reflect.ValueOf(v)),
			})
		}
		m := &Map{Type: mapTy, Entries: entries}
		m.Sort()
		return &Value{ValueId: id, Val: &Value_Map{m}}
	}

	switch t.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			ptrTy := b.ty(v.Type().Elem())
			return &Value{ValueId: id, Val: &Value_Reference{&Reference{Val: &Reference_Null{ptrTy}}}}
		}
		ptrVal := b.val(v.Elem())
		return &Value{ValueId: id, Val: &Value_Reference{&Reference{Val: &Reference_Value{ptrVal}}}}

	case reflect.Struct:
		structTy := b.ty(t)
		fields := make([]*Value, 0, t.NumField())
		for i, c := 0, t.NumField(); i < c; i++ {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue // Unexported.
			}
			if f.Tag.Get("nobox") == "true" {
				continue // Explictly disabled.
			}
			fields = append(fields, b.val(v.FieldByName(f.Name)))
		}
		return &Value{ValueId: id, Val: &Value_Struct{&Struct{Type: structTy, Fields: fields}}}

	case reflect.Slice, reflect.Array:
		arrTy := b.ty(v.Type())
		entries := []*Value{}
		for i, c := 0, v.Len(); i < c; i++ {
			entries = append(entries, b.val(v.Index(i)))
		}
		return &Value{ValueId: id, Val: &Value_Array{&Array{Type: arrTy, Entries: entries}}}
	}

	panic(fmt.Errorf("Unsupported Type %v", t))
}

func (b *boxer) ty(t reflect.Type) *Type {
	switch {
	case IsHandle(t):
		return &Type{Ty: &Type_Handle{true}}
	case IsMemoryPointer(t):
		return &Type{Ty: &Type_Pointer{true}}
	case IsMemorySlice(t):
		return &Type{Ty: &Type_Slice{true}}
	}

	if podTy, ok := pod.TypeOf(t); ok {
		return &Type{Ty: &Type_Pod{podTy}}
	}

	switch t.Kind() {
	case reflect.Interface:
		return &Type{Ty: &Type_Any{true}}
	}

	// Types below this point can be back-referenced.
	id, ok := b.types[t]
	if ok {
		return &Type{TypeId: id, Ty: &Type_BackReference{true}}
	}
	id = uint32(len(b.types) + 1)
	b.types[t] = id

	if d := dictionary.From(reflect.New(t).Interface()); d != nil {
		keyTy := b.ty(d.KeyTy())
		valTy := b.ty(d.ValTy())
		return &Type{TypeId: id, Ty: &Type_Map{&MapType{KeyType: keyTy, ValueType: valTy}}}
	}

	switch t.Kind() {
	case reflect.Ptr:
		return &Type{TypeId: id, Ty: &Type_Reference{b.ty(t.Elem())}}

	case reflect.Struct:
		fields := make([]*StructField, 0, t.NumField())
		for i, c := 0, t.NumField(); i < c; i++ {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue // Unexported.
			}
			if f.Tag.Get("nobox") == "true" {
				continue // Explictly disabled.
			}
			fields = append(fields, &StructField{Type: b.ty(f.Type), Name: f.Name})
		}
		return &Type{TypeId: id, Ty: &Type_Struct{&StructType{Fields: fields}}}

	case reflect.Map:
		keyTy := b.ty(t.Key())
		valTy := b.ty(t.Elem())
		return &Type{TypeId: id, Ty: &Type_Map{&MapType{KeyType: keyTy, ValueType: valTy}}}

	case reflect.Array, reflect.Slice:
		elTy := b.ty(t.Elem())
		return &Type{TypeId: id, Ty: &Type_Array{&ArrayType{ElementType: elTy}}}
	}

	panic(fmt.Errorf("Unsupported Type %v", t))
}

type unboxer struct {
	values map[uint32]reflect.Value
	types  map[uint32]reflect.Type
}

func newUnboxer() *unboxer {
	return &unboxer{
		map[uint32]reflect.Value{},
		map[uint32]reflect.Type{},
	}
}

func (b *unboxer) val(v *Value) (out reflect.Value) {
	switch v := v.Val.(type) {
	case *Value_Pod:
		if v := v.Pod.Get(); v != nil {
			return reflect.ValueOf(v)
		}
		panic(fmt.Errorf("Unsupported POD Value %+v", v))
	case *Value_Handle:
		return reflect.ValueOf(v.Handle.Value)
	case *Value_Pointer:
		p := memory.BytePtr(v.Pointer.Address)
		return reflect.ValueOf(p)
	case *Value_Slice:
		p := memory.NewSlice(
			v.Slice.Root,
			v.Slice.Base.Address,
			v.Slice.Size,
			v.Slice.Count,
			memory.PoolID(v.Slice.Pool),
			v.Slice.Type.Get(),
		)
		return reflect.ValueOf(p)
	}

	if v.GetBackReference() {
		if val, ok := b.values[v.ValueId]; ok {
			return val
		}
		panic(fmt.Errorf("Unknown value id %v", v.ValueId))
	}

	defer func() { b.values[v.ValueId] = out }()

	switch v := v.Val.(type) {
	case *Value_Reference:
		switch p := v.Reference.Val.(type) {
		case *Reference_Null:
			if p.Null == nil {
				return noValue
			}
			return reflect.New(reflect.PtrTo(b.ty(p.Null))).Elem()
		case *Reference_Value:
			val := b.val(p.Value)
			clone := reflect.New(val.Type()).Elem()
			clone.Set(val)
			return clone.Addr()
		}
	case *Value_Map:
		mapTy := b.ty(v.Map.Type)
		if mapTy.Kind() != reflect.Map {
			panic(fmt.Errorf("Expected map, got %v (%v)", mapTy, mapTy.Kind()))
		}
		mapVal := reflect.MakeMap(mapTy)
		for _, e := range v.Map.Entries {
			k, v := b.val(e.Key), b.val(e.Value)
			mapVal.SetMapIndex(k, v)
		}
		return mapVal
	case *Value_Array:
		arrTy := b.ty(v.Array.Type)
		arrVal := slice.New(arrTy, len(v.Array.Entries), len(v.Array.Entries))
		for i, e := range v.Array.Entries {
			v := b.val(e)
			arrVal.Index(i).Set(v)
		}
		return arrVal
	case *Value_Struct:
		structTy := b.ty(v.Struct.Type)
		structVal := reflect.New(structTy).Elem()
		for i, c := 0, structTy.NumField(); i < c; i++ {
			f := structVal.FieldByName(structTy.Field(i).Name)
			if v := b.val(v.Struct.Fields[i]); v != noValue {
				f.Set(v)
			}
		}
		return structVal
	}

	panic(fmt.Errorf("Unsupported Value %+v", v))
}

func (b *unboxer) ty(t *Type) (out reflect.Type) {
	switch t := t.Ty.(type) {
	case *Type_Pod:
		if ty := t.Pod.Get(); ty != nil {
			return ty
		}
		panic(fmt.Errorf("Unsupported POD type %v", t.Pod))

	case *Type_Any:
		return tyEmptyInterface

	case *Type_Handle:
		return tyHandle

	case *Type_Pointer:
		return tyMemoryPointer

	case *Type_Slice:
		return tyMemorySlice
	}

	id := t.TypeId
	if t.GetBackReference() {
		if ty, ok := b.types[id]; ok {
			return ty
		}
		panic(fmt.Errorf("Unknown type id %v. Known types: %+v", id, b.types))
	}

	defer func() { b.types[id] = out }()

	switch t := t.Ty.(type) {
	case *Type_Reference:
		// Workaround for https://github.com/golang/go/issues/20013
		// Until the pointee type has been built, use an interface{}.
		b.types[id] = tyEmptyInterface
		if t.Reference.GetBackReference() {
			if _, known := b.types[t.Reference.TypeId]; !known {
				return tyEmptyInterface
			}
		}
		return reflect.PtrTo(b.ty(t.Reference))
	case *Type_Struct:
		fields := make([]reflect.StructField, len(t.Struct.Fields))
		for i := range fields {
			fields[i] = reflect.StructField{
				Name: t.Struct.Fields[i].Name,
				Type: b.ty(t.Struct.Fields[i].Type),
			}
		}
		return reflect.StructOf(fields)
	case *Type_Map:
		keyTy := b.ty(t.Map.KeyType)
		valTy := b.ty(t.Map.ValueType)
		return reflect.MapOf(keyTy, valTy)
	case *Type_Array:
		elTy := b.ty(t.Array.ElementType)
		return reflect.SliceOf(elTy)
	default:
		panic(fmt.Errorf("Unsupported Type %T", t))
	}
}

// Sort sorts the entries in the map using the keys lexicographic order.
func (m *Map) Sort() {
	keys := make([]string, len(m.Entries))
	for i, e := range m.Entries {
		keys[i] = fmt.Sprint(e.Key.Get())
	}
	sort.Sort(mapSorter{keys, m.Entries})
}

type mapSorter struct {
	keys    []string
	entries []*MapEntry
}

func (m mapSorter) Len() int           { return len(m.keys) }
func (m mapSorter) Less(i, j int) bool { return m.keys[i] < m.keys[j] }
func (m mapSorter) Swap(i, j int) {
	m.keys[i], m.keys[j] = m.keys[j], m.keys[i]
	m.entries[i], m.entries[j] = m.entries[j], m.entries[i]
}
