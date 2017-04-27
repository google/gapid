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
	"github.com/google/gapid/core/data/pod"
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

var (
	tyEmptyInterface = reflect.TypeOf((*interface{})(nil)).Elem()
	tyMemoryPointer  = reflect.TypeOf(memory.Pointer{})
	noValue          = reflect.Value{}
)

type boxer struct {
	values map[reflect.Value]uint32
	types  map[reflect.Type]uint32
}

func newBoxer() *boxer {
	return &boxer{map[reflect.Value]uint32{}, map[reflect.Type]uint32{}}
}

func (b *boxer) val(v reflect.Value) *Value {
	if b := pod.NewValue(v.Interface()); b != nil {
		return &Value{0, &Value_Pod{b}}
	}

	t := v.Type()
	switch t.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return &Value{0, &Value_Reference{&Reference{&Reference_Null{}}}}
		}
		return b.val(v.Elem())
	}

	switch {
	case t.ConvertibleTo(tyMemoryPointer):
		p := v.Convert(tyMemoryPointer).Interface().(memory.Pointer)
		return &Value{0, &Value_Pointer{&Pointer{p.Address, uint32(p.Pool)}}}
	}

	id, ok := b.values[v]
	if ok {
		return &Value{id, &Value_BackReference{true}}
	}
	id = uint32(len(b.values) + 1)
	b.values[v] = id

	switch t.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			ptrTy := b.ty(v.Type().Elem())
			return &Value{id, &Value_Reference{&Reference{&Reference_Null{ptrTy}}}}
		}
		ptrVal := b.val(v.Elem())
		return &Value{id, &Value_Reference{&Reference{&Reference_Value{ptrVal}}}}

	case reflect.Struct:
		structTy := b.ty(t)
		fields := make([]*Value, 0, t.NumField())
		for i, c := 0, t.NumField(); i < c; i++ {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue // Unexported.
			}
			fields = append(fields, b.val(v.FieldByName(f.Name)))
		}
		return &Value{id, &Value_Struct{&Struct{structTy, fields}}}

	case reflect.Map:
		mapTy := b.ty(v.Type())
		entries := []*MapEntry{}
		for _, k := range v.MapKeys() {
			entries = append(entries, &MapEntry{
				Key:   b.val(k),
				Value: b.val(v.MapIndex(k)),
			})
		}
		m := &Map{Type: mapTy, Entries: entries}
		m.Sort()
		return &Value{id, &Value_Map{m}}
	}

	panic(fmt.Errorf("Unsupported Type %v", t))
}

func (b *boxer) ty(t reflect.Type) *Type {
	if podTy, ok := pod.TypeOf(t); ok {
		return &Type{0, &Type_Pod{podTy}}
	}

	switch t.Kind() {
	case reflect.Interface:
		return &Type{0, &Type_Any{true}}
	}

	switch {
	case t.ConvertibleTo(tyMemoryPointer):
		return &Type{0, &Type_Pointer{true}}
	}

	id, ok := b.types[t]
	if ok {
		return &Type{id, &Type_BackReference{true}}
	}
	id = uint32(len(b.types) + 1)
	b.types[t] = id

	switch t.Kind() {
	case reflect.Ptr:
		return &Type{id, &Type_Reference{b.ty(t.Elem())}}

	case reflect.Struct:
		fields := make([]*StructField, 0, t.NumField())
		for i, c := 0, t.NumField(); i < c; i++ {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue // Unexported.
			}
			fields = append(fields, &StructField{Type: b.ty(f.Type), Name: f.Name})
		}
		return &Type{id, &Type_Struct{&StructType{fields}}}

	case reflect.Map:
		keyTy := b.ty(t.Key())
		valTy := b.ty(t.Elem())
		return &Type{id, &Type_Map{&MapType{keyTy, valTy}}}
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
	case *Value_Pointer:
		p := memory.Pointer{Address: v.Pointer.Address, Pool: memory.PoolID(v.Pointer.Pool)}
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
		mapVal := reflect.MakeMap(mapTy)
		for _, e := range v.Map.Entries {
			k, v := b.val(e.Key), b.val(e.Value)
			mapVal.SetMapIndex(k, v)
		}
		return mapVal
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

	case *Type_Pointer:
		return tyMemoryPointer
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
