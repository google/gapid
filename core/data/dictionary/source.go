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

package dictionary

import (
	"reflect"

	"github.com/google/gapid/core/data/generic"
)

type (
	// K represents a generic key type.
	K = generic.T1

	// V represents a generic value type.
	V = generic.T2
)

var (
	kTy = generic.T1Ty
	vTy = generic.T2Ty
)

// Source is a placeholder interface used to prototype a generic dictionary
// type. K and V can be subsituted with a different type.
type Source interface {
	// Get returns the value of the entry with the given key.
	Get(K) V
	// Add inserts the key-value pair, replacing any existing entry with the
	// same key.
	Add(K, V)
	// Lookup searches for the value of the entry with the given key.
	Lookup(K) (val V, ok bool)
	// Contains returns true if the dictionary contains an entry with the given
	// key.
	Contains(K) bool
	// Remove removes the entry with the given key. If no entry with the given
	// key exists then this call is a no-op.
	Remove(K)
	// Len returns the number of entries in the dictionary.
	Len() int
	// Keys returns all the entry keys in the map.
	Keys() []K
}

var sourceTy = reflect.TypeOf((*Source)(nil)).Elem()

// IsSource returns nil if v implements the generic interface Source.
func IsSource(o interface{}) []error {
	return generic.Implements(reflect.TypeOf(o), sourceTy, reflect.TypeOf(K{}), reflect.TypeOf(V{})).Errors
}

func newSource(v reflect.Value) I {
	m := generic.Implements(v.Type(), sourceTy, reflect.TypeOf(K{}), reflect.TypeOf(V{}))
	if !m.Ok() {
		return nil
	}

	return source{
		keyTy:    m.Bindings[kTy],
		valTy:    m.Bindings[vTy],
		get:      v.MethodByName("Get"),
		add:      v.MethodByName("Add"),
		lookup:   v.MethodByName("Lookup"),
		contains: v.MethodByName("Contains"),
		remove:   v.MethodByName("Remove"),
		len:      v.MethodByName("Len"),
		keys:     v.MethodByName("Keys"),
	}
}

// source implements I using a Source.
type source struct {
	keyTy    reflect.Type
	valTy    reflect.Type
	get      reflect.Value
	add      reflect.Value
	lookup   reflect.Value
	contains reflect.Value
	remove   reflect.Value
	len      reflect.Value
	keys     reflect.Value
}

func (s source) Get(key interface{}) interface{} {
	return s.get.Call([]reflect.Value{reflect.ValueOf(key)})[0].Interface()
}

func (s source) Add(key interface{}, val interface{}) {
	s.add.Call([]reflect.Value{reflect.ValueOf(key), reflect.ValueOf(val)})
}

func (s source) Lookup(key interface{}) (interface{}, bool) {
	res := s.lookup.Call([]reflect.Value{reflect.ValueOf(key)})
	return res[0].Interface(), res[1].Interface().(bool)
}

func (s source) Contains(key interface{}) bool {
	return s.contains.Call([]reflect.Value{reflect.ValueOf(key)})[0].Interface().(bool)
}

func (s source) Remove(key interface{}) {
	s.remove.Call([]reflect.Value{reflect.ValueOf(key)})
}

func (s source) Len() int {
	return s.len.Call([]reflect.Value{})[0].Interface().(int)
}

func (s source) Keys() []interface{} {
	l := s.keys.Call([]reflect.Value{})[0]
	out := make([]interface{}, l.Len())
	for i := range out {
		out[i] = l.Index(i).Interface()
	}
	return out
}

func (s source) KeyTy() reflect.Type { return s.keyTy }

func (s source) ValTy() reflect.Type { return s.valTy }
