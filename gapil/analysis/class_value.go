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

package analysis

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/gapid/gapil/semantic"
)

// Value interface compliance checks.
var (
	_ = Value(&ClassValue{})
	_ = fieldHolder(&ClassValue{})
)

// ClassValue is an implementation of Value that represents all the possible
// values of a class type.
type ClassValue struct {
	// The class type.
	Class *semantic.Class

	// A map of field name to possible values for that field.
	Fields map[string]Value
}

// Print returns a textual representation of the value.
func (v *ClassValue) Print(results *Results) string {
	parts := make([]string, 0, len(v.Fields))
	for n, f := range v.Fields {
		value := strings.Replace(f.Print(results), "\n", "\n  ", -1)
		parts = append(parts, fmt.Sprintf("  %v: %v", n, value))
	}
	sort.Strings(parts)
	return fmt.Sprintf("%v{\n%v\n}", v.Class.Name(), strings.Join(parts, "\n"))
}

// Type returns the semantic class type of the value.
func (v *ClassValue) Type() semantic.Type {
	return v.Class
}

// Equivalent returns true iff v and o are equivalent.
// Unlike Equals() which returns the possibility of two values being equal,
// Equivalent() returns true iff the set of possible field values are exactly
// equal.
// o must be of type *ClassValue.
func (v *ClassValue) Equivalent(o Value) bool {
	if v == o {
		return true
	}
	a, b := v, o.(*ClassValue)
	for n, f := range a.Fields {
		if !f.Equivalent(b.Fields[n]) {
			return false
		}
	}
	return true
}

// Equals returns the possibility of all fields of v being equal to those in o.
// o must be of type *ClassValue.
func (v *ClassValue) Equals(o Value) Possibility {
	if v == o && v.Valid() {
		return True
	}
	a, b := v, o.(*ClassValue)
	result := True
	for n, f := range a.Fields {
		switch f.Equals(b.Fields[n]) {
		case False:
			return False
		case Maybe:
			result = Maybe
		}
	}
	return result
}

// Valid returns true if there is any possibility of this value equaling
// any other.
func (v *ClassValue) Valid() bool {
	for _, f := range v.Fields {
		if !f.Valid() {
			return false
		}
	}
	return true
}

// Union (∪) returns the class value with field values that are found in v or
// o.
// o must be of type *ClassValue.
func (v *ClassValue) Union(o Value) Value {
	a, b := v, o.(*ClassValue)
	if a == b {
		return a
	}
	out := &ClassValue{Class: v.Class, Fields: make(map[string]Value, len(v.Fields))}
	for n, f := range a.Fields {
		out.Fields[n] = f.Union(b.Fields[n])
	}
	return out
}

// Intersect (∩) returns the class value with field values that are found in v
// and o.
// o must be of type *ClassValue.
func (v *ClassValue) Intersect(o Value) Value {
	a, b := v, o.(*ClassValue)
	if a == b {
		return a
	}
	out := &ClassValue{Class: v.Class, Fields: make(map[string]Value, len(v.Fields))}
	for n, f := range a.Fields {
		out.Fields[n] = f.Intersect(b.Fields[n])
	}
	return out
}

// Difference (\) returns the class value with field values that are found in v
// but not found in o.
// o must be of type *ClassValue.
func (v *ClassValue) Difference(o Value) Value {
	a, b := v, o.(*ClassValue)
	out := &ClassValue{Class: v.Class, Fields: make(map[string]Value, len(v.Fields))}
	for n, f := range a.Fields {
		out.Fields[n] = f.Difference(b.Fields[n])
	}
	return out
}

// Clone returns a copy of v with a unique pointer.
func (v *ClassValue) Clone() Value {
	out := &ClassValue{Class: v.Class, Fields: make(map[string]Value, len(v.Fields))}
	for n, f := range v.Fields {
		out.Fields[n] = f
	}
	return out
}

// field returns the value of the field with the specified name.
func (v *ClassValue) field(s *scope, name string) Value {
	field, ok := v.Fields[name]
	if !ok {
		panic(fmt.Errorf("ClassValue %v did not contain field %v", v.Type().Name(), name))
	}
	return field
}

// setField returns the a new class value with the field with the specified name
// set to val.
func (v *ClassValue) setField(s *scope, name string, val Value) Value {
	out := v.Clone().(*ClassValue)
	out.Fields[name] = val
	return out
}
