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

package compare

import (
	"fmt"
	"reflect"
	"unicode"
	"unicode/utf8"
	"unsafe"
)

// Comparator is passed to custom comparison functions holding the current comparison context.
type Comparator struct {
	Path    Path
	Handler Handler
	seen    seen
	custom  *Custom
}

type seen map[seenKey]struct{}

type seenKey struct {
	typ   reflect.Type
	addr1 uintptr
	addr2 uintptr
}

func toValue(v reflect.Value, wasPointer bool) interface{} {
	if wasPointer {
		v = v.Addr()
	}
	return v.Interface()
}

// Compare can be called by custom comparison functions to recurse into children.
func (t Comparator) Compare(reference, value interface{}) {
	switch {
	case reference == nil && value == nil:
		return
	case reference == nil || value == nil:
		t.Handler(t.Path.Nil(reference, value))
		return
	}
	t.compareValues(reflect.ValueOf(reference), reflect.ValueOf(value), false)
}

// With returns a new Comparator based on t but with the specified Path.
func (t Comparator) With(p Path) Comparator {
	t.Path = p
	return t
}

// AddDiff can be used by custom comparison functions to register a new difference.
// The current handler will be invoked with the current path plus a Diff fragment.
func (t Comparator) AddDiff(reference, value interface{}) {
	t.Handler(t.Path.Diff(reference, value))
}

func (t Comparator) compareValues(v1, v2 reflect.Value, ptr bool) {
	// First see if we have seen this comparison already
	switch v1.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.Struct:
		if v1.CanAddr() && v2.CanAddr() {
			key := seenKey{
				typ:   v1.Type(),
				addr1: v1.UnsafeAddr(),
				addr2: v2.UnsafeAddr(),
			}
			if key.addr1 > key.addr2 {
				// swap for stable ordering
				key.addr1, key.addr2 = key.addr2, key.addr1
			}
			if _, seen := t.seen[key]; seen {
				// Already seen, no need to traverse again
				return
			}
			t.seen[key] = struct{}{}
		}
	}

	key := customKey{v1.Type(), v2.Type()}
	args := []reflect.Value{reflect.ValueOf(t), v1, v2}
	if t.custom.call(key, args) == Done {
		return
	}

	// All following tests assume the types are the same, so check it
	if v1.Type() != v2.Type() {
		t.Handler(t.Path.Type(toValue(v1, ptr), toValue(v2, ptr)).Diff(v1.Type(), v2.Type()))
		return
	}
	// Do all the nil comparison tests in one place
	switch v1.Kind() {
	case reflect.Chan, reflect.Map, reflect.Ptr, reflect.Slice, reflect.Interface:
		switch {
		case v1.IsNil() && v2.IsNil():
			return
		case v1.IsNil():
			t.Handler(t.Path.Nil(nil, toValue(v2, ptr)))
			return
		case v2.IsNil():
			t.Handler(t.Path.Nil(toValue(v1, ptr), nil))
			return
		}
	}
	// Do pointer early out tests
	switch v1.Kind() {
	case reflect.Chan, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		if v1.Pointer() == v2.Pointer() {
			return
		}
	}
	// And now do the kind specific comparisons
	switch v1.Kind() {
	case reflect.Array, reflect.Slice:
		len1, len2 := v1.Len(), v2.Len()
		length, shink := len1, false
		switch {
		case len1 > len2:
			// emit the length diff after the missings so that delta changes happen in
			// logical order.
			length, shink = v2.Len(), true
		case len1 < len2:
			t.Handler(t.Path.Length(toValue(v1, ptr), toValue(v2, ptr)).Diff(len1, len2))
		}
		for i := 0; i < length; i++ {
			t.With(t.Path.Index(i, toValue(v1, ptr), toValue(v2, ptr))).compareValues(v1.Index(i), v2.Index(i), false)
		}
		for i := length; i < len1; i++ {
			t.Handler(t.Path.Index(i, toValue(v1, ptr), toValue(v2, ptr)).Missing(toValue(v1.Index(i), false), Missing))
		}
		for i := length; i < len2; i++ {
			t.Handler(t.Path.Index(i, toValue(v1, ptr), toValue(v2, ptr)).Missing(Missing, toValue(v2.Index(i), false)))
		}
		if shink {
			t.Handler(t.Path.Length(toValue(v1, ptr), toValue(v2, ptr)).Diff(len1, len2))
		}
	case reflect.Interface:
		t.compareValues(v1.Elem(), v2.Elem(), false)
	case reflect.Ptr:
		t.compareValues(v1.Elem(), v2.Elem(), true)
	case reflect.Struct:
		t1 := v1.Type()
		for i, n := 0, t1.NumField(); i < n; i++ {
			f := t1.Field(i)
			if r, _ := utf8.DecodeRuneInString(f.Name); unicode.IsUpper(r) {
				t.With(t.Path.Member(f.Name, toValue(v1, ptr), toValue(v2, ptr))).compareValues(v1.Field(i), v2.Field(i), false)
			} else {
				// Filthy hack to inspect hidden fields.
				// Credit to https://stackoverflow.com/a/43918797
				h1, h2 := addressable(v1).Field(i), addressable(v2).Field(i)
				f1 := reflect.NewAt(h1.Type(), unsafe.Pointer(h1.UnsafeAddr())).Elem()
				f2 := reflect.NewAt(h2.Type(), unsafe.Pointer(h2.UnsafeAddr())).Elem()
				name := fmt.Sprintf("Field<%d>", i)
				t.With(t.Path.Member(name, toValue(v1, ptr), toValue(v2, ptr))).compareValues(f1, f2, false)
			}
		}
	case reflect.Map:
		if v1.Len() != v2.Len() {
			t.Handler(t.Path.Length(toValue(v1, ptr), toValue(v2, ptr)).Diff(v1.Len(), v2.Len()))
		}
		// Check reference keys in value map
		for _, k := range v1.MapKeys() {
			e1 := v1.MapIndex(k)
			e2 := v2.MapIndex(k)
			path := t.Path.Entry(toValue(k, false), toValue(v1, ptr), toValue(v2, ptr))
			if !e2.IsValid() {
				t.Handler(path.Missing(toValue(e1, false), Missing))
			} else {
				t.With(path).compareValues(e1, e2, false)
			}
		}
		// Check for keys in value map that were not in reference
		for _, k := range v2.MapKeys() {
			if !v1.MapIndex(k).IsValid() {
				t.Handler(t.Path.Entry(toValue(k, false), toValue(v1, ptr), toValue(v2, ptr)).Missing(Missing, toValue(v2.MapIndex(k), false)))
			}
		}
	case reflect.Func:
		if !v1.IsNil() || !v2.IsNil() {
			// cant actually compare functions, so any non nil is considered a difference
			t.Handler(t.Path.Diff(toValue(v1, ptr), toValue(v2, ptr)))
		}
	default:
		// Normal equality suffices
		if toValue(v1, false) != toValue(v2, false) {
			t.Handler(t.Path.Diff(toValue(v1, false), toValue(v2, false)))
		}
	}
}

// addressable returns a copy (or passthough) of v so that it can be addressed.
func addressable(v reflect.Value) reflect.Value {
	if v.CanAddr() {
		return v
	}
	c := reflect.New(v.Type()).Elem()
	c.Set(v)
	return c
}
