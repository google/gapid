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

package schema

import (
	"fmt"
	"reflect"

	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/registry"
)

// Array is the Type descriptor for fixed size buffers of known type.
type Array struct {
	Alias     string      // The alias this array type was given, if present
	ValueType binary.Type // The value type stored in the array
	Size      uint32      // The fixed size of the array
}

// Slice is the Type descriptor for dynamically sized buffers of known type,
// encoded with a preceding count.
type Slice struct {
	Alias     string      // The alias this array type was given, if present
	ValueType binary.Type // The value type stored in the slice.
}

func (a *Array) Representation() string {
	return fmt.Sprintf("%r", a)
}

func (a *Array) String() string {
	return fmt.Sprint(a)
}

// Format implements the fmt.Formatter interface
func (a *Array) Format(f fmt.State, c rune) {
	switch c {
	case 'z': // Private format specifier, supports Entity.Signature
		fmt.Fprintf(f, "[%d]%z", a.Size, a.ValueType)
	case 'r': // Private format specifier, supports Type.Representation
		fmt.Fprintf(f, "[%d]%r", a.Size, a.ValueType)
	default:
		if a.Alias != "" {
			fmt.Fprint(f, a.Alias)
		} else {
			fmt.Fprintf(f, "[%d]%v", a.Size, a.ValueType)
		}
	}
}

func (a *Array) EncodeValue(e binary.Encoder, value interface{}) {
	v := value.([]interface{})
	for i := range v {
		a.ValueType.EncodeValue(e, v[i])
	}
}

func (a *Array) DecodeValue(d binary.Decoder) interface{} {
	v := make([]interface{}, a.Size)
	for i := range v {
		v[i] = a.ValueType.DecodeValue(d)
	}
	return v
}

func (s *Slice) Representation() string {
	return fmt.Sprintf("%r", s)
}

func (s *Array) Subspace() *binary.Subspace {
	if s.Size == 0 {
		return nil
	}
	if s.ValueType.HasSubspace() {
		// We don't have examples of this in the stream, so not to bothered
		// if this isn't an efficient approach.
		types := make(binary.TypeList, s.Size, s.Size)
		for i := range types {
			types[i] = s.ValueType
		}
		return &binary.Subspace{Inline: true, SubTypes: types}
	}
	return nil
}

func (s *Array) HasSubspace() bool {
	return s.Size > 0 && s.ValueType.HasSubspace()
}

func (s *Array) IsPOD() bool {
	return s.ValueType.IsPOD()
}

func (s *Array) IsSimple() bool {
	return s.ValueType.IsSimple()
}

func arrayFactory(t reflect.Type, tag reflect.StructTag, makeType binary.MakeTypeFun, pkg string) binary.Type {
	return &Array{
		Alias:     binary.TypeName(t, pkg),
		ValueType: makeType(t.Elem(), reflect.StructTag(""), makeType, pkg),
		Size:      uint32(t.Len()),
	}
}

func (s *Slice) String() string {
	return fmt.Sprint(s)
}

// Format implements the fmt.Formatter interface
func (s *Slice) Format(f fmt.State, c rune) {
	switch c {
	case 'z': // Private format specifier, supports Entity.Signature
		fmt.Fprintf(f, "[]%z", s.ValueType)
	case 'r': // Private format specifier, supports Type.Representation
		fmt.Fprintf(f, "[]%r", s.ValueType)
	default:
		if s.Alias != "" {
			fmt.Fprint(f, s.Alias)
		} else {
			fmt.Fprintf(f, "[]%v", s.ValueType)
		}
	}
}

func (s *Slice) EncodeValue(e binary.Encoder, value interface{}) {
	v := value.([]interface{})
	e.Uint32(uint32(len(v)))
	for i := range v {
		s.ValueType.EncodeValue(e, v[i])
	}
}

func (s *Slice) DecodeValue(d binary.Decoder) interface{} {
	size := d.Count()
	v := make([]interface{}, size)
	for i := range v {
		v[i] = s.ValueType.DecodeValue(d)
	}
	return v
}

func (s *Slice) Subspace() *binary.Subspace {
	var subs binary.TypeList
	if s.ValueType.HasSubspace() {
		subs = binary.TypeList{s.ValueType}
	}
	return &binary.Subspace{Counted: true, SubTypes: subs}
}

func (s *Slice) HasSubspace() bool {
	// Always has to decode a count (even if the loop has no sub-types).
	return true
}

func (s *Slice) IsPOD() bool {
	return false
}

func (s *Slice) IsSimple() bool {
	return s.ValueType.IsPOD()
}

func sliceFactory(t reflect.Type, tag reflect.StructTag, makeType binary.MakeTypeFun, pkg string) binary.Type {
	s := &Slice{Alias: binary.TypeName(t, pkg)}
	if tag.Get("variant") != "" && t.Elem().Kind() == reflect.Interface {
		// codergen just ignores the directive if it isn't an interface
		s.ValueType = variantFactory(t, reflect.StructTag(""), makeType, pkg)
	} else {
		s.ValueType = registry.PreventLoops(
			t, reflect.StructTag(""), s, makeType, t.Elem(), pkg)
	}
	return s
}

func init() {
	registry.Factories.Add(reflect.Slice, sliceFactory)
	registry.Factories.Add(reflect.Array, arrayFactory)
}
