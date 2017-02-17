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

// Map is the Type descriptor for key/value stores.
type Map struct {
	Alias     string      // The alias this array type was given, if present
	KeyType   binary.Type // The key type used.
	ValueType binary.Type // The value type stored in the map.
}

func (m *Map) Representation() string {
	return fmt.Sprintf("%r", m)
}

func (m *Map) String() string {
	return fmt.Sprint(m)
}

// Format implements the fmt.Formatter interface
func (m *Map) Format(f fmt.State, c rune) {
	switch c {
	case 'z': // Private format specifier, supports Entity.Signature
		fmt.Fprintf(f, "map[%z]%z", m.KeyType, m.ValueType)
	case 'r': // Private format specifier, supports Type.Representation
		fmt.Fprintf(f, "map[%r]%r", m.KeyType, m.ValueType)
	default:
		if m.Alias != "" {
			fmt.Fprint(f, m.Alias)
		} else {
			fmt.Fprintf(f, "map[%v]%v", m.KeyType, m.ValueType)
		}
	}
}

func (m *Map) EncodeValue(e binary.Encoder, value interface{}) {
	v := value.(map[interface{}]interface{})
	e.Uint32(uint32(len(v)))
	for k, o := range v {
		m.KeyType.EncodeValue(e, k)
		m.ValueType.EncodeValue(e, o)
	}
}

func (m *Map) DecodeValue(d binary.Decoder) interface{} {
	count := d.Count()
	v := make(map[interface{}]interface{}, count)
	for i := uint32(0); i < count; i++ {
		k := m.KeyType.DecodeValue(d)
		v[k] = m.ValueType.DecodeValue(d)
	}
	return v
}

func (m *Map) Subspace() *binary.Subspace {
	var subs binary.TypeList
	if m.KeyType.HasSubspace() {
		subs = binary.TypeList{m.KeyType}
	}
	if m.ValueType.HasSubspace() {
		subs = append(subs, m.ValueType)
	}
	return &binary.Subspace{Counted: true, SubTypes: subs}
}

func (m *Map) HasSubspace() bool {
	// Always has to decode a count (even if the loop has no sub-types).
	return true
}

func (*Map) IsPOD() bool {
	return false
}

func (m *Map) IsSimple() bool {
	return m.KeyType.IsPOD() && m.ValueType.IsPOD()
}

func mapFactory(t reflect.Type, tag reflect.StructTag, makeType binary.MakeTypeFun, pkg string) binary.Type {
	m := &Map{Alias: binary.TypeName(t, pkg)}
	makeTypePrime := registry.PreventLoopsFun(t, m, makeType)
	m.KeyType = makeTypePrime(t.Key(), reflect.StructTag(""), makeTypePrime, pkg)
	m.ValueType = makeTypePrime(t.Elem(), reflect.StructTag(""), makeTypePrime, pkg)
	return m
}

func init() {
	registry.Factories.Add(reflect.Map, mapFactory)
}
